# DOJ

DOJ is an online judge rebuilt around a React frontend, a Go API server, and a Go judger/runner pair.

This branch is the v4 rewrite line built on the React/Go architecture described below.

## Stack

- Frontend: React, Vite, TypeScript, antd, TanStack Query.
- Backend: Go, Echo, GORM, OpenAPI.
- Judging: Go `doj-judger` on the host side and Go `doj-runner` inside language containers.
- Data services: PostgreSQL, Valkey, S3/MinIO.

## Layout

- `cmd/`: binary entries for `server`, `judger`, and `runner`.
- `api/`: OpenAPI contracts.
- `models/`: GORM models and database bootstrap.
- `services/`: server-side web/admin/judger API handlers.
- `judger/`: judger and runner implementation.
- `index.html`: Vite HTML entry.
- `web/`: Vite app source.
- `web/client/`: generated TypeScript API schema and frontend client wrapper.

## Development

Install dependencies with pnpm:

```bash
pnpm install
```

Start the API server. By default it connects to a real local PostgreSQL instance at `postgres://postgres@localhost/postgres?sslmode=disable` and creates a development administrator `admin` / `admin` if no administrator exists yet:

```bash
go run -tags server ./cmd/server.go
```

Start the web dev server:

```bash
pnpm dev --host 0.0.0.0 --port 28080
```

The Vite dev app defaults to API port `7974`. The product server always reads from the configured services; if the default PostgreSQL endpoint is unavailable, startup fails with the real connection error.

The server uses HTTP-only cookie sessions. If no administrator exists yet, the bootstrap settings create the first administrator. These settings all have development defaults and can be overridden:

```bash
DOJ_BOOTSTRAP_ADMIN=admin
DOJ_BOOTSTRAP_MAIL=admin@localhost
DOJ_BOOTSTRAP_PASSWORD=admin
```

After at least one administrator exists, normal restarts no longer use these bootstrap settings to create accounts.

During this early rewrite phase the server uses the GORM model set to keep the development schema in sync on startup. When the schema stabilizes, explicit migrations can be introduced without changing the API contract workflow.

For cross-origin development between Vite and the API server, an empty `DOJ_CORS_ORIGINS` accepts localhost, loopback, and private-network origins. Set it to a comma-separated exact origin list for stricter deployments. `DOJ_CORS_ORIGINS=*` is intentionally rejected because browser requests carry credentials.

`DOJ_DATABASE_URL` accepts the standard PostgreSQL URL form, such as `postgres://doj:secret@localhost:5432/doj?sslmode=disable`, and defaults to `postgres://postgres@localhost/postgres?sslmode=disable`. A judger connecting from loopback can run without a token for local development. Any non-loopback judger must use a token created in the admin judger page; pass it to `doj-judger` with `--token` or `--token-file`.

Cookie behavior can be configured explicitly:

```bash
DOJ_COOKIE_SECURE=true
DOJ_COOKIE_SAMESITE=lax
DOJ_COOKIE_DOMAIN=example.com
```

`DOJ_COOKIE_SECURE` accepts `auto`, `true`, or `false`; `auto` follows whether the incoming request is TLS. `DOJ_COOKIE_SAMESITE` accepts `lax`, `strict`, or `none`; `none` requires `DOJ_COOKIE_SECURE=true` and is only needed for truly cross-site browser requests. Leave `DOJ_COOKIE_DOMAIN` empty unless the frontend and API need to share cookies across subdomains.

Language syntax highlighting is matched from the language `id` through CodeMirror `language-data`. If the id is not supported by the bundled CodeMirror language descriptions, the editor falls back to plain text. Language execution is configured with a Dockerfile plus a source filename; the Dockerfile must copy that source from the build context and provide a `CMD` for the user program.

Problem memory limits are stored and edited in MB. Judger limits, submission memory, and per-case memory stay in KB.

