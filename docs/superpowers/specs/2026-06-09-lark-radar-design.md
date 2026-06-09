# Lark Radar 设计文档

> 在 WeChat Radar 中新增飞书数据源支持，实现"双版本"运行：用户可在 setup 选择微信（wx-cli）或飞书（lark-cli）作为消息来源。

## 背景与目标

WeChat Radar 当前通过 `wx-cli` 读取本机微信客户端数据，实现本地优先的群聊情报看板。飞书没有类似 `wx-cli` 的本地数据库读取工具，但可以通过 `lark-cli` 调用飞书开放平台 API 拉取用户加入的群聊消息。

**目标**：在不破坏现有微信功能的前提下，新增"飞书模式"，让用户能把飞书群消息聚合到同一个本地 SQLite 看板中。

## 核心设计原则

1. **浅适配，不破坏现有功能**：新增独立的飞书数据摄取层，不重构现有微信逻辑。
2. **本地优先**：消息数据仍然落到本地 `~/.wechat-radar/radar.db`。
3. **MVP 范围**：仅支持文本类消息（`text`、`post`、`card`），图片/文件/视频等显示占位符。
4. **混合调度**：页面打开时自动增量同步，同时提供 CLI 脚本可挂系统 cron。

## 架构

```
现有层（基本不改）：
  components/                看板组件
  app/                       页面路由
  lib/db.ts                  SQLite schema（小扩展）
  lib/messages-store.ts      消息写入与链接提取（复用）
  lib/topics.ts              话题聚合
  lib/link-intelligence.ts   链接情报
  lib/mentions.ts            @ 我的消息
  lib/dashboard-intelligence.ts  看板情报

新增层：
  lib/lark.ts                调用 lark-cli 的封装
  lib/lark-sync.ts           增量同步逻辑
  app/api/lark/sync/route.ts 手动/自动同步 API
  app/api/lark/chats/route.ts 获取飞书群列表
  scripts/lark_sync.mjs      可挂 cron 的 CLI 脚本
```

## 数据映射

飞书消息格式映射到现有 `messages` 表字段：

| 飞书字段 | 本地字段 | 说明 |
|---|---|---|
| `chat_id` (`oc_xxx`) | `chatroom_id` | 群唯一标识 |
| `message_id` (`om_xxx`) | `local_id` | 消息唯一标识 |
| `sender.id` (`ou_xxx`) | `sender` | 发送者 ID |
| `sender.name` | `sender_name` | 发送者显示名（新增字段） |
| `create_time` (毫秒时间戳) | `timestamp` / `time` / `date` | 本地统一用秒 + 日期字符串 |
| `msg_type` | `type` | `text` / `post` / `card` / `image` / `file` / ... |
| 提取后的纯文本 | `content` | 用于搜索、话题聚合、@检测 |
| 原始 JSON（可选） | `raw` | 用于调试和后续扩展 |

### 消息类型处理

| 飞书 msg_type | 处理方式 |
|---|---|
| `text` | 直接取 `content.text` |
| `post` | 递归遍历 post 内容，提取所有 `text` 节点 |
| `card` | 提取 card 中的 `header.title` 和元素中的 `text` 字段，生成摘要 |
| `image` / `file` / `media` | `content` 存占位符 `[图片]` / `[文件]` / `[视频]` |
| `sticker` / `emoji` | 占位符 `[表情]` |
| 其他 | 占位符 `[未知消息]` |

## 同步策略

### 首次同步

1. 调用 `lark-cli im +chat-list --as user` 列出用户加入的所有群。
2. 对每个群调用 `lark-cli im +chat-messages-list --chat-id <id> --as user --page-size 50`。
3. 分页拉取最近 **200 条** 消息（可配置）。
4. 映射后调用 `bulkInsertMessages()` 写入 `messages` 表。
5. 更新 `sync_state` 表，记录每个群的 `last_sync_time`。

### 增量同步

1. 从 `sync_state` 读取每个飞书群的 `last_sync_time`。
2. 调用 `+chat-messages-list` 时传入 `--start <ISO8601>`，只拉取新消息。
3. 用 `message_id` 去重：`INSERT OR IGNORE`。
4. 更新 `last_sync_time` 为本次同步的最大消息时间。

### 失败处理

- 单群同步失败记录 `last_error`，不影响其他群。
- 网络错误 / rate limit 时指数退避重试（最多 3 次）。
- lark-cli 未登录 / 权限不足时，API 返回明确错误，前端提示用户重新授权。

## 数据库变更

### `messages` 表

新增可选字段：

```sql
ALTER TABLE messages ADD COLUMN sender_name TEXT;
ALTER TABLE messages ADD COLUMN raw TEXT;          -- 原始飞书消息 JSON
ALTER TABLE messages ADD COLUMN source TEXT DEFAULT 'wechat'; -- 'wechat' | 'lark'
```

