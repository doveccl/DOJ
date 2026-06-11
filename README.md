# DOJ

DOJ is a new Bun-based rewrite of the online judge used by the older `v3` branch. The rewrite keeps Docker-based judging as the core execution model while moving the product stack to PostgreSQL, Redis/Valkey, S3-compatible object storage, Vue 3, Naive UI, configurable judge languages, and separately deployed judge agents.

The `main` branch is still pre-release. Database migrations may be regenerated from a fresh baseline until the first stable release.

## Stack

- Runtime and tooling: Bun
- Server: Hono HTTP API + WebSocket services
- Database: PostgreSQL + Drizzle
- Cache and sessions: Redis-compatible Valkey, with in-memory fallback for single-process development
- Frontend: Vue 3, Vite, Naive UI, vue-i18n
- Judging: server-side scheduling + WebSocket judge agents + agent-local Docker sandbox runner
- Object storage: S3-compatible storage via Bun's native `S3Client`

## Local Development

Start local infrastructure:

```sh
docker compose -f compose.dev.yml up -d
```

Install dependencies and copy the example environment:

```sh
bun install
cp .env.example .env
```

Prepare the database and object bucket:

```sh
bun run db:migrate
bun run s3:ensure-bucket
bun run db:seed
```

Run the server, web app, and local judge agent together:

```sh
bun run dev
```

Judge deployment model:

- The server handles API, judge scheduling, browser WebSocket, and agent WebSocket.
- `apps/agent` runs on each judge machine, keeps Docker local, and connects only to the server.
- The server reads problem assets from S3/MinIO, sends a unique problem hash first, and agents request the tgz judging bundle only when needed; agents do not need S3 credentials.
- The server and agents share `SECRET`; agents default to `SERVER=http://127.0.0.1:7974`, `AGENT_NAME=<hostname>`, `AGENT_CONCURRENCY=1`, and A image cache is controlled by `AGENT_CACHE_GB`.

Default local URLs:

- Web: `http://localhost:28080`
- Server: `http://localhost:7974`
- MinIO console: `http://localhost:9001`

## Useful Commands

```sh
bun run check
bun run lint
bun run format
bun run smoke
bun run smoke -- auth
bun run db:reset
```

`bun run smoke -- <name>` runs one smoke target. Run `bun run smoke -- unknown` to print the available target names.

## 中文说明

DOJ 是基于 Bun 的新版在线评测系统重写分支，目标是替代旧的 `v3` 实现。新版继续以 Docker 作为评测执行核心，同时迁移到 PostgreSQL、Redis/Valkey、S3 兼容对象存储、Vue 3、Naive UI，并支持可配置的评测语言和独立部署的评测 agent。

当前 `main` 分支仍处于正式发布前阶段。第一次稳定版发布前，数据库迁移可以按需要清空并重新生成基线。

## 技术栈

- 运行时与工具链：Bun
- Server：Hono HTTP API + WebSocket 服务
- 数据库：PostgreSQL + Drizzle
- 缓存与会话：Redis 兼容的 Valkey，单进程开发可使用内存 fallback
- 前端：Vue 3、Vite、Naive UI、vue-i18n
- 评测：server 调度 + WebSocket judge agent + agent 本机 Docker 沙箱 runner
- 对象存储：通过 Bun 原生 `S3Client` 访问 S3 兼容存储

## 本地开发

启动本地基础设施：

```sh
docker compose -f compose.dev.yml up -d
```

安装依赖并复制环境变量：

```sh
bun install
cp .env.example .env
```

初始化数据库和对象存储桶：

```sh
bun run db:migrate
bun run s3:ensure-bucket
bun run db:seed
```

同时启动 server、前端和本地 judge agent：

```sh
bun run dev
```

评测部署模型：

- server 同时负责 API、评测调度、浏览器 WebSocket 和 agent WebSocket。
- `apps/agent` 部署在每台评测机器上，Docker 始终走本机 socket，并且只连接 server。
- server 从 S3/MinIO 读取题目资产，先下发唯一题目 hash；agent 仅在需要时请求 tgz 评测文件包，不需要 S3 凭据。
- server 和 agent 共享 `SECRET`；agent 默认 `SERVER=http://127.0.0.1:7974`、`AGENT_NAME=<hostname>`、`AGENT_CONCURRENCY=1`，A 镜像缓存容量通过 `AGENT_CACHE_GB` 控制。

默认本地地址：

- 前端：`http://localhost:28080`
- Server：`http://localhost:7974`
- MinIO 控制台：`http://localhost:9001`

## 常用命令

```sh
bun run check
bun run lint
bun run format
bun run smoke
bun run smoke -- auth
bun run db:reset
```

`bun run smoke -- <名称>` 可以运行单个冒烟测试目标。传入未知名称时会打印可用目标列表。
