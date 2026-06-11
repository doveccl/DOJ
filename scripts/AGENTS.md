# Scripts AI 规则

## 范围

本文件约束 `scripts/`。这里维护检查脚本、开发辅助脚本和 smoke 验收。

## Check

- `bun run check` 是基础验收，必须覆盖 shared、db、agent、api、web。
- 脚本应使用 Bun 运行，不引入额外 Node-only 运行假设。
- 读取子包 dev 环境时注意 cwd；需要根 `.env` 的 dev 脚本必须显式传 `--env-file=../../.env`。

## Smoke

- `bun run smoke` 是最小端到端验收入口；`bun run smoke -- <name>` 运行单项。
- smoke name 与文件名保持一致。
- 当前最小目标：`auth`、`settings`、`members`、`problem-assets`、`judge-default`、`judge-strict`、`judge-custom`、`submission-security`、`realtime-progress`、`redis-derived`、`assignments`、`contests`、`limits-and-hash`、`discussion`。
- 评测类 smoke 必须通过 server + Agent 路径，不得绕过真实调度、WebSocket 或 Docker runner。
- `limits-and-hash` 必须覆盖固定资源限制、output limit、wall-clock cap 和 `bundleHash`。
- `redis-derived` 必须证明 Redis 派生统计可以从 PostgreSQL 修复。
- 权限类 smoke 必须覆盖隐藏题、源码裁剪、私有 assets、封榜/真实榜单裁剪。
