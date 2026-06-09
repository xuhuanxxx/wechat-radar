# Lark Radar 数据服务 API 规范

> 版本: v2.0
> 基础路径: 用户 Mac 上的 HTTP 服务（默认 `http://localhost:3456`）
> 格式: JSON
> CORS: 允许所有来源
> 协议: HTTP/1.1 + WebSocket

## 设计原则

- 数据服务只暴露 API，不感知调用方是谁
- Web 服务通过配置连接到数据服务
- 支持 HTTP REST API 和 WebSocket 实时推送

## 通用约定

### 请求格式

- `Content-Type: application/json`
- 查询参数编码为 URL query string
- POST body 为 JSON object

### 响应格式

```typescript
interface ApiResponse<T = unknown> {
  ok: boolean;
  error?: string;
  // 业务数据...
}
```

### 错误码

| HTTP 状态码 | 含义 |
|-------------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 404 | 资源不存在 |
| 429 | 请求过于频繁 |
| 500 | 服务器内部错误 |
| 503 | 数据层服务不可用 |

---

## 1. 健康检查

### GET /health

**响应:**
```json
{
  "ok": true,
  "version": "1.0.0",
  "uptime": 3600,
  "cache": { "keys": 42, "hits": 1200, "misses": 80 },
  "db": { "connected": true, "path": "/Users/xxx/.lark-radar/radar.db" }
}
```

---

## 2. 设置

### GET /api/setup

获取配置状态和系统检查。

**响应:**
```json
{
  "ok": true,
  "dataDir": "/Users/xxx/.lark-radar",
  "configPath": "/Users/xxx/.lark-radar/config.json",
  "configured": true,
  "config": {
    "myNicknames": ["张三"],
    "defaultRange": "week",
    "source": "lark",
    "port": 3456,
    "autoSyncInterval": 0
  },
  "checks": {
    "larkInstalled": true,
    "larkAuthenticated": true,
    "larkError": null
  }
}
```

### POST /api/setup

初始化配置。

**请求体:**
```json
{
  "myNicknames": ["张三"],
  "privacyConfirmed": true,
  "demoMode": false,
  "defaultSyncDays": 7,
  "source": "lark",
  "larkChatFilter": {
    "mode": "all",
    "allowlist": [],
    "blocklist": []
  },
  "port": 3456,
  "autoSyncInterval": 0
}
```

**响应:**
```json
{
  "ok": true,
  "configured": true,
  "config": { ... }
}
```

---

## 3. 统计

### GET /api/stats

获取 Dashboard 统计数据。

**查询参数:**
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| range | string | 否 | "week" | day/week/month/quarter/year |
| date | string | 否 | 今天 | 锚定日期 YYYY-MM-DD |

**响应:**
```json
{
  "ok": true,
  "range": "week",
  "window": { "since": "2026-06-03", "until": "2026-06-09", "days": 7 },
  "cards": {
    "active_groups": 12,
    "total_groups": 15,
    "total_messages": 3420,
    "mentions": 8,
    "silent_groups": 3,
    "avg_per_group": 228
  },
  "trend": {
    "data": [{ "date": "2026-06-03", "count": 480 }, ...],
    "peak": { "date": "2026-06-07", "count": 620 },
    "avg": 488.5,
    "total": 3420
  },
  "active_groups": [
    {
      "chatroom_id": "oc_xxx",
      "name": "AI产品蝗虫团",
      "total": 520,
      "top_senders": [{ "sender": "张三", "count": 45 }]
    }
  ],
  "categories": [
    {
      "id": 1,
      "name": "AI产品蝗虫团",
      "color": "#ef4444",
      "emoji": "🐝",
      "group_count": 3,
      "message_count": 1200
    }
  ],
  "intelligence": {
    "date": "2026-06-09",
    "must_read": [...],
    "opportunities": [...],
    "signal_sources": [...],
    "action_items": [...],
    "topic_lifecycle": [...],
    "link_highlights": [...],
    "people_radar": [...],
    "content_ideas": [...],
    "anomalies": [...]
  },
  "sidebar_counts": {
    "all": 15,
    "favorites": 3,
    "unsorted": 2
  }
}
```

---

## 4. 会话（群列表）

### GET /api/sessions

获取所有群会话列表。