The WYSIWYG Markdown editor is lazy-loaded. Its editor chunk is intentionally larger than ordinary page chunks, while the initial application shell stays separate.

Image uploads used by avatars and Markdown editors use the same server endpoint in every environment. By default the server stores objects under `DOJ_UPLOAD_DIR`; when S3/MinIO variables are configured, it stores them in the bucket. Stored object keys keep the stable business convention: user images use `users/{uid}/{yyyy}/{mm}/{dd}/{hash}.ext`, and problem Markdown images use `problems/{pid}/assets/{yyyy}/{mm}/{dd}/{hash}.ext`. The API returns shorter relative media URLs such as `/api/media/users/...` and `/api/media/problems/{pid}/...`; those URLs do not change the S3 key convention.
In the UI, the Markdown editor image upload hook uses this endpoint. The server detects image content type, derives the stored extension from the detected MIME type, rejects SVG content, proxies media with immutable cache headers, and applies a same-site referer guard for media reads.

```bash
DOJ_BODY_LIMIT=160M
DOJ_READ_HEADER_TIMEOUT=5s
DOJ_READ_TIMEOUT=0s
DOJ_WRITE_TIMEOUT=0s
DOJ_IDLE_TIMEOUT=60s
DOJ_UPLOAD_DIR=.data/uploads
DOJ_S3_ENDPOINT=http://localhost:9000
DOJ_S3_BUCKET=doj
DOJ_S3_ACCESS_KEY=doj
DOJ_S3_SECRET_KEY=doj-minio-password
DOJ_S3_USE_SSL=false
DOJ_SHUTDOWN_TIMEOUT=15s
DOJ_WEB_DIR=dist
```

`DOJ_BODY_LIMIT` defaults to `160M`, which leaves room for the current `128M` per-file problem asset upload limit plus multipart overhead.
`DOJ_READ_HEADER_TIMEOUT` defaults to `5s`, and `DOJ_IDLE_TIMEOUT` defaults to `60s`. `DOJ_READ_TIMEOUT` and `DOJ_WRITE_TIMEOUT` default to `0s` so large uploads and downloads are not interrupted by the Go server unless you set explicit limits.
`DOJ_SHUTDOWN_TIMEOUT` defaults to `15s`. The server handles `SIGINT` and `SIGTERM` with graceful shutdown, so container stop grace periods should be longer than this value.
When `DOJ_WEB_DIR` points to a built web directory, the server serves `index.html`, static assets, and H5 history fallback from that directory. The release server image sets this automatically.

The endpoint accepts either `host:port` or a URL with `http://` / `https://`. The bucket is created on first upload when the configured account is allowed to create buckets.

## API Contract

`api/web.yaml` is the web/admin API contract. Regenerate the frontend schema after contract changes:

```bash
pnpm api:gen
```

Go Echo handlers are implemented in the server code and must stay aligned with this contract. Keep OpenAPI changes small and update the handler, generated client, UI, and tests in the same patch.

## Health Checks

- `/api/health` is a shallow liveness check for the HTTP process.
- `/api/ready` checks runtime dependencies. In database mode it pings the database and returns `503` when dependencies are not ready.

## Docker Compose

An example compose stack is provided:

```bash
docker compose -f compose.example.yml up --build
```