> 注意：`source` 字段用于区分数据来源，便于多源共存时过滤和调试。

### `sync_state` 表

扩展现有表：

```sql
ALTER TABLE sync_state ADD COLUMN source TEXT DEFAULT 'wechat';
ALTER TABLE sync_state ADD COLUMN last_sync_time TEXT; -- ISO8601，飞书用
```

主键变为复合主键：`(chatroom_id, source)`。

### `groups` 表

现有 `groups` 表用于用户自定义分类和收藏。飞书群也需要能收藏和分类。

方案：**复用现有 `groups` 表**，`id` 字段直接存 `oc_xxx`，`source` 字段区分 `wechat` / `lark`。

```sql
ALTER TABLE groups ADD COLUMN source TEXT DEFAULT 'wechat';
```

## API 设计

### `POST /api/lark/sync`

触发飞书消息同步。

请求体：

```json
{
  "chat_id": "oc_xxx" // 可选，不传则同步所有群
}
```

响应：

```json
{
  "ok": true,
  "synced": {
    "oc_xxx": { "inserted": 12, "skipped": 0, "error": null },
    "oc_yyy": { "inserted": 0, "skipped": 5, "error": null }
  }
}
```

### `GET /api/lark/chats`

获取飞书群列表（用于前端展示和选择）。

响应：

```json
{
  "ok": true,
  "chats": [
    { "id": "oc_xxx", "name": "产品技术群", "member_count": 120 }
  ]
}
```

## Setup 页面改动

在 `/setup` 增加"数据源"选择步骤：

1. **选择数据源**
   - 微信（默认）
   - 飞书
   - Demo 数据

2. **如果选择飞书**
   - 检测 `lark-cli` 是否已安装
   - 检测是否已配置应用：`lark-cli doctor`
   - 检测是否已用户授权：`lark-cli auth login --as user`
   - 引导用户输入显示名（用于 @ 检测）
   - 写入配置：`{ source: 'lark', setupCompleted: true, ... }`

3. **隐私确认**（复用现有）

## 自动同步机制

### 前端触发

首页加载时，如果 `config.source === 'lark'`，自动调用 `POST /api/lark/sync`（带防抖，避免频繁刷新）。

### CLI 脚本

新增 `scripts/lark_sync.mjs`：

```bash
node scripts/lark_sync.mjs
```

可配置环境变量：

```bash
WECHAT_RADAR_SOURCE=lark
WECHAT_RADAR_LARK_DAYS_BACK=7
```

用户可以把它挂到 `crontab`：

```cron
*/5 * * * * cd /path/to/wechat-radar && node scripts/lark_sync.mjs >> ~/.wechat-radar/lark_sync.log 2>&1
```

## 依赖与前置条件

- 已安装 `lark-cli`
- 已完成 `lark-cli config init`
- 已完成 `lark-cli auth login --as user`
- 飞书应用已开通必要 scope：
  - `im:chat:readonly`（读取群列表）
  - `im:message:readonly`（读取群消息）

## 风险与限制

1. **飞书 API 有 rate limit**：需要控制并发，建议每次同步最多 3 个群并发。
2. **历史消息范围**：飞书 API 能拉到的历史消息范围受限于租户策略，不一定能拿到全部历史。
3. **消息类型有限**：MVP 不支持图片、文件、语音的内容分析。
4. **权限依赖**：如果用户授权过期，需要重新 `auth login`。

## 群聊黑白名单

用户可能加入大量飞书群，不希望全部同步。支持在配置中指定黑白名单。

### 配置字段

```json
{
  "larkChatFilter": {
    "mode": "all",      // "all" | "allowlist" | "blocklist"
    "allowlist": [],    // 允许的 chat_id 列表
    "blocklist": []     // 禁止的 chat_id 列表
  }
}
```

### 过滤规则

- `all`：同步所有群（默认）
- `allowlist`：只同步 `allowlist` 中的群
- `blocklist`：同步除 `blocklist` 外的所有群

### 前端管理

在 `/setup` 或新增 `/settings/lark` 页面提供群列表管理：
- 调用 `GET /api/lark/chats` 展示所有飞书群
- 支持搜索、批量选择加入 allowlist/blocklist
- 实时保存到 `~/.wechat-radar/config.json`

### 同步时应用

`lib/lark-sync.ts` 在调用 `+chat-list` 后，根据 `larkChatFilter` 过滤群列表，只对符合条件的群拉取消息。

## 后续可扩展

- 支持图片 OCR 文字提取
- 支持飞书机器人模式（实时事件推送）
- 支持飞书妙记（Minutes）内容分析
- 支持多源并存（同时采集微信 + 飞书）
