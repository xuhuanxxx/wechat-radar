# Security

## Reporting

Please open a GitHub security advisory or a private issue if you find a vulnerability.

This project is for learning, research, personal evaluation, and non-commercial experimentation only. Commercial profit or commercial advantage is prohibited. See [DISCLAIMER.md](DISCLAIMER.md) and [LICENSE](LICENSE).

## Local data

The most sensitive asset is your local SQLite database. Keep it outside synced folders and do not publish it. The default path is `~/.lark-radar/radar.db`.

## Lark usage boundary

- 当前只建议读取历史聊天记录。
- 不要自动发消息、加好友、改资料或做任何写入/社交操作。
- 本项目与飞书（Lark）、字节跳动均无官方关联，未获得飞书或字节跳动授权、认可、赞助或背书。
- 用户自行承担账号、数据、合规和平台风险。

## No warranty

The software is provided as-is. The authors and contributors do not guarantee availability, accuracy, security, legal compliance, account safety, or compatibility with any Lark, lark-cli, Node.js, macOS, or third-party service version.

## Command execution

The app invokes `lark-cli` via `child_process.execFile` with argument arrays. Avoid changing this to shell string execution.