It starts PostgreSQL, Valkey, MinIO, and one Go server that serves both the API and the built web app. Change all example secrets before any real deployment.
The server image builds the React app with `VITE_API_BASE=/` by default. `/api/*` is handled by Echo, and all other non-asset paths fall back to `index.html` for H5 history routing.
Most compose values can be overridden by environment variables or a `.env` file, including `DOJ_POSTGRES_PASSWORD`, `DOJ_MINIO_ROOT_USER`, `DOJ_MINIO_ROOT_PASSWORD`, `DOJ_BOOTSTRAP_ADMIN`, `DOJ_BOOTSTRAP_MAIL`, `DOJ_BOOTSTRAP_PASSWORD`, `DOJ_CORS_ORIGINS`, and `VITE_API_BASE`.
The optional `judger` profile starts a privileged judger container. Create a judger token in the admin UI, write it to a local ignored `judger.token` file, then run `docker compose -f compose.example.yml --profile judger up --build`. The server healthcheck uses `/api/ready`, so the judger waits for the database-backed API to be ready rather than merely listening.
The server service uses `DOJ_SHUTDOWN_TIMEOUT=15s` and a `20s` compose stop grace period so ordinary deploy restarts can drain in-flight requests before Docker sends a hard kill.
The judger example sets `DOJ_CGROUP_ROOT=/sys/fs/cgroup/doj` so user programs inside language containers are attached to host cgroups before execution.

## Tests

Frontend checks:

```bash
pnpm typecheck
pnpm test
pnpm build
```

Go tests:

```bash
go test ./...
go test -tags server ./cmd
```

OpenAPI generated-client drift check:

```bash
pnpm api:gen
```

Release binary builds:

```bash
CGO_ENABLED=0 go build -tags server -o .local/build/doj-server ./cmd/server.go
CGO_ENABLED=0 go build -tags judger -o .local/build/doj-judger ./cmd/judger.go
CGO_ENABLED=0 go build -tags runner -o .local/build/doj-runner ./cmd/runner.go
```

Compose syntax and wiring check:

```bash
docker compose -f compose.example.yml config
```

Linux cgroup tests require cgroup v2 and a writable test root:

```bash
DOJ_CGROUP_TEST_ROOT=/sys/fs/cgroup/doj-test go test ./judger
```

Docker-backed judger tests require Docker and are opt-in. With cgroup v2 available, this also covers warm language containers, host PID mapping, per-case cgroup attach, custom JudgeProgram assets, and basic UserProgram artifact isolation:

```bash
DOJ_DOCKER_TEST=1 DOJ_CGROUP_TEST_ROOT=/sys/fs/cgroup/doj-test go test ./judger -count=1
```

## Release Checklist

Before publishing a deployment, run the local checks above, the Linux judger tests, a compose smoke, and a browser smoke on the chosen access URL. Also scan tracked source and docs for private local hostnames, paths, and secrets; example compose values are placeholders and must be replaced in real deployments.

The GitHub Actions workflow runs the non-privileged gates: OpenAPI generated-client drift, Go tests, frontend checks, compose config, Docker image builds, release binary builds, and a tracked-source scan for private local paths or hosts. Linux cgroup and Docker-backed judger tests still need to run on a suitable Linux host.

## Current Notes

- Auth uses server sessions backed by Valkey when available, with an in-process fallback for local development. Development and production both read real server data.
- S3/MinIO is wired for avatars, Markdown images, and file-level problem assets. Default/strict judging consumes problem data assets through the server-to-judger API. Custom JudgeProgram execution is wired for executable/source assets under `judge/`, and `judge/Dockerfile` builds copy `/out/judge` through Docker. Linux validation covers runner protocol tests, cgroup v2 memory/pids tests, Dockerfile custom judge builds, warm-container multi-case reuse, host PID mapping, per-case cgroup attach, and UserProgram denial for `/jobs` answer/result/judge/control-socket artifacts.
- Valkey is used for lightweight session state when configured.
- The judger supports a long-running `serve` command, and the admin page exposes queue totals and managed judger records.

---

# DOJ 中文说明

DOJ 是一次围绕 React 前端、Go API 服务和 Go 评测机/runner 的在线评测系统重构。

本分支是基于下方 React/Go 架构的 v4 重构线。

## 技术栈

- 前端：React、Vite、TypeScript、antd、TanStack Query。
- 后端：Go、Echo、GORM、OpenAPI。
- 评测：宿主侧 Go `doj-judger`，语言容器内 Go `doj-runner`。
- 数据服务：PostgreSQL、Valkey、S3/MinIO。

## 目录