**响应:**
```json
{
  "ok": true,
  "total": 15,
  "groups": [
    {
      "chatroom_id": "oc_xxx",
      "name": "AI产品蝗虫团",
      "last_msg_type": "text",
      "last_sender": "张三",
      "summary": "今天讨论了新产品功能...",
      "time": "2026-06-09 14:30:00",
      "timestamp": 1717921800,
      "unread": 0,
      "is_favorite": true,
      "group_ids": [1, 3]
    }
  ],
  "categories": [
    {
      "id": 1,
      "name": "AI产品蝗虫团",
      "color": "#ef4444",
      "emoji": "🐝",
      "member_count": 3
    }
  ]
}
```

---

## 5. 群详情

### GET /api/group/:chatroom_id

获取指定群的详细消息和统计。

**查询参数:**
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| date | string | 否 | 今天 | 查看日期 YYYY-MM-DD |
| limit | number | 否 | 1000 | 消息条数上限 |

**响应:**
```json
{
  "ok": true,
  "chatroom_id": "oc_xxx",
  "date": "2026-06-09",
  "stats": {
    "chat": "AI产品蝗虫团",
    "total": 156,
    "by_hour": [{ "hour": 0, "count": 2 }, ...],
    "by_type": [{ "type": "text", "count": 120 }, ...],
    "top_senders": [{ "sender": "张三", "count": 30 }, ...]
  },
  "recent": [
    {
      "local_id": "om_xxx",
      "sender": "张三",
      "content": "今天的产品会议确定了新功能...",
      "time": "2026-06-09 14:30:00",
      "timestamp": 1717921800,
      "type": "text"
    }
  ],
  "daily_history": [
    { "date": "2026-06-01", "total": 120 },
    { "date": "2026-06-02", "total": 200 }
  ],
  "sync_state": {
    "chatroom_id": "oc_xxx",
    "source": "lark",
    "last_synced_at": 1717921800000,
    "total_messages": 5000,
    "status": "ok"
  },
  "synced_dates": ["2026-06-01", "2026-06-02", "2026-06-09"]
}
```

---

## 6. 同步

### POST /api/lark/sync

触发飞书消息同步。

**请求体:**
```json
{
  "chat_id": "oc_xxx",      // 可选，指定单个群
  "days_back": 7,            // 可选，回溯天数
  "stream": false            // 可选，是否流式响应
}
```

**非流式响应:**
```json
{
  "ok": true,
  "synced": {
    "oc_xxx": {
      "inserted": 45,
      "skipped": 120,
      "error": null
    }
  }
}
```

**流式响应 (SSE):**
```
data: {"type":"start"}

data: {"type":"progress","chatId":"oc_xxx","phase":"fetch","count":0}

data: {"type":"progress","chatId":"oc_xxx","phase":"persist","count":45}

data: {"type":"finished","ok":true,"synced":{"oc_xxx":{"inserted":45,"skipped":120,"error":null}}}
```

---

## 7. @提及

### GET /api/mentions

获取 @ 我的消息列表。

**查询参数:**
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| limit | number | 否 | 1000 | 返回条数上限 |

**响应:**
```json
{
  "ok": true,
  "total": 8,
  "items": [
    {
      "chatroom_id": "oc_xxx",
      "local_id": "om_xxx",
      "sender": "李四",
      "content": "@张三 帮忙看一下这个方案",
      "time": "2026-06-09 10:00:00",
      "timestamp": 1717908000,
      "seen": 0,
      "chat_name": "AI产品蝗虫团"
    }
  ]
}
```

### POST /api/mentions

标记提及为已读。

**请求体:**
```json
{
  "chatroom_id": "oc_xxx"  // 可选，不传则标记全部
}
```

**响应:**
```json
{ "ok": true }
```

---

## 8. 话题

### GET /api/topics

获取指定日期的话题列表。

**查询参数:**
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| date | string | 否 | 今天 | YYYY-MM-DD |

**响应:**
```json
{
  "ok": true,
  "date": "2026-06-09",
  "topics": [
    {
      "id": 1,
      "date": "2026-06-09",
      "title": "Claude Code 工作流讨论",
      "summary": "多个群讨论了 Claude Code 的使用体验...",
      "message_count": 12,
      "group_count": 3
    }
  ]
}
```

### GET /api/topics/:id

获取话题详情。

