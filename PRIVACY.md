# Privacy

Lark Radar is designed as a local-first tool.

- Chat data is stored in a local SQLite database under `~/.lark-radar` by default.
- The app does not upload chat records to a hosted service.
- The app reads data through your local `lark-cli` installation.
- Do not commit `*.db`, `.env.local`, logs, or generated runtime data.
- If you enable optional LLM/Codex workflows, review what data those tools receive before using them.

You are responsible for complying with local law, platform terms, and group member expectations before reading, storing, or processing chat data.

This project is for learning, research, personal evaluation, and non-commercial experimentation only. Commercial profit, commercial monitoring, paid reporting, hosted services, consulting delivery, data resale, or other revenue-generating use is prohibited. See [DISCLAIMER.md](DISCLAIMER.md) and [LICENSE](LICENSE).

## Lark account safety

- 当前只建议读取历史聊天记录，用于本地检索、聚合和摘要。
- 不要自动发消息、加好友、改资料或做任何写入/社交操作。
- 请确认你的使用方式符合飞书开放平台规则、当地法律、组织制度和群成员隐私预期。
- 不要把包含真实聊天内容的数据库、截图或导出文件上传到公开仓库。

## Third-party and platform disclaimer

- 本项目与飞书（Lark）、字节跳动均无官方关联，未获得飞书或字节跳动授权、认可、赞助或背书。
- 你需要自行确认使用方式符合飞书开放平台规则、当地法律、组织制度和群成员隐私预期。
- 项目作者和贡献者不对账号限制、数据丢失、隐私泄露、法律纠纷、平台处罚或其它使用后果承担责任。