- `cmd/`：`server`、`judger`、`runner` 三个入口。
- `api/`：OpenAPI 契约。
- `models/`：GORM 模型和数据库初始化。
- `services/`：server 侧 web/admin/judger API。
- `judger/`：评测机和 runner 实现。
- `index.html`：Vite HTML 入口。
- `web/`：Vite 前端源码。
- `web/client/`：生成的 TypeScript API schema 和前端 client 封装。

## 本地开发

安装依赖：

```bash
pnpm install
```

启动 API。默认会连接真实的本地 PostgreSQL：`postgres://postgres@localhost/postgres?sslmode=disable`。如果当前还没有管理员，会创建开发管理员 `admin` / `admin`：

```bash
go run -tags server ./cmd/server.go
```

启动前端：

```bash
pnpm dev --host 0.0.0.0 --port 28080
```

Vite 开发前端默认访问 `7974` 端口的 API。产品 server 不提供示例 UI 数据；如果默认 PostgreSQL 连不上，启动会暴露真实连接错误。

server 使用 HTTP-only cookie session。如果当前还没有管理员，bootstrap 配置会创建第一个管理员。这些配置都有开发默认值，也可以覆盖：

```bash
DOJ_BOOTSTRAP_ADMIN=admin
DOJ_BOOTSTRAP_MAIL=admin@localhost
DOJ_BOOTSTRAP_PASSWORD=admin
```

至少存在一个管理员之后，正常重启不会再用这些 bootstrap 配置创建账号。

当前重构早期，server 启动时使用 GORM 模型集合保持开发库结构同步。等表结构稳定后，可以在不改变 OpenAPI 工作流的前提下再引入显式 migration。

Vite 与 API server 跨端口开发时，空的 `DOJ_CORS_ORIGINS` 会接受 localhost、loopback 和私有网段来源。需要更严格的部署时，可以把它设置成逗号分隔的精确允许来源。由于浏览器请求会携带凭据，`DOJ_CORS_ORIGINS=*` 会被启动配置拒绝。

`DOJ_DATABASE_URL` 支持标准 PostgreSQL URL 形式，例如 `postgres://doj:secret@localhost:5432/doj?sslmode=disable`，默认值是 `postgres://postgres@localhost/postgres?sslmode=disable`。从 loopback 连接的本机 judger 可以不带 token，用于快速开发；非 loopback judger 必须使用管理页创建的 token，并通过 `doj-judger --token` 或 `--token-file` 传入。

cookie 行为可以显式配置：

```bash
DOJ_COOKIE_SECURE=true
DOJ_COOKIE_SAMESITE=lax
DOJ_COOKIE_DOMAIN=example.com
```

`DOJ_COOKIE_SECURE` 支持 `auto`、`true`、`false`；`auto` 会跟随进入 Go server 的请求是否为 TLS。`DOJ_COOKIE_SAMESITE` 支持 `lax`、`strict`、`none`；`none` 要求 `DOJ_COOKIE_SECURE=true`，只有真正跨站请求时才需要。除非前端和 API 需要跨子域共享登录态，否则可以不设置 `DOJ_COOKIE_DOMAIN`。

语言高亮由语言配置里的 `id` 通过 CodeMirror `language-data` 匹配。若该 id 不在当前打包的 CodeMirror 语言描述中，编辑器会自动降级为纯文本。语言运行环境由 Dockerfile 和源文件名配置；Dockerfile 需要从构建上下文复制该源文件，并用 `CMD` 描述用户程序入口。

题目内存限制用 MB 存储和编辑。judger 限制、提交内存和单点内存继续使用 KB。

WYSIWYG Markdown 编辑器按需懒加载。它的编辑器 chunk 会明显大于普通页面 chunk，但不会进入首屏应用壳。

