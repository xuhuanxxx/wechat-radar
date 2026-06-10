# Lark Radar

> 群太多，真正有价值的消息却总是被淹没。
> Lark Radar turns noisy Lark groups into a local-first intelligence dashboard.

[![GitHub stars](https://img.shields.io/github/stars/joeseesun/wechat-radar?style=social)](https://github.com/joeseesun/wechat-radar/stargazers)
[![GitHub forks](https://img.shields.io/github/forks/joeseesun/wechat-radar?style=social)](https://github.com/joeseesun/wechat-radar/network/members)
[![Issues](https://img.shields.io/github/issues/joeseesun/wechat-radar)](https://github.com/joeseesun/wechat-radar/issues)
[![Last commit](https://img.shields.io/github/last-commit/joeseesun/wechat-radar)](https://github.com/joeseesun/wechat-radar/commits/main)
[![License: Non-commercial research](https://img.shields.io/badge/License-Non--commercial%20research-orange.svg)](LICENSE)

![Lark Radar product preview](docs/assets/product-preview.svg)

**[中文](#中文) | [English](#english)**

> 重要：本项目仅供学习研究和个人非商业实验使用，禁止用于商业盈利。请先阅读 [免责声明](DISCLAIMER.md) 和 [许可证](LICENSE)。

---

<a name="中文"></a>

## 中文

Lark Radar 是一个本地优先的飞书群聊情报看板。它把群消息、话题、链接、@我的消息和高信号人物聚合成一个可按日期查看的工作台。

你得到的不是"聊天记录列表"，而是每天可以直接处理的情报：

- 今日优先看：消息、文章、工具、异动分区展示
- 话题雷达：用 Codex CLI 按天聚合跨群话题
- 链接情报：文章/工具资源去重，生成可读标题
- 群日报：每天活跃群可生成摘要报告，方便复制给 AI 继续处理
- 本地存储：聊天数据落到你自己的 SQLite，不上传到第三方服务
- 明暗主题：默认奶白色浅色主题，也支持深色模式

## 快速开始

```bash
git clone https://github.com/joeseesun/wechat-radar.git
cd wechat-radar
pnpm install
pnpm rebuild better-sqlite3
pnpm dev
```

打开 [http://localhost:3000](http://localhost:3000)。首次进入会跳到 `/setup`，按页面提示填写你的飞书名、确认隐私说明，也可以先启用 demo 数据体验。

## 前置条件

- [ ] macOS，且已安装飞书客户端
- [ ] Node.js 20+：`node --version`
- [ ] pnpm：`corepack enable && pnpm --version`
- [ ] lark-cli：`lark-cli --version`
- [ ] lark-cli 已登录：`lark-cli auth login --as user`
- [ ] 如果要让话题聚合更好，安装并登录 Codex CLI：`codex --version`

lark-cli 可参考 [lark-cli 官方文档](https://open.larksuite.com/document/home/index)安装与初始化。

## 配置

默认数据目录是 `~/.lark-radar/`，不会写进项目目录。

你可以用环境变量覆盖：

```bash
cp .env.example .env.local
```

常用配置：

```bash
LARK_RADAR_DATA_DIR=~/.lark-radar
LARK_RADAR_MY_NAMES=张三,San Zhang,zhangsan
LARK_RADAR_DEMO=0
LARK_RADAR_CODEX_MODEL=
```

也可以直接在 `/setup` 页面配置。配置会写入 `~/.lark-radar/config.json`。

## 使用方式

1. 进入首页，选择日期或时间范围。
2. 点击"同步"同步当前范围消息。
3. 点击"全量同步"拉取更长历史。
4. 打开"话题雷达"查看跨群主题。
5. 打开"链接情报"查看文章和工具资源。
6. 在活跃群列表点击"日报"查看单群日报。

你可以这样和 AI 配合：

- "把今天所有 Codex 相关话题整理成一篇博客大纲。"
- "复制这个群日报，帮我提炼值得回复的机会。"
- "把链接情报里的工具做成一张试用优先级表。"

## 数据与隐私

Lark Radar 默认只在本机读写数据：

- `~/.lark-radar/radar.db`：SQLite 主数据库
- `~/.lark-radar/config.json`：本地配置
- `~/.lark-radar/backups/`：可选备份

安全设计：

- lark-cli 调用使用 `child_process.execFile` 参数数组，不拼 shell
- SQLite 使用 prepared statements
- 页面只以 React 文本节点渲染聊天内容
- 不把飞书密钥、会话、数据库、模型缓存提交进仓库

重要风险提示：

- 当前只建议读取历史聊天记录，用于本地检索、聚合和摘要。
- 不要自动发消息、加好友、改资料或做任何写入/社交操作。
- 请确认你的使用方式符合飞书开放平台规则、当地法律、群成员隐私预期和你所在组织的合规要求。
- 不要把包含真实聊天内容的数据库或截图上传到公开仓库。

## 免责声明与禁止事项

本项目仅供学习研究、个人评估和非商业实验使用。禁止用于商业盈利，包括但不限于 SaaS、托管服务、付费报表、咨询交付、商业监控、线索挖掘、数据售卖、企业内部生产系统或任何直接/间接商业收益场景。

本项目与飞书（Lark）、字节跳动均无官方关联，未获得飞书或字节跳动授权、认可、赞助或背书。飞书、Lark、字节跳动等名称和商标归其权利人所有。

用户需要自行承担安装、配置、运行、修改、部署、数据处理和传播行为的全部责任。因使用或误用本项目导致的账号限制、数据丢失、隐私泄露、商业损失、法律纠纷、平台处罚、服务中断、误判摘要或其它后果，项目作者和贡献者不承担责任。

完整条款见 [DISCLAIMER.md](DISCLAIMER.md) 和 [LICENSE](LICENSE)。

## 项目结构

```text
apps/
  web/               Next.js 16 前端（@lark-radar/web）
    app/             App Router 页面与 catch-all API 代理
    components/      看板、侧边栏、图表、消息渲染组件
    lib/             api-client、日期工具等纯前端工具
    Dockerfile       Web 镜像构建（构建上下文 = 仓库根）
  data-service/      Go 数据服务（lark-cli + SQLite，仅 macOS）
    main.go          HTTP 路由表
    handlers/        每个 API 端点的实现
    models/          数据模型
    sync/            lark-cli 调用与同步引擎
  macos-menu/        Swift 菜单栏，监管 data-service 进程
packages/            预留给跨语言契约包（OpenAPI 等）
docs/                架构与开发文档
scripts/             发布脚本（build-macos-app.sh 等）
```

`pnpm-workspace.yaml` 在仓库根。所有日常脚本（`pnpm dev` / `build` / `lint`）也都在根目录运行，会通过 `pnpm --filter @lark-radar/web` 转发到对应子包。

## 常见问题

| 问题 | 解决方法 |
| --- | --- |
| `lark-cli 未登录` | 先运行 `lark-cli auth login --as user`，再刷新页面。 |
| `better-sqlite3` native 模块报错 | 运行 `pnpm rebuild better-sqlite3`。 |
| 首页没有数据 | 先完成 `/setup`，确认 `lark-cli doctor` 有输出，然后点击"同步"。 |
| 话题雷达为空 | 打开对应日期会自动构建；也可以点击"构建话题"。需要本机可运行 `codex`。 |
| 不想读取真实飞书 | 在 `/setup` 勾选 demo 模式，或设置 `LARK_RADAR_DEMO=1`。 |

## 致谢

- [飞书开放平台](https://open.larksuite.com/)：本项目通过 lark-cli 读取飞书数据。
- [Next.js](https://nextjs.org/)、[ECharts](https://echarts.apache.org/)、[better-sqlite3](https://github.com/WiseLibs/better-sqlite3)。

---

<a name="english"></a>

## English

Lark Radar is a local-first intelligence dashboard for Lark groups. It turns noisy group chats into daily briefings, cross-group topics, link intelligence, mentions, and per-group reports.

### Features

- Daily dashboard for messages, links, tools, anomalies, and people
- Codex CLI powered topic clustering by date
- Link intelligence with generated titles and deduplication
- Per-group daily reports with copy-friendly output
- Local SQLite storage by default
- Light and dark themes

### Install

```bash
git clone https://github.com/joeseesun/wechat-radar.git
cd wechat-radar
pnpm install
pnpm rebuild better-sqlite3
pnpm dev
```

Open [http://localhost:3000](http://localhost:3000). The first run redirects to `/setup`, where you can configure your Lark display names and privacy confirmation, or enable demo mode.

### Requirements

- [ ] macOS with Lark client installed
- [ ] Node.js 20+
- [ ] pnpm
- [ ] lark-cli installed and authenticated
- [ ] Optional: Codex CLI for better topic/link summaries

### Privacy

By default, runtime data is stored locally under `~/.lark-radar/`. The app does not upload your chat database. You are responsible for using it in a way that respects Lark platform rules, local laws, group privacy expectations, and organizational compliance.

Safety guidance:

- Use this project for read-only historical chat access.
- Do not automate sending messages, adding friends, profile changes, or any other write/social action.
- The tested approach relies on lark-cli; platform changes may affect functionality.

### Troubleshooting

| Problem | Fix |
| --- | --- |
| lark-cli not authenticated | Run `lark-cli auth login --as user`. |
| better-sqlite3 fails to load | Run `pnpm rebuild better-sqlite3`. |
| No dashboard data | Finish `/setup`, confirm `lark-cli doctor` works, then click sync. |
| Topic radar is empty | Open the date or click build topics; make sure `codex` is available. |

## License

Non-commercial research use only. See [LICENSE](LICENSE) and [DISCLAIMER.md](DISCLAIMER.md).
