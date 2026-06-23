# judger 规则

`judger/` 放所有评测机相关代码，包括宿主侧 judger 和容器内 runner。

## 数据边界

- judger 只访问 server API。
- PostgreSQL、Valkey、S3/MinIO 只由 server 访问。
- 题包、提交源码、语言配置、任务信息和结果回传都通过 server judger API 流转。
- 语言配置来自 server 的源文件名和 Dockerfile。Dockerfile 需要复制提交源码并提供 `CMD`。
- 题目表里的内存限制是 MB，server 下发给 judger 的资源限制是 KB，judger 回写提交和 case 内存也使用 KB。

## 执行模型

- 每个 submission 启动或复用一个 warm language container。
- `doj-runner` 在 language container 内运行。
- 每个 case 重新启动 JudgeProgram 和 UserProgram。
- 普通评测、special judge、interactive judge 都使用同一 pipe 模型：
  - JudgeProgram stdout -> UserProgram stdin
  - UserProgram stdout -> JudgeProgram stdin
- JudgeProgram 的结构化结果走专用内部通道，UserProgram 不能继承该通道。

## judger / runner 协议

- judger 与 runner 使用 Unix domain socket + Go gob。
- 控制协议不复用 stdout/stderr。
- job 目录由 judger 创建并 bind mount 到 language container。

## cgroup 与进程

- Linux cgroup v2 是资源统计真值。
- 每个 case 建独立 cgroup。
- memory 真值来自 `memory.peak`。
- time、memory、OOM、exit code、signal 由 judger 观测。
- UserProgram 必须先进入 cgroup，再 exec 用户代码。
- JudgeProgram 和 UserProgram 使用不同子进程、不同 UID/GID、不同 process group。

UserProgram 启动顺序：

1. runner fork UserProgram child。
2. child 暂停，暂不 exec。
3. runner 通知 judger inner PID。
4. judger 映射 host PID。
5. judger 创建并配置 case cgroup。
6. judger 将 host PID 写入 cgroup.procs。
7. judger 通知 runner 放行。
8. child 设置 UID/GID、rlimit、关闭 fd 后 exec。

## 清理要求

- 每个 case 结束后清理 JudgeProgram process group。
- 每个 case 结束后清理 UserProgram process group。
- 关闭所有 pipe 和内部 fd。
- 清理 case 临时目录。
- 使用 cgroup 清理残留 UserProgram 进程。

## 测试要求

- 普通题、custom checker、interactive judge、Quine 类回显、输出爆炸、超时、编译限制和 case 隔离必须有 Go 测试覆盖。
- cgroup v2 的 memory/pids 恶意测试只在 Linux 上运行；设置 `CGROUP_TEST_ROOT` 后执行 `go test ./judger`。
- 非 Linux 环境的本地测试只能证明 runner 协议、pipe 模型和普通进程清理，不能替代 Linux cgroup 资源隔离测试。