**响应:**
```json
{
  "ok": true,
  "id": 1,
  "date": "2026-06-09",
  "title": "Claude Code 工作流讨论",
  "summary": "多个群讨论了 Claude Code 的使用体验...",
  "message_count": 12,
  "group_count": 3,
  "messages": [
    {
      "chatroom_id": "oc_xxx",
      "chat_name": "AI产品蝗虫团",
      "local_id": "om_xxx",
      "sender": "张三",
      "content": "Claude Code 的 CLI 体验很好...",
      "time": "2026-06-09 10:00:00",
      "timestamp": 1717908000,
      "type": "text",
      "score": 1.0
    }
  ]
}
```

### POST /api/topics/build

构建指定日期的话题（SSE 流式）。

**请求体:**
```json
{ "date": "2026-06-09" }
```

**流式响应:**
```
data: {"type":"start"}

data: {"type":"load","message":"加载当日消息..."}

data: {"type":"llm","done":1,"total":3,"message":"Codex 聚合 1/3"}

data: {"type":"save","done":1,"total":5,"message":"Claude Code 工作流讨论"}

data: {"type":"done","count":5}
```

---

## 9. 链接情报

### GET /api/topics/links

获取指定日期的链接情报。

**查询参数:**
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| date | string | 否 | 今天 | YYYY-MM-DD |
| refresh | boolean | 否 | false | 强制刷新缓存 |

**响应:**
```json
{
  "ok": true,
  "date": "2026-06-09",
  "articles": [
    {
      "kind": "article",
      "url": "https://mp.weixin.qq.com/s/xxx",
      "canonical_url": "https://mp.weixin.qq.com/s/xxx",
      "title": "AI 产品年度复盘",
      "domain": "mp.weixin.qq.com",
      "count": 5,
      "group_count": 3,
      "first_seen": "2026-06-09 10:00:00",
      "last_seen": "2026-06-09 15:00:00",
      "sources": [...]
    }
  ],
  "tools": [...]
}
```

---

## 10. 搜索

### GET /api/search

全局搜索。

**查询参数:**
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| q | string | 是 | - | 搜索关键词（最少 2 字符） |

**响应:**
```json
{
  "ok": true,
  "results": [
    {
      "id": "group:oc_xxx",
      "type": "group",
      "title": "AI产品蝗虫团",
      "subtitle": "oc_xxx",
      "href": "/groups/oc_xxx"
    },
    {
      "id": "message:oc_xxx:2026-06-09 10:00:00:张三",
      "type": "message",
      "title": "今天的产品会议确定了新功能...",
      "subtitle": "AI产品蝗虫团 · 张三 · 2026-06-09 10:00:00",
      "href": "/groups/oc_xxx?date=2026-06-09"
    }
  ]
}
```

---

## 11. 分组管理

### GET /api/groups

获取所有分组。

**响应:**
```json
{
  "ok": true,
  "groups": [
    {
      "id": 1,
      "name": "AI产品蝗虫团",
      "color": "#ef4444",
      "emoji": "🐝",
      "sort_order": 0,
      "member_count": 3
    }
  ]
}
```

### POST /api/groups

创建分组。

**请求体:**
```json
{
  "name": "新分组",
  "color": "#3b82f6",
  "emoji": "🎯"
}
```

**响应:**
```json
{ "ok": true, "id": 16 }
```

### DELETE /api/groups

删除分组。

**请求体:**
```json
{ "id": 16 }
```

**响应:**
```json
{ "ok": true }
```

---

## 12. 群标签

### GET /api/group-tags

获取群的标签。

**查询参数:**
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| chatroom_id | string | 是 | 群 ID |

**响应:**
```json
{
  "ok": true,
  "group_ids": [1, 3]
}
```

### POST /api/group-tags

更新群的标签或收藏状态。

**请求体（添加/删除标签）:**
```json
{
  "chatroom_id": "oc_xxx",
  "group_id": 1,
  "action": "add"  // 或 "remove"
}
```

**请求体（收藏）:**
```json
{
  "chatroom_id": "oc_xxx",
  "fav": true
}
```

**响应:**
```json
{ "ok": true }
```

---

## 13. AI 分类

### GET /api/ai-classify

获取 AI 分类建议。

**响应:**
```json
{
  "ok": true,
  "groups": [...],
  "suggestions": [
    {
      "chatroom_id": "oc_xxx",
      "name": "新群",
      "summary": "...",
      "current_group_ids": [],
      "suggested_group_id": 1,
      "suggested_group_name": "AI产品蝗虫团",
      "reason": "匹配到关键词"
    }
  ]
}
```

### POST /api/ai-classify

应用分类建议。

**请求体:**
```json
{
  "picks": [
    { "chatroom_id": "oc_xxx", "group_id": 1 }
  ]
}
```