头像和 Markdown 编辑器图片上传共用同一个 server 接口。默认写入 `DOJ_UPLOAD_DIR`；配置 S3/MinIO 后写入对象存储桶。对象 key 继续遵守固定业务约定：用户图片使用 `users/{uid}/{yyyy}/{mm}/{dd}/{hash}.ext`，题目 Markdown 图片使用 `problems/{pid}/assets/{yyyy}/{mm}/{dd}/{hash}.ext`。API 返回更短的相对媒体 URL，例如 `/api/media/users/...` 和 `/api/media/problems/{pid}/...`；这个 URL 变化不代表 S3 key 约定变化。
前端里，Markdown 编辑器的图片上传 hook 会走这个接口。server 会检测图片内容类型，按照检测到的 MIME 类型决定存储扩展名，拒绝 SVG 内容，并在代理读取媒体时加长期缓存头和同站 referer 防外链检查。

```bash
DOJ_BODY_LIMIT=160M
DOJ_READ_HEADER_TIMEOUT=5s
DOJ_READ_TIMEOUT=0s
DOJ_WRITE_TIMEOUT=0s
DOJ_IDLE_TIMEOUT=60s
DOJ_UPLOAD_DIR=.data/uploads
DOJ_S3_ENDPOINT=http://localhost:9000
DOJ_S3_BUCKET=doj
DOJ_S3_ACCESS_KEY=doj
DOJ_S3_SECRET_KEY=doj-minio-password
DOJ_S3_USE_SSL=false
DOJ_SHUTDOWN_TIMEOUT=15s
DOJ_WEB_DIR=dist
```

`DOJ_BODY_LIMIT` 默认是 `160M`，用于覆盖当前单个题目资源 `128M` 的上传上限和 multipart 额外开销。
`DOJ_READ_HEADER_TIMEOUT` 默认是 `5s`，`DOJ_IDLE_TIMEOUT` 默认是 `60s`。`DOJ_READ_TIMEOUT` 和 `DOJ_WRITE_TIMEOUT` 默认是 `0s`，避免 Go server 在未显式设置限制时打断较慢的大文件上传或下载。
`DOJ_SHUTDOWN_TIMEOUT` 默认是 `15s`。server 会在收到 `SIGINT` 和 `SIGTERM` 时优雅退出，因此容器 stop grace period 应该长于这个值。
`DOJ_WEB_DIR` 指向构建后的前端目录时，server 会从该目录提供 `index.html`、静态资源和 H5 history fallback。发布用 server 镜像会自动设置它。

`DOJ_S3_ENDPOINT` 可以写 `host:port`，也可以写带 `http://` / `https://` 的 URL。配置账号有权限时，bucket 会在首次上传时自动创建。

## API 契约

`api/web.yaml` 是 web/admin API 契约。契约变化后需要重新生成前端 schema：

```bash
pnpm api:gen
```

Go Echo handler 在 server 代码中手写实现，并且必须和这份契约保持一致。OpenAPI 变更应小步提交，同步更新 handler、生成 client、UI 和测试。

## 健康检查

- `/api/health` 是 HTTP 进程的浅层存活检查。
- `/api/ready` 会检查运行时依赖。数据库模式下会 ping 数据库；依赖不可用时返回 `503`。

## Docker Compose

仓库提供 compose 示例：

```bash
docker compose -f compose.example.yml up --build
```

