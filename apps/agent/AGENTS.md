# Agent AI 规则

## 范围

本文件约束 `apps/agent/`。Agent 只连接 server，在本机 Docker 执行评测；Runner 是 Agent 内部模块。

## 架构原则

- Agent 不连接 PostgreSQL、Redis 或 S3；只通过 server 的 HTTP/WS 协议获取任务和评测包。
- `SERVER` 默认 `http://127.0.0.1:7974`，跨机部署必须配置。
- `SECRET` 使用 `Authorization: Bearer <SECRET>` 和 server 共享。
- `AGENT_NAME` 默认主机名，`AGENT_CONCURRENCY` 默认 1。
- Agent `key` 是运行态连接 ID，不落 PostgreSQL，不要求跨重启稳定。
- Runner 逻辑放在 `apps/agent/src/runner/`，不要重新拆回独立 workspace package。

## 协议

- WebSocket 只走 JSON 控制消息：server 发 `ping`、`run`；Agent 发 `hello`、`pong`、`progress`、`result`。
- Agent 最终只发一次 `result`；错误合并成 `result.status='SE'` 和 message，不单独发 error。
- Agent 不感知题目 `mode`；server 已经把 mode 翻译为 `JudgerSpec`。
- `default`/`strict` 使用预构建判题镜像 `doveccl/doj:judger`，由 `CHECK=trim|pe` 切换语义。
- `custom` 才使用题目根目录 `Dockerfile` 构建 A 镜像。
- `bundleHash` 是题目评测文件内容指纹；Agent 缓存未命中时通过 `GET /api/agents/bundle/:problemId` 下载 tar。

## Runner 与沙箱

- A/B 评测模型固定为双 FIFO + shell 重定向：A 读写 checker/interactor，B 是用户程序。
- A/B 镜像必须含 `/bin/sh`，不支持 `FROM scratch`。
- B 容器资源字段只代表用户程序：CPU time、峰值内存、输出上限；A 资源只用于判定和排障。
- verdict 由 A 容器 exit code + stderr 给出；Agent 不计算分数，score 由 server 按 case AC 均分。
- 对 B 固定禁用网络，drop capabilities，限制 pid，使用只读根文件系统和 tmpfs。
- 必须同时有 CPU time 限制和 wall-clock cap，避免 sleep 或阻塞程序永远占用任务。
- 构建用户提交包时使用更高但固定的 build memory，避免 C++ 编译器被运行时内存限制误杀。
