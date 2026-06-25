# DOJ

DOJ is a self-hosted online judge with a React web app, a Go API server, and a Go judger/runner.

## Features

- Problems, submissions, contests, assignments, discussions, ranking, and user profiles.
- Markdown problem statements with image uploads and math rendering.
- ICPC and OI contest scoreboards, including ICPC freeze support.
- Go judger with Docker-based language containers and cgroup v2 resource measurement.
- PostgreSQL for data, Redis/Valkey for sessions and runtime state, and local or S3-compatible object storage.

## Quick Start

The example compose file starts PostgreSQL, Valkey, MinIO, and one DOJ image. The server container serves both the API and the built web app:

```bash
docker compose -f compose.example.yml up --build
```

Open `http://localhost:28080`. If the database has no administrator, DOJ creates:

```text
username: admin
password: admin
```

Change the default password before exposing the service.

## Configuration

The server reads three environment variables:

| Variable | Default | Description |
| --- | --- | --- |
| `DATABASE` | `postgres://postgres@localhost` | PostgreSQL connection string. |
| `REDIS` | `redis://localhost:6379/0` | Redis or Valkey connection string. |
| `STORAGE` | current user home | Local storage path, or an S3-compatible `http(s)` URL. |

Storage examples:

```bash
STORAGE=/var/lib/doj
STORAGE=http://access:secret@localhost:9000/doj
STORAGE=https://access:secret@s3.example.com/doj?region=auto
```

The server listens on `:7974`. When `dist/` exists, it serves the web app and supports history fallback for frontend routes.

## Judger

The judger is Linux-only. It reads:

| Variable | Default | Description |
| --- | --- | --- |
| `SERVER` | `http://localhost:7974` | DOJ server URL. |
| `TOKEN` | empty | Judger token created in the admin UI. |

To run the compose judger profile, create a judger in the admin UI, copy the token, then run:

```bash
TOKEN=replace-with-generated-token docker compose -f compose.example.yml --profile judger up --build
```

The judger profile uses the same DOJ image with a different command. It is privileged, uses the host Docker socket, and mounts `/var/lib/doj:/var/lib/doj`.

## Development

Install frontend dependencies:

```bash
pnpm install
```

Run the server:

```bash
go run ./cmd/server.go
```

Run the Vite dev server:

```bash
pnpm dev --host 0.0.0.0 --port 28080
```

Regenerate the TypeScript API schema after editing `api/web.yaml`:

```bash
pnpm api:gen
```

Run checks:

```bash
go test ./...
pnpm test
pnpm build
```

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

示例 compose 会启动 PostgreSQL、Valkey、MinIO，以及一个 DOJ 镜像。server 容器同时提供 API 和前端静态文件：

```bash
docker compose -f compose.example.yml up --build
```

打开 `http://localhost:28080`。如果数据库里还没有管理员，DOJ 会创建：

```text
用户名：admin
密码：admin
```

正式暴露服务前请先修改默认密码。

## 配置

server 读取三个环境变量：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `DATABASE` | `postgres://postgres@localhost` | PostgreSQL 连接串。 |
| `REDIS` | `redis://localhost:6379/0` | Redis 或 Valkey 连接串。 |
| `STORAGE` | 当前用户 home | 本地存储路径，或 S3 兼容 `http(s)` URL。 |

存储配置示例：

```bash
STORAGE=/var/lib/doj
STORAGE=http://access:secret@localhost:9000/doj
STORAGE=https://access:secret@s3.example.com/doj?region=auto
```

server 固定监听 `:7974`。存在 `dist/` 时，server 会提供前端静态文件，并支持前端路由的 history fallback。

## 评测机

评测机只支持 Linux。它读取：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `SERVER` | `http://localhost:7974` | DOJ server 地址。 |
| `TOKEN` | 空 | 管理页创建的评测机 token。 |

运行 compose 评测机 profile 前，先在管理页创建评测机并复制 token：

```bash
TOKEN=replace-with-generated-token docker compose -f compose.example.yml --profile judger up --build
```

评测机 profile 使用同一个 DOJ 镜像，但用不同 command 启动。它使用 privileged 模式，挂载宿主机 Docker socket，并挂载 `/var/lib/doj:/var/lib/doj`。

## 本地开发

安装前端依赖：

```bash
pnpm install
```

启动 server：

```bash
go run ./cmd/server.go
```

启动 Vite：

```bash
pnpm dev --host 0.0.0.0 --port 28080
```

修改 `api/web.yaml` 后重新生成 TypeScript API schema：

```bash
pnpm api:gen
```

运行检查：

```bash
go test ./...
pnpm test
pnpm build
```

Linux 评测链路需要在具备 Docker 和 cgroup v2 的 Linux 主机上验证：

```bash
go test ./judger -count=1
```