示例会启动 PostgreSQL、Valkey、MinIO，以及一个同时提供 API 和前端静态文件的 Go server。正式部署前必须替换示例密钥。
server 镜像默认使用 `VITE_API_BASE=/` 构建 React 前端。`/api/*` 由 Echo 处理，其他非资源路径会 fallback 到 `index.html`，用于 H5 history 路由。
compose 中的大部分示例值都可以通过环境变量或 `.env` 覆盖，包括 `DOJ_POSTGRES_PASSWORD`、`DOJ_MINIO_ROOT_USER`、`DOJ_MINIO_ROOT_PASSWORD`、`DOJ_BOOTSTRAP_ADMIN`、`DOJ_BOOTSTRAP_MAIL`、`DOJ_BOOTSTRAP_PASSWORD`、`DOJ_CORS_ORIGINS` 和 `VITE_API_BASE`。
可选的 `judger` profile 会启动 privileged judger 容器。先在管理页创建 judger token，写入本地已忽略的 `judger.token` 文件，再运行 `docker compose -f compose.example.yml --profile judger up --build`。server healthcheck 使用 `/api/ready`，因此 judger 会等待数据库-backed API 真正 ready，而不是只等 HTTP 端口开始监听。
server 服务使用 `DOJ_SHUTDOWN_TIMEOUT=15s` 和 `20s` compose stop grace period，普通部署重启时可以在 Docker 硬杀之前尽量完成正在处理的请求。
judger 示例会设置 `DOJ_CGROUP_ROOT=/sys/fs/cgroup/doj`，让宿主侧任务在用户代码开始前把已暂停的用户进程写入 cgroup。

## 测试

前端检查：

```bash
pnpm typecheck
pnpm test
pnpm build
```

Go 测试：

```bash
go test ./...
go test -tags server ./cmd
```

OpenAPI 生成 client 漂移检查：

```bash
pnpm api:gen
```

发布二进制构建：

```bash
CGO_ENABLED=0 go build -tags server -o .local/build/doj-server ./cmd/server.go
CGO_ENABLED=0 go build -tags judger -o .local/build/doj-judger ./cmd/judger.go
CGO_ENABLED=0 go build -tags runner -o .local/build/doj-runner ./cmd/runner.go
```

Compose 语法和服务连线检查：

```bash
docker compose -f compose.example.yml config
```

Linux cgroup 测试需要 cgroup v2 和可写测试目录：

```bash
DOJ_CGROUP_TEST_ROOT=/sys/fs/cgroup/doj-test go test ./judger
```

Docker 相关评测测试需要 Docker，默认不跑。具备 cgroup v2 时，这组测试也会覆盖 warm language container、host PID 映射、逐 case cgroup attach、自定义 JudgeProgram 资产，以及 UserProgram 对评测产物的基础隔离：

```bash
DOJ_DOCKER_TEST=1 DOJ_CGROUP_TEST_ROOT=/sys/fs/cgroup/doj-test go test ./judger -count=1
```

## 发布检查

正式发布前需要跑完上面的本地检查、Linux 评测机测试、compose smoke，以及在最终访问地址上的浏览器 smoke。还需要扫描 tracked source 和文档，确认没有私有本地主机名、路径或密钥；compose 示例里的值都是占位，真实部署必须替换。

GitHub Actions workflow 会自动跑不需要 privileged 权限的门禁：OpenAPI 生成 client 漂移检查、Go 测试、前端检查、compose config、Docker 镜像构建、发布二进制构建，以及 tracked source 私有本机路径/主机扫描。Linux cgroup 和 Docker 评测机测试仍需要在合适的 Linux 主机上执行。

## 当前说明

- 认证 session 在配置 Valkey 时写入 Valkey，本地开发可退回进程内存；开发和部署都读取真实 server 数据。
- S3/MinIO 已接入头像、Markdown 图片和题目资产的文件级管理。默认/严格评测已通过 server-to-judger API 消费题目数据资产；`judge/` 下的可执行文件/源码形式自定义 JudgeProgram 已接入，`judge/Dockerfile` 构建会通过 Docker 拷出 `/out/judge`。当前 Linux 验证已覆盖 runner 协议测试、cgroup v2 memory/pids 测试、Dockerfile 自定义评测构建、warm container 多 case 复用、host PID 映射、逐 case cgroup attach，以及 UserProgram 对 `/jobs` 中答案、结果、judge、控制 socket 产物的读取隔离。
- Valkey 已用于轻量 session 状态。
- judger 已支持长驻 `serve` 命令，管理页会展示队列总览和评测机记录。
