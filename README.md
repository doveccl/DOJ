# DOJ

DOJ is a new Bun-based rewrite of the online judge used by the older `v3` branch. The rewrite keeps Docker-based judging as the core execution model while moving the product stack to PostgreSQL, Redis/Valkey, Vue 3, Naive UI, and configurable judge languages/runners.

The `main` branch is still pre-release. Database migrations may be regenerated from a fresh baseline until the first stable release.

## Stack

- Runtime and tooling: Bun
- API: Hono
- Database: PostgreSQL + Drizzle
- Cache and sessions: Redis-compatible Valkey through Bun's native Redis client
- Frontend: Vue 3, Vite, Naive UI, vue-i18n
- Judging: Docker runner, moving toward separately deployed judge agents
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

Run the API, web app, and worker together:

```sh
bun run dev
```

Default local URLs:

- Web: `http://localhost:28080`
- API: `http://localhost:7974`
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

DOJ 是基于 Bun 的新版在线评测系统重写分支，目标是替代旧的 `v3` 实现。新版继续以 Docker 作为评测执行核心，同时迁移到 PostgreSQL、Redis/Valkey、Vue 3、Naive UI，并支持可配置的评测语言和评测机。

当前 `main` 分支仍处于正式发布前阶段。第一次稳定版发布前，数据库迁移可以按需要清空并重新生成基线。

## 技术栈

- 运行时与工具链：Bun
- API：Hono
- 数据库：PostgreSQL + Drizzle
- 缓存与会话：通过 Bun 原生 Redis 客户端连接 Redis 兼容的 Valkey
- 前端：Vue 3、Vite、Naive UI、vue-i18n
- 评测：Docker runner，并逐步迁移到独立部署的 judge agent
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

同时启动 API、前端和 worker：

```sh
bun run dev
```

默认本地地址：

- 前端：`http://localhost:28080`
- API：`http://localhost:7974`
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
