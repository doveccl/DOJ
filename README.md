# DOJ

DOJ is an online judge built with a React frontend, a Go API server, and a Go judger/runner pair.

## Stack

- Frontend: React, Vite, TypeScript, antd, TanStack Query.
- Backend: Go, Echo, GORM, OpenAPI.
- Judging: Go `doj-judger` on the host side and Go `doj-runner` inside language containers.
- Data services: PostgreSQL, Redis/Valkey, S3/MinIO or local storage.

## Layout

- `cmd/`: binary entries for `server`, `judger`, and `runner`.
- `api/`: OpenAPI contracts.
- `models/`: GORM models and database access.
- `services/`: server-side web/admin/judger API handlers.
- `judger/`: judger and runner implementation.
- `index.html`: Vite HTML entry.
- `web/`: frontend source.
- `web/client/`: generated TypeScript API schema and client wrapper.

## Configuration

The server intentionally keeps runtime configuration small:

| Variable | Default | Meaning |
| --- | --- | --- |
| `DATABASE` | `postgres://postgres@localhost` | PostgreSQL connection URL. |
| `REDIS` | empty | Redis/Valkey URL. Empty uses in-process session/cache state. |
| `STORAGE` | `/var/lib/doj` | Local storage path, or an S3-compatible `http(s)` URL. |

`STORAGE` examples:

```bash
STORAGE=/var/lib/doj
STORAGE=http://access:secret@localhost:9000/doj
STORAGE=https://access:secret@s3.example.com/doj?region=auto
```

When `STORAGE` is local, objects are written under that directory. When it starts with `http://` or `https://`, the URL user info supplies S3 credentials, the host is the endpoint, and the single path segment is the bucket name.

The server listens on `:7974`, serves built web files from `dist` when present, and falls back to `index.html` for H5 history routes. If no administrator exists, startup creates `admin` / `admin` with `admin@localhost`.

The judger reads only:

| Variable | Default | Meaning |
| --- | --- | --- |
| `SERVER` | `http://localhost:7974` | Server API base URL. |
| `TOKEN` | empty | Judger token generated in the admin UI. |

Judger working data is stored under `/var/lib/doj`: runner binary cache in `bin/`, jobs in `jobs/`, and cgroup root at `/sys/fs/cgroup/doj`.

## Development

Install dependencies:

```bash
pnpm install
```

Start the API server:

```bash
go run -tags server ./cmd/server.go
```

Start the web dev server:

```bash
pnpm dev --host 0.0.0.0 --port 28080
```

The frontend calls same-origin `/api` by default. During Vite development, `/api` is proxied to the server on `7974`; in deployment, the Go server serves both API and built web files. The product server reads real database/storage state; it does not serve mock application data.

During early development, GORM keeps the development schema aligned on startup. When the schema stabilizes, explicit migrations can be introduced without changing the OpenAPI workflow.

## Live Updates

The browser uses SSE at `/api/events` for lightweight live updates. Submission creation, judger lease, and judger result callbacks publish a generic submission-change event, and the frontend invalidates relevant TanStack Query data instead of keeping a second client-side cache.

Judgers receive work through `/api/judger/lease` long polling. If no task is ready, the server holds the request briefly and returns `task: null`; when a submission changes, waiting leases wake and try to reserve work immediately. Task ownership and renewal live on the submission row (`judger_id`, `lease_until`, `attempt`). Session/cache state always uses Redis/Valkey; empty `REDIS` connects to the local default Redis endpoint.

## API Contract

`api/web.yaml` is the web/admin API contract. Regenerate the frontend schema after contract changes:

```bash
pnpm api:gen
```

Go Echo handlers must stay aligned with the contract. Keep OpenAPI changes small and update the handler, generated client, UI, and tests together.

## Storage

Business object keys are derived from ids and are not duplicated in the database when they can be inferred:

- User assets: `users/{uid}/{yyyy}/{mm}/{dd}/{hash}.ext`
- Problem assets: `problems/{pid}/*`
- Problem Markdown images: `problems/{pid}/assets/{hash}.ext`

The API returns relative media URLs such as `/api/users/{uid}/{yyyy}/{mm}/{dd}/{file}` and `/api/problems/{pid}/assets/{file}`. The server rejects SVG uploads, detects image content type, serves media with immutable cache headers, and applies a same-site referer guard for media reads.