**响应:**
```json
{ "ok": true, "applied": 1 }
```

---

## 14. 飞书相关

### GET /api/lark/chats

获取飞书群列表。

**响应:**
```json
{
  "ok": true,
  "chats": [
    {
      "id": "oc_xxx",
      "name": "AI产品蝗虫团",
      "member_count": 150,
      "filtered": true
    }
  ],
  "filter": {
    "mode": "all",
    "allowlist": [],
    "blocklist": []
  }
}
```

### GET /api/lark/filter

获取群过滤配置。

**响应:**
```json
{
  "ok": true,
  "filter": {
    "mode": "all",
    "allowlist": [],
    "blocklist": []
  }
}
```

### POST /api/lark/filter

更新群过滤配置。

**请求体:**
```json
{
  "mode": "allowlist",
  "allowlist": ["oc_xxx"],
  "blocklist": []
}
```

**响应:**
```json
{ "ok": true, "filter": { ... } }
```

---

## 15. 日期列表

### GET /api/dates

获取有消息的所有日期。

**响应:**
```json
{
  "ok": true,
  "dates": [
    { "date": "2026-06-09", "count": 3420 },
    { "date": "2026-06-08", "count": 2800 }
  ]
}
```

---

## 16. 数据库信息

### GET /api/dbinfo

获取数据库统计信息。

**响应:**
```json
{
  "dataDir": "/Users/xxx/.lark-radar",
  "dbPath": "/Users/xxx/.lark-radar/radar.db",
  "dbSize": 10485760,
  "counts": {
    "groups": 14,
    "messages": 50000,
    "daily_stats": 90,
    "sync_state": 15
  },
  "topGroups": [
    { "chatroom_id": "oc_xxx", "n": 12000 }
  ]
}
```

---

## 17. 消息链接

### GET /api/message-links/raw

获取原始链接列表。

**查询参数:**
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| date | string | 否 | 今天 | YYYY-MM-DD |

**响应:**
```json
{
  "ok": true,
  "date": "2026-06-09",
  "links": [
    {
      "chatroom_id": "oc_xxx",
      "local_id": "om_xxx",
      "sender": "张三",
      "time": "2026-06-09 10:00:00",
      "url": "https://mp.weixin.qq.com/s/xxx",
      "canonical_url": "https://mp.weixin.qq.com/s/xxx",
      "title": null,
      "domain": "mp.weixin.qq.com",
      "source": "lark_raw",
      "raw_kind": "article",
      "chat_name": "AI产品蝗虫团"
    }
  ]
}
```

### POST /api/message-links/resolve

手动解析链接。

**请求体:**
```json
{
  "chatroom_id": "oc_xxx",
  "local_id": "om_xxx",
  "url": "https://example.com/article",
  "title": "文章标题",
  "description": "文章描述",
  "source": "manual",
  "confidence": 1.0
}
```

**响应:**
```json
{ "ok": true }
```

### POST /api/message-links/backfill

批量回填链接。

**请求体:**
```json
{
  "since": "2026-06-01",
  "until": "2026-06-09"
}
```

**响应:**
```json
{
  "ok": true,
  "processed": 100,
  "resolved": 45
}
```

---

## 18. WebSocket API

### WS /ws

实时推送连接，用于同步进度、新消息通知等。

**连接:**
```javascript
const ws = new WebSocket('ws://localhost:3456/ws');
```

**事件类型:**

#### sync-progress
同步进度推送（替代 SSE）。

```json
{
  "type": "sync-progress",
  "chatId": "oc_xxx",
  "phase": "persist",
  "count": 45
}
```

#### sync-finished
同步完成。

```json
{
  "type": "sync-finished",
  "ok": true,
  "synced": {
    "oc_xxx": { "inserted": 45, "skipped": 120, "error": null }
  }
}
```

#### new-messages
新消息到达通知。

```json
{
  "type": "new-messages",
  "chatroom_id": "oc_xxx",
  "count": 5,
  "timestamp": 1717921800
}
```

#### mention
@提及通知。

```json
{
  "type": "mention",
  "chatroom_id": "oc_xxx",
  "sender": "李四",
  "content": "@张三 帮忙看一下",
  "timestamp": 1717908000
}
```

#### ping/pong
心跳保活。

```json
// 服务端 → 客户端
{ "type": "ping", "timestamp": 1717921800 }

// 客户端 → 服务端
{ "type": "pong", "timestamp": 1717921800 }
```
