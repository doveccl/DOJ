# DOJ

DOJ is a self-hosted online judge with a React web app, a Go API server, and a Go judger/runner.

## Features

- Problems, submissions, contests, assignments, discussions, ranking, and user profiles.
- Markdown problem statements with image uploads and math rendering.
- ICPC and OI contest scoreboards, including ICPC freeze support.
- Go judger with Docker-based language containers and cgroup v2 resource measurement.
- PostgreSQL for data, Redis/Valkey for sessions and runtime state, and local or S3-compatible object storage.

## Quick Start

The compose file starts PostgreSQL, Valkey, the DOJ server, and a local judger. The server container serves both the API and the built web app:

```bash
docker compose up -d
```

Open `http://localhost:7974`. If the database has no administrator, DOJ creates the `admin` account with password `admin`. The web UI shows a prominent warning until that password is changed.

Uploaded files, problem packages, and backups are stored in `./storage` by default.

## Configuration

The server reads these environment variables:

| Variable | Default | Description |
| --- | --- | --- |
| `DATABASE` | `postgres://postgres@localhost` | PostgreSQL connection string. |
| `REDIS` | `redis://localhost:6379/0` | Redis or Valkey connection string. |
| `STORAGE` | current user home, or `storage` | Local storage path, or an S3-compatible `http(s)` URL. |
| `LISTEN` | `:7974` | Server listen address. |

Storage examples:

```bash
STORAGE=/storage
STORAGE=http://access:secret@localhost:9000/doj
STORAGE=https://access:secret@s3.example.com/doj
STORAGE=https://access:secret@s3.example.com/bucket?lookup=dns
```

`lookup=dns` forces virtual-host style bucket URLs. Use it only when your S3-compatible provider requires bucket names in the host.

Problem `data/` and `judge/` files share one content-addressed ZIP at `problems/{id}/packages/{hash}.zip`; its manifest and cases live in the problem row. Public statement assets remain separate under `problems/{id}/assets/`. An omitted case score defaults to 10, and totals are not capped at 100.

When `dist/` exists, the server serves the web app and supports history fallback for frontend routes.

## Judger

The judger is Linux-only. It reads:

| Variable | Default | Description |
| --- | --- | --- |
| `SERVER` | `http://localhost:7974` | DOJ server URL. |
| `TOKEN` | empty | Judger token created in the admin UI. |
| `CONCURRENCY` | `1` | Number of concurrent judging workers. |
| `CACHE_GB` | `20` | Maximum extracted problem cache size in GiB. |

The default compose judger shares the server container's network namespace, so it connects to `127.0.0.1` and is treated as a local judger. No token is needed for the default compose deployment.

The outer judger container is a deployment wrapper, not a security boundary. It intentionally runs privileged with the host PID and cgroup namespaces and the host Docker socket, which gives it host-root control, so run judgers only on dedicated, disposable machines. With privileged mode and the host cgroup namespace, Docker exposes the host cgroup hierarchy read-write; no explicit `/sys/fs/cgroup` bind is needed. Untrusted source is isolated by the runner language containers and per-case cgroups, not by the outer judger container. Runner binaries live in `/var/lib/doj/bin`, active work in `/var/lib/doj/tasks`, and problem cache in `/var/lib/doj/cache/P{id}/{hash}`.

## Development

Install frontend dependencies once:

```bash
pnpm install
```

Run the server:

```bash
go run . server
```

Run the Vite dev server:

```bash
pnpm dev --host 0.0.0.0 --port 28080
```

Run checks:

```bash
go test ./...
pnpm test
pnpm build
```

Regenerate the TypeScript API schema after editing the Go-first web contract in `server/api/openapi.go`:

```bash
pnpm api:gen
```

This writes the ignored OpenAPI artifact to `contract/web/openapi.yaml`, then updates `web/client/schema.ts`.

Linux judger validation should also run on a Linux host with Docker and cgroup v2 available:

```bash
go test ./judger -count=1
```

---

# DOJ 中文说明

DOJ 是一个可自部署的在线评测系统，包含 React 前端、Go API 服务，以及 Go 评测机/runner。