Problem memory limits are stored and edited in MB. Judger limits, submission memory, and per-case memory use KB.

## Docker Compose

Run the example stack:

```bash
docker compose -f compose.example.yml up --build
```

It starts PostgreSQL, Valkey, MinIO, and one Go server image that serves both the API and the built web app. Change all example secrets before any real deployment.

The optional `judger` profile starts a privileged judger container. Create a judger in the admin UI, copy its token, set `TOKEN`, and run:

```bash
TOKEN=replace-with-generated-token docker compose -f compose.example.yml --profile judger up --build
```

The judger container controls host Docker through `/var/run/docker.sock` and mounts `/var/lib/doj:/var/lib/doj` so the host-side controller and language containers agree on runner and job paths.

## Tests

Frontend checks:

```bash
pnpm typecheck
pnpm test
pnpm build
```

Go checks:

```bash
go test ./...
go test -tags server ./cmd
```

On Linux, also test judger and runner entrypoints:

```bash
go test -tags judger ./cmd
go test -tags runner ./cmd
```

Release binary builds:

```bash
CGO_ENABLED=0 go build -tags server -o .local/build/doj-server ./cmd/server.go
GOOS=linux CGO_ENABLED=0 go build -tags judger -o .local/build/doj-judger ./cmd/judger.go
GOOS=linux CGO_ENABLED=0 go build -tags runner -o .local/build/doj-runner ./cmd/runner.go
```

Compose syntax check:

```bash
docker compose -f compose.example.yml config
```

Linux judger tests automatically use Docker and cgroup v2 when available:

```bash
go test ./judger -count=1
```

## Release Checklist

Before publishing a deployment, run the checks above, Linux judger tests, a compose smoke, and a browser smoke on the chosen access URL. Also scan tracked source and docs for private local hostnames, paths, and secrets.

---

# DOJ 中文说明

DOJ 是一个基于 React 前端、Go API 服务和 Go 评测机/runner 的在线评测系统。

## 技术栈

- 前端：React、Vite、TypeScript、antd、TanStack Query。
- 后端：Go、Echo、GORM、OpenAPI。
- 评测：宿主侧 Go `doj-judger`，语言容器内 Go `doj-runner`。
- 数据服务：PostgreSQL、Redis/Valkey、S3/MinIO 或本地存储。

## 目录

- `cmd/`：`server`、`judger`、`runner` 三个入口。
- `api/`：OpenAPI 契约。
- `models/`：GORM 模型和数据库访问。
- `services/`：server 侧 web/admin/judger API。
- `judger/`：评测机和 runner 实现。
- `index.html`：Vite HTML 入口。
- `web/`：前端源码。
- `web/client/`：生成的 TypeScript API schema 和 client 封装。

## 配置

server 只读取少量运行配置：

| 变量 | 默认值 | 含义 |
| --- | --- | --- |
| `DATABASE` | `postgres://postgres@localhost` | PostgreSQL 连接 URL。 |
| `REDIS` | 空 | Redis/Valkey URL；为空时使用进程内 session/cache。 |
| `STORAGE` | `/var/lib/doj` | 本地存储目录，或 S3 兼容 `http(s)` URL。 |

`STORAGE` 示例：

```bash
STORAGE=/var/lib/doj
STORAGE=http://access:secret@localhost:9000/doj
STORAGE=https://access:secret@s3.example.com/doj?region=auto
```

`STORAGE` 为本地路径时对象写入该目录。以 `http://` 或 `https://` 开头时，URL 用户信息是 S3 账号密码，host 是 endpoint，唯一 path 段是 bucket。

server 固定监听 `:7974`。存在 `dist` 时会提供构建后的前端静态文件，并对 H5 history 路由 fallback 到 `index.html`。如果数据库里还没有管理员，启动时会创建 `admin` / `admin`，邮箱为 `admin@localhost`。

judger 只读取：

| 变量 | 默认值 | 含义 |
| --- | --- | --- |
| `SERVER` | `http://localhost:7974` | server API 地址。 |
| `TOKEN` | 空 | 管理页生成的评测机 token。 |

judger 工作数据固定在 `/var/lib/doj` 下：runner 二进制缓存放 `bin/`，任务目录放 `jobs/`，cgroup root 为 `/sys/fs/cgroup/doj`。

## 本地开发

安装依赖：

```bash
pnpm install
```

启动 API：

```bash
go run -tags server ./cmd/server.go
```

启动前端：

```bash
pnpm dev --host 0.0.0.0 --port 28080
```

前端默认访问同源 `/api`。Vite 开发时 `/api` 代理到 `7974` 端口的 server；部署时 Go server 同时提供 API 和构建后的前端文件。产品 server 读取真实数据库和存储状态，不提供 mock 应用数据。

当前快速开发阶段，server 启动时使用 GORM 保持开发库结构同步。等表结构稳定后，可以在不改变 OpenAPI 工作流的前提下引入显式 migration。

## 实时更新

浏览器通过 `/api/events` 的 SSE 获取轻量实时更新。创建提交、judger 领取任务、judger 回传结果时会广播通用的提交变化事件，前端只失效相关 TanStack Query 数据，不再维护第二套客户端缓存。

评测机通过 `/api/judger/lease` 长轮询领取任务。没有任务时 server 会短暂挂起请求并返回 `task: null`；提交状态变化时，等待中的 lease 会被唤醒并立即尝试领取任务。任务归属和续租保存在提交记录的 `judger_id`、`lease_until`、`attempt`。session/cache 状态统一使用 Redis/Valkey；`REDIS` 为空时连接本地默认 Redis。

## API 契约

`api/web.yaml` 是 web/admin API 契约。契约变化后重新生成前端 schema：

```bash
pnpm api:gen
```

Go Echo handler 必须和契约保持一致。OpenAPI 变更应小步提交，并同步更新 handler、生成 client、UI 和测试。

## 存储

能从业务 id 推导出来的对象 key 不在数据库里重复保存：

- 用户资产：`users/{uid}/{yyyy}/{mm}/{dd}/{hash}.ext`
- 题目资产：`problems/{pid}/*`
- 题目 Markdown 图片：`problems/{pid}/assets/{hash}.ext`

API 返回 `/api/users/{uid}/{yyyy}/{mm}/{dd}/{file}`、`/api/problems/{pid}/assets/{file}` 这类相对媒体 URL。server 会拒绝 SVG 上传、检测图片 MIME、为媒体读取加长期缓存头，并对媒体读取做同站 referer 防外链检查。

题目内存限制使用 MB 存储和编辑。judger 限制、提交内存和单点内存使用 KB。

## Docker Compose

运行示例栈：

```bash
docker compose -f compose.example.yml up --build
```

示例会启动 PostgreSQL、Valkey、MinIO，以及一个同时提供 API 和构建后前端的 Go server 镜像。正式部署前必须替换示例密钥。

可选的 `judger` profile 会启动 privileged judger 容器。先在管理页创建评测机并复制 token，然后运行：

```bash
TOKEN=replace-with-generated-token docker compose -f compose.example.yml --profile judger up --build
```

judger 容器通过 `/var/run/docker.sock` 控制宿主机 Docker，并挂载 `/var/lib/doj:/var/lib/doj`，让宿主侧控制程序和语言容器使用一致的 runner 与 job 路径。

## 测试

前端检查：

```bash
pnpm typecheck
pnpm test
pnpm build
```

Go 检查：

```bash
go test ./...
go test -tags server ./cmd
```

在 Linux 上额外测试 judger 和 runner 入口：

```bash
go test -tags judger ./cmd
go test -tags runner ./cmd
```

发布二进制构建：

```bash
CGO_ENABLED=0 go build -tags server -o .local/build/doj-server ./cmd/server.go
GOOS=linux CGO_ENABLED=0 go build -tags judger -o .local/build/doj-judger ./cmd/judger.go
GOOS=linux CGO_ENABLED=0 go build -tags runner -o .local/build/doj-runner ./cmd/runner.go
```

Compose 语法检查：

```bash
docker compose -f compose.example.yml config
```

Linux 评测机测试会在可用时自动使用 Docker 和 cgroup v2：

```bash
go test ./judger -count=1
```

## 发布检查

发布前需要跑完上面的检查、Linux 评测机测试、compose smoke，以及最终访问地址上的浏览器 smoke。还需要扫描 tracked source 和文档，确认没有私有本地主机名、路径或密钥。