## 功能

- 题目、提交、比赛、作业、讨论、排名和用户资料。
- 支持 Markdown 题面、图片上传和数学公式渲染。
- 支持 ICPC / OI 榜单，以及 ICPC 封榜。
- Go 评测机使用 Docker 语言容器，并通过 cgroup v2 统计资源。
- 使用 PostgreSQL 存数据，Redis/Valkey 存 session 和运行态，支持本地或 S3 兼容对象存储。

## 快速开始

compose 会启动 PostgreSQL、Valkey、DOJ server 和本地评测机。server 容器同时提供 API 和前端静态文件：

```bash
docker compose up -d
```

打开 `http://localhost:7974`。如果数据库里还没有管理员，DOJ 会创建密码同为 `admin` 的 `admin` 账号；修改默认密码前，前端会持续给出醒目提示。

上传文件、题目数据和备份默认存放在 `./storage`。

## 配置

server 读取这些环境变量：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `DATABASE` | `postgres://postgres@localhost` | PostgreSQL 连接串。 |
| `REDIS` | `redis://localhost:6379/0` | Redis 或 Valkey 连接串。 |
| `STORAGE` | 当前用户 home，或 `storage` | 本地存储路径，或 S3 兼容 `http(s)` URL。 |
| `LISTEN` | `:7974` | server 监听地址。 |

存储配置示例：

```bash
STORAGE=/storage
STORAGE=http://access:secret@localhost:9000/doj
STORAGE=https://access:secret@s3.example.com/doj
STORAGE=https://access:secret@s3.example.com/bucket?lookup=dns
```

`lookup=dns` 强制使用 bucket 子域名风格。只有对象存储服务要求 bucket 出现在 host 里时才需要它。

题目的 `data/` 和 `judge/` 文件共用一个内容寻址 ZIP，路径为 `problems/{id}/packages/{hash}.zip`；清单和测试点信息保存在题目行中。题面公开附件独立存放在 `problems/{id}/assets/`。测试点未填写分数时默认 10 分，总分不限制为 100。

存在 `dist/` 时，server 会提供前端静态文件，并支持前端路由的 history fallback。

## 评测机

评测机只支持 Linux。它读取：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `SERVER` | `http://localhost:7974` | DOJ server 地址。 |
| `TOKEN` | 空 | 管理页创建的评测机 token。 |
| `CONCURRENCY` | `1` | 并发评测 worker 数。 |
| `CACHE_GB` | `20` | 题目解包缓存上限（GiB）。 |

默认 compose 里的评测机会共享 server 容器的网络命名空间，因此它连接 `127.0.0.1`，会被 server 视为本地评测机。默认 compose 部署不需要 token。

外层评测机容器只是部署包装，不是安全边界。它会以 privileged 模式使用宿主机 PID/cgroup namespace 和 Docker socket，等同拥有宿主机 root 控制权，因此只应运行在专用、可随时重建的机器上。privileged 模式配合宿主机 cgroup namespace 时，Docker 已会以可写方式暴露宿主机 cgroup 层级，无需显式映射 `/sys/fs/cgroup`。不受信源码由 runner 语言容器和逐用例 cgroup 隔离，而不是由外层评测机容器隔离。runner 二进制放在 `/var/lib/doj/bin`，当前评测工作目录放在 `/var/lib/doj/tasks`，题目缓存放在 `/var/lib/doj/cache/P{id}/{hash}`。

## 本地开发

首次安装前端依赖：

```bash
pnpm install
```

启动 server：

```bash
go run . server
```

启动 Vite：

```bash
pnpm dev --host 0.0.0.0 --port 28080
```

运行检查：

```bash
go test ./...
pnpm test
pnpm build
```

修改 `server/api/openapi.go` 里的 Go-first Web 契约后重新生成 TypeScript API schema：

```bash
pnpm api:gen
```

该命令会把忽略的 OpenAPI 产物写到 `contract/web/openapi.yaml`，再更新 `web/client/schema.ts`。

Linux 评测链路需要在具备 Docker 和 cgroup v2 的 Linux 主机上验证：

```bash
go test ./judger -count=1
```
