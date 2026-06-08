# 评测内核设计

本文件是评测核心实现细节的单一事实源：server/Agent 协议、调度租约、题目资产下发、A/B 评测模型、沙箱限制、缓存与实时进度。产品形态见 `01-overview-and-ui.md`，schema 与存储约定见 `02-data-and-storage.md`，HTTP/WS 接口字段见 `03-api.md`。

## 代码组织

Runner 作为概念保留，但实现合入 `apps/agent`，不长期保留独立 `packages/runner` workspace package，也不把 Runner 逻辑散进 Agent 入口文件。推荐结构：

```text
apps/agent/src/
  index.ts
  connection.ts
  jobRunner.ts
  cache/
    dataCache.ts
  runner/
    dockerRunner.ts
    judge.ts
    sandbox.ts
    verdict.ts
    testdata.ts
```

新代码文件名尽量不使用连字符 `-`，优先 camelCase 或简洁单词。

## 总体形态

这一版使用合并 `server` 架构：

- server 提供 HTTP API、消费评测任务、接受浏览器 WebSocket 和 Agent WebSocket。
- Agent 只连接 server，在本机 Docker 执行评测，不连接 PostgreSQL、Redis、S3。

如果未来需要多 server 横向扩展，必须重新设计共享 Agent registry、job lease、WebSocket fanout 和任务调度。

## 评测模式

题目的评测行为由 `problems.mode` 决定，固定三种：

- `default`：默认检查器，对答案与 B 输出做 trim 比较（CR/LF 归一、行尾空白与尾部空行裁剪），不产生 PE。
- `strict`：严格检查器，使用同一判题逻辑但启用 PE 风格判定（字节级 trim 一致为 AC；trim 不一致但 ASCII whitespace 切分后的 token 序列一致为 PE；否则 WA）。
- `custom`：自定义评测，使用题目根目录 `Dockerfile` 构建 A 镜像，覆盖 special judge、交互题、Quine 等场景。

关键简化：

- `default` 和 `strict` 共用一个**预构建判题镜像** `doveccl/doj:judger`，行为由 server 下发的 `CHECK` 环境变量切换。99% 普通题目**不再生成或构建任何 A 镜像**，Agent 也不缓存判题镜像（首次 `docker pull` 后由 Docker 自身保留）。
- 只有 `custom` 题目才构建 A 镜像。
- 镜像名不带版本后缀，文档不引入 digest / sha256 pinning 作为设计要求。

server 把 `mode` 翻译成 A 容器的镜像与环境后下发，**Agent 不感知 `mode`**：Agent 只拿到「A 镜像来源 + A 环境变量 + cases + limits + B 构建上下文」，对所有题目执行同一套 A/B 编排。

## 提交流程

1. 用户在统一题目详情页提交代码。
2. server 校验登录、限流、语言、题目可见性和软删除状态。
3. server 按题目、用户、当前时间窗口和范围自动判断 `contestId` 与 `assignmentId`，二者可同时存在。
4. server 插入 `submissions`，状态 `WAITING`。
5. server 在同一事务创建 `judge_tasks`，未完成任务对同一 `submissionId` 保持唯一。
6. 分配任务给可用 Agent 并把 `judge_tasks.status` 改为 `RUNNING` 时，同事务把 `submissions.status` 改为 `JUDGING`。
7. 评测完成后写最终 submission 和 submission_cases。
8. 若最终 AC，写 Redis solved/ranking/count 派生状态。
9. WebSocket 推送状态和 per-case progress；PG 不保存 live progress。

失败策略：

- 初版不自动重试 system error，失败一次即最终 `SE`。
- 不让用户看到 `SE -> JUDGING -> 最终状态` 的跳变。

## Agent 连接

Agent 配置：

- `SERVER`：连接地址，默认 `http://127.0.0.1:7974`，跨机部署必须配置。
- `SECRET`：server/Agent 共享 secret。
- `AGENT_NAME`：展示名，默认主机名，取不到时 `agent`。
- `AGENT_CONCURRENCY`：并发数，默认 `1`。

认证：

- 使用 `Authorization: Bearer <SECRET>`，`SECRET` 来自环境变量。
- 不使用 PG `judge_agents`、不使用 query string token、不做 labels、不做在线刷新 secret（轮换需改 env 并重启）。

Agent hello：

```json
{
  "type": "hello",
  "info": {
    "key": "9f3a7c2d4e1b8a60",
    "name": "Local Agent",
    "concurrency": 1,
    "version": "..."
  }
}
```

`key` 规则：

- `key` 是运行态连接 ID，不落 PostgreSQL，不要求跨重启稳定。
- Agent 启动时生成 `key = sha256(AGENT_NAME + hostname + processStartTime + 16 random bytes).slice(0, 16)`，进程内重连沿用同一 `key`。
- `version` 取 Agent 包 `package.json` version；取不到时用根仓库 version，仍取不到固定 `unknown`。
- server 要求当前在线连接的 `key` 唯一；碰撞时拒绝新连接并记录日志，Agent 退避后重新生成 `key` 再连。
- 管理端展示 `name` 和 `key`，用户识别机器主要看 `name`。

## Server 与 Agent 协议

Agent 主动连接 server。连接后：

- Agent 发 `hello`。
- server 每 10 秒发 `ping`；Agent 收到 `ping` 后立即发 `pong`，携带 `activeJobs`。Agent 也可每 10 秒主动发一次 `pong` 作为 heartbeat；server 只以最近一次 `pong` 时间判断在线。
- server 超过 30 秒未收到 `pong` 视为离线。
- server 发 `run`；Agent 发 `progress`；Agent 最终只发一次 `result`。
- 不单独发 `error`，错误合并为 `result.status='SE'` 和 message。
- 初版不做用户/管理员 cancel；内部超时仍可中止本地 Docker。

固定消息契约：

```ts
type JudgeStatus =
  | 'WAITING'
  | 'JUDGING'
  | 'AC'
  | 'WA'
  | 'PE'
  | 'TLE'
  | 'MLE'
  | 'OLE'
  | 'RE'
  | 'CE'
  | 'SE'

interface AgentHello {
  key: string
  name: string
  concurrency: number
  version: string
}

interface JudgeCase {
  caseNo: number
  inputPath: string  // /data 下 POSIX 相对路径，如 data/in1.txt
  answerPath: string // /data 下 POSIX 相对路径，如 data/ans1.txt
}

// 只携带每题独有的两个基础限制；A/B 派生限制、wall-clock、build 超时、pid 与 output
// 上限都是常量或可从这两个值推导，固定写在“沙箱限制”里由 Agent 自算，不上协议。
interface JudgeLimit {
  timeLimit: number   // 毫秒
  memoryLimit: number // 字节
}

// JudgerSpec 描述 A 容器来源；Agent 据此决定是 pull 预构建镜像还是 build 自定义镜像，
// 但不据此分支判题逻辑。
type JudgerSpec =
  | { kind: 'prebuilt'; image: string; check: 'trim' | 'pe' }
  | { kind: 'custom'; bundleHash: string }

interface SubmissionPackage {
  languageId: string
  source: string      // 源文件名，必须等于语言 source，如 main.cc
  code: string        // 用户源码原文
  dockerfile: string  // 语言 Dockerfile
}

interface JudgePayload {
  submissionId: number
  problemId: number
  bundleHash: string  // 题目评测文件包（data/ 及 custom 评测资源）的内容指纹
  judger: JudgerSpec
  cases: JudgeCase[]
  limits: JudgeLimit
  submission: SubmissionPackage
}

type JudgePhase = 'queued' | 'building-a' | 'building-b' | 'running' | 'finished'

interface JudgeProgress {
  phase: JudgePhase
  caseNo?: number
  status?: JudgeStatus
  timeMs?: number
  memoryBytes?: number
  message?: string
}

interface JudgeCaseResult {
  caseNo: number
  status: JudgeStatus
  timeMs: number
  memoryBytes: number
  message: string
}

interface JudgeResult {
  status: JudgeStatus
  timeMs: number
  memoryBytes: number
  message: string
  cases: JudgeCaseResult[]
}

type ServerToAgent =
  | { type: 'run'; jobId: string; payload: JudgePayload }
  | { type: 'ping' }

type AgentToServer =
  | { type: 'hello'; info: AgentHello }
  | { type: 'pong'; activeJobs: number }
  | { type: 'progress'; jobId: string; progress: JudgeProgress }
  | { type: 'result'; jobId: string; result: JudgeResult }
```

WebSocket 只走 JSON 控制消息，不混传二进制。题目评测文件（`data/` 及 `custom` 评测资源）走 HTTP 下载，见“题目文件下发”。

字段规则：

- `JudgeLimit` 只含 `timeLimit`（B 的 CPU time 上限，毫秒）和 `memoryLimit`（B 的内存上限，字节）；A 的派生限制、wall-clock cap、build 超时、PID 与 output 上限都是常量或可从这两个值推导，由 Agent 按“沙箱限制”自算，不上协议。
- `JudgerSpec.kind='prebuilt'` 时 `image` 固定为 `doveccl/doj:judger`，`check` 取 `trim`（default 模式）或 `pe`（strict 模式）。
- `JudgerSpec.kind='custom'` 时 Agent 用题目评测文件内的根目录 `Dockerfile` 构建 A 镜像。
- `bundleHash` 是 server 计算的题目评测文件内容指纹，详见“题目文件下发”。
- `SubmissionPackage` 内联 B 构建上下文，随 `run` 一起下发，不单独下载。
- `JudgeCaseResult.timeMs` 和 `memoryBytes` 固定表示用户程序 B 的 CPU time 和峰值内存；A 容器资源只用于判定和排障，不写入 case 资源字段。
- `JudgeProgress` 是增量展示事件，server 可覆盖保存同一 submission 的最新 progress。
- `JudgeResult` 是最终结果，每个 job 只发一次；Agent 只上报 status/time/memory/message/cases，不上报分数，score 由 server 计算。

## 调度与租约

`judge_tasks` 是调度事实源，在线 Agent 连接是可用执行资源。

1. 创建提交时，server 在同一事务插入 `submissions(status='WAITING')` 和 `judge_tasks(status='WAITING')`。
2. 调度循环固定每 1 秒运行一次；新任务创建或 Agent 连接/空闲时也立即触发一次。
3. 按 `judge_tasks.createdAt ASC` 取 WAITING 或锁过期的 RUNNING 任务。
4. 可用 Agent 条件：WebSocket 在线、最近 30 秒有 heartbeat、`activeJobs < concurrency`。
5. Agent 选择规则：`activeJobs/concurrency` 最小优先，再按最近一次 heartbeat 最早、`key` 字典序升序。
6. 分配任务时把 `judge_tasks.status` 改为 `RUNNING`，设置 `lockedUntil = now + 60s`，记录内存态 `jobId -> agentKey`。
7. server 每 20 秒为仍在线且仍在执行的任务续租，把 `lockedUntil` 延长到 `now + 60s`。
8. Agent heartbeat 中的 `activeJobs` 只作为调度容量信号；任务事实以 `judge_tasks` 和 server 内存态为准。
9. Agent 断开或 30 秒无 heartbeat 后停止续租；`lockedUntil` 到期后任务重新可调度。
10. 同一任务重新调度使用新 `jobId`；旧 Agent 迟到返回 result 时，server 因 `jobId` 不匹配或任务已完成而丢弃。
11. 收到有效 `result` 后，server 在事务中写 submission、submission_cases，并把 `judge_tasks.status` 改为 `DONE`。
12. server 处理 job 遇到不可恢复基础设施错误时，写最终 SE、cases 空，`judge_tasks.status=FAILED`，首版不自动重试；Agent 返回含 case 的 SE 属于有效结果，任务状态 `DONE`。
13. server 重启后内存态丢失；启动扫描 `RUNNING` 且 `lockedUntil <= now` 的任务重新调度，未过期 RUNNING 等过期后回收。

状态映射：

- `judge_tasks.WAITING` 对应 `submissions.WAITING`。
- `judge_tasks.RUNNING` 对应 `submissions.JUDGING`。
- `judge_tasks.DONE` 对应 `submissions` 的最终非运行态。
- `judge_tasks.FAILED` 对应 `submissions.SE`，cases 为空。

## 题目文件下发

Agent 不直连 S3，也不接收文件列表。server 内部负责 list、校验、计算 `bundleHash`，并通过 HTTP 提供下载。

server 内部行为：

1. list `problems/{problemId}/`。
2. 排除 `statement.md` 和 `assets/`（与评测无关）。
3. 校验 `data/` 完整性；`custom` 题目额外校验根目录 `Dockerfile` 及其依赖资源存在。
4. 基于评测相关文件（`default`/`strict` 为 `data/`；`custom` 为 `data/` + `Dockerfile` + 评测资源）计算 `bundleHash`。

`bundleHash`：

- SHA-256，小写 hex，对 Agent 不透明。
- 覆盖评测相关文件的路径和内容变化；`statement.md`、`assets/` 不参与。
- 每个文件条目计算 `contentHash = sha256(file bytes)`；按 `path` 字典序升序，将 `path\nsize\ncontentHash\n` 拼接后整体 SHA-256。
- `path` 使用归一化 POSIX 相对路径，禁止前导 `/`、`.`、`..`。
- 仅用于 Agent 侧 `data/` 缓存命中判断和 `custom` A 镜像重建判断，不作为跨题目镜像 LRU 的 key。
- 如果对象存储只提供 multipart etag，server 不能假设其等价于内容 hash；必要时上传时自行计算并保存到对象 metadata。

下载通道：

- 评测文件走 HTTP 二进制下载，不走 WebSocket，避免 JSON 与二进制混传。
- 接口固定为 `GET /api/agents/bundle/:problemId`（鉴权见下），返回该题评测文件的 tar 流（`data/`，`custom` 额外含根目录 `Dockerfile` 和评测资源；不含 `statement.md`、`assets/`），响应头带 `X-Bundle-Hash`。
- 该接口与浏览器侧 `GET /api/problems/:id/assets/:filename` 是不同用途：前者是 Agent 取整包评测文件（SECRET 鉴权、含 `data/` 和评测资源），后者是浏览器取单个公开附件（session/访客鉴权、只服务 `assets/`）。鉴权、粒度、内容都不同，不合并。
- HTTP 鉴权中间件同时支持两种凭据：浏览器走 `Authorization: Bearer <session-token>`，Agent 走 `Authorization: Bearer <SECRET>`；`/api/agents/*` 只接受 SECRET。

Agent 下载策略：

- server 在 `run` 中下发 `bundleHash`、`judger`、cases、limits 和 `submission`，不附带文件本体。
- Agent 检查本地 `data/` 缓存是否已有同一 `bundleHash`；命中则不下载，未命中再 `GET /api/agents/bundle/:problemId`。
- 对 `custom` 题目，Agent 还需判断本地是否已有与 `bundleHash` 对应的 A 镜像；无则下载后构建。
- Agent 不需要知道 S3 endpoint、bucket、etag、文件列表或题目 `mode`，只和 server 交互。

文件落地与构建：

- tar 解包到 Agent 本地缓存目录，`data/` 运行时只读挂载到容器 `/data`，永不进任何 docker build context，也不 bake 进镜像。
- `custom` 构建 A 镜像时，build context 只取根目录 `Dockerfile` 和评测资源、显式排除 `data/`；用 dockerode 的本地目录 build（`{ context, src }`）由库自行打包上下文，server 不再额外打包构建用 tarball。
- `custom` A 镜像 build 完成后删除解包的临时目录；`data/` 缓存保留。

## 题目完整性与命名

- 所有题目都必须有完整 `data/`。
- 不能认为交互题只有一个 case；case 由 `data/` 决定。
- data 不完整时，管理端上传/编辑就应暴露，不允许普通用户提交后才发现。

宽松命名规则：

- 输入文件：文件名包含 `in` 且包含数字。
- 输出文件：文件名包含 `out` 或 `ans` 且包含数字。
- 从文件名提取第一个数字串作为 `caseNo`。
- 每个 `caseNo` 必须恰好一个输入和一个输出；缺一或重复都是题目配置错误。

## 统一 A/B 评测模型

A 是题目容器（判题器/checker/interactor），B 是用户提交容器。所有 `mode` 都归一化为同一套 A/B 编排，Agent 不做题目形态分支。

固定契约：

- 每个 case 启动一组新的 A/B 容器，共享一个本次 run 专属的临时目录。
- `data/` 只读挂载到 A 容器 `/data`；A 工作目录固定 `/judge`。
- A 容器环境变量（不加前缀）：
  - `CASE_NO`：当前 case 序号。
  - `INPUT`：输入文件路径，如 `/data/in1.txt`。
  - `OUT`：答案文件路径，如 `/data/ans1.txt`。
  - `SOURCE`：用户提交源码文件路径（只读挂载，供 Quine 等 checker 读取）。
  - `TIME_LIMIT_MS`、`MEMORY_LIMIT_BYTES`：B 的限制，供 interactor 参考。
- A stdin 连接 B stdout，A stdout 连接 B stdin，A stderr 作为 case message。
- A/B 都禁网，都有资源限制。
- A 不自主产出 score，Agent 也不产出 score；verdict 用 A 的 exit code + stderr。

### A/B 通信：双 FIFO + shell 重定向

A/B 之间用两个 FIFO 直连内核管道，由 Agent 负责创建、接管和清理。**这是经实测确定的方案**：Docker attach/exec 的进程内桥接（host-attach bridge、双连接半关）都无法对容器 stdin 单独投递 EOF，会导致单文件多 case 的程序读不到 EOF 而挂死；双 FIFO 在 200KB 级输入下可干净半关、无 pipe buffer 死锁。

- 每个 run 在 Agent 临时目录下开唯一 slot：`<agent-tmp>/<runId>/{a2b,b2a}`，host 侧 `mkfifo` 创建，bind-mount 进 A/B 容器固定路径 `/pipe/a2b`、`/pipe/b2a`。多 slot 路径互不冲突，run 结束（含超时/异常）在 `finally` 中 `rm -rf` slot 目录并强删容器。
- 路径不写死：FIFO 路径含 `runId`，由 Agent 统一管理，A/B 程序心智上只是纯 stdio，不感知 FIFO。
- 接管方式：Agent 覆写容器启动命令，用 shell 把 FIFO 接到 fd0/fd1，再 `exec` 镜像原始入口。A/B 镜像必须含 `/bin/sh`（预构建判题镜像 `FROM busybox` 自带；语言镜像如 gcc/python/node 自带；`custom` A 镜像与用户语言镜像均要求含 `/bin/sh`，不支持 `FROM scratch` 用户镜像）。
  - A 容器：`["/bin/sh","-c","exec 0</pipe/b2a 1>/pipe/a2b; exec \"$@\"","sh", ...A 原始 argv]`。
  - B 容器：`["/bin/sh","-c","exec 0</pipe/a2b 1>/pipe/b2a; exec \"$@\"","sh", ...B 原始 argv]`。
  - A/B 原始 argv 由 Agent 通过 image inspect 读取镜像默认 `Entrypoint`/`Cmd` 得到；Agent 不要求用户在 Dockerfile 里手写重定向。
- 半关语义：A 写完 `INPUT` 后关闭自身 stdout（即 `a2b` 写端）→ B 读到 EOF → B 输出后退出 → B 关闭 `b2a` → A 读到 EOF。default/strict 判题器必须**并发**执行“写 input 到 stdout 并在写完后关闭 stdout”和“从 stdin 读 B 输出到 EOF”，不能先完整写 input 再读，否则在单文件多组数据、B 边读边流式输出时，`b2a` 超过系统 pipe buffer（典型 64 KiB）后会与 A 互相阻塞形成死锁。
- 启动顺序：先创建 FIFO，再启动 A，再启动 B；二者并发运行，任一方超时/异常时结束该 case。
- 交互题（custom）A 可在 stdin/stdout 上与 B 多轮交互，仍按 env 读取 `CASE_NO`、`INPUT`、`OUT`。

### 预构建判题镜像 `doveccl/doj:judger`

- 多阶段构建：builder 阶段用 Go 编译静态二进制 `/judge/checker`；runtime 阶段 `FROM busybox`，复制 `/judge/checker`，`CMD ["/judge/checker"]`。选用 busybox（约 4–5MB）而非 scratch，体积可接受且自带 `/bin/sh` 与 coreutils，便于接管 FIFO 和排障；设计上不对镜像做全局 prune。
- 判题器读取 `CHECK` 环境变量切换比较语义：`trim`（default）只做 trim 比较，不产生 PE；`pe`（strict）trim 一致为 AC（exit 0），token 一致但 trim 不一致写 `presentation error\n` 到 stderr 并 exit 2，否则写 `wrong answer\n` 并 exit 1。
- trim 归一：先把 CRLF 和单独 CR 都转 LF；按 LF 分行；每行只移除行尾 ASCII space `0x20` 和 tab `0x09`；只删末尾连续空行；中间空行和行首空白保留；用单个 LF 连接且不追加结尾 LF。
- 判题器固定使用 Go 静态二进制，不使用脚本语言；用 `io.Copy` + goroutine 并发写读，规避大 IO 死锁。
- 该镜像由仓库 CI 构建并推送；server 在 `run` 中把 `image` 固定为 `doveccl/doj:judger`，Agent 首次使用时 `docker pull`。判题逻辑升级即重新发布镜像，不在题目层做模板版本指纹。

判题器 Go 源码参考（runtime 阶段 `FROM busybox`）：

```Dockerfile
FROM golang:1.24-alpine AS build
WORKDIR /src
RUN cat > /src/checker.go <<'EOF'
package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

func fail(code int, message string) {
	_, _ = fmt.Fprint(os.Stderr, message)
	os.Exit(code)
}

func normalize(data []byte) []byte {
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	data = bytes.ReplaceAll(data, []byte("\r"), []byte("\n"))
	lines := bytes.Split(data, []byte("\n"))
	for i := range lines {
		lines[i] = bytes.TrimRight(lines[i], " \t")
	}
	for len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	return bytes.Join(lines, []byte{'\n'})
}

func fields(data []byte) [][]byte {
	return bytes.FieldsFunc(data, func(r rune) bool {
		switch r {
		case ' ', '\t', '\n', '\r', '\v', '\f':
			return true
		default:
			return false
		}
	})
}

func fieldsEqual(a, b [][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !bytes.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

func writeInput(path string, done chan<- error) {
	input, err := os.Open(path)
	if err != nil {
		done <- err
		return
	}
	defer input.Close()
	_, _ = io.Copy(os.Stdout, input)
	_ = os.Stdout.Close() // 半关：B 读到 EOF
	done <- nil
}

func main() {
	inputPath := os.Getenv("INPUT")
	answerPath := os.Getenv("OUT")
	pe := os.Getenv("CHECK") == "pe"
	if inputPath == "" || answerPath == "" {
		fail(6, "missing checker environment\n")
	}

	done := make(chan error, 1)
	go writeInput(inputPath, done)

	actual, err := io.ReadAll(os.Stdin)
	if err != nil {
		fail(6, "failed to read contestant output\n")
	}
	if err := <-done; err != nil {
		fail(6, "failed to read input file\n")
	}

	expected, err := os.ReadFile(answerPath)
	if err != nil {
		fail(6, "failed to read answer file\n")
	}

	if bytes.Equal(normalize(actual), normalize(expected)) {
		os.Exit(0)
	}
	if pe && fieldsEqual(fields(actual), fields(expected)) {
		fail(2, "presentation error\n")
	}
	fail(1, "wrong answer\n")
}
EOF
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /checker /src/checker.go

FROM busybox
WORKDIR /judge
COPY --from=build /checker /judge/checker
CMD ["/judge/checker"]
```

### 自定义评测（custom）

- server 用题目根目录 `Dockerfile` 和评测资源构建 A 镜像。
- A 镜像运行阶段必须含 `/bin/sh`（用于 Agent 接管 FIFO），工作目录推荐 `WORKDIR /judge`。
- 自定义 checker/interactor 通过 env 读取 `CASE_NO`、`INPUT`、`OUT`、`SOURCE`，从 `/data` 读输入答案；Agent 始终把 `data/` 挂到 `/data`，自定义 Dockerfile 无需自行 COPY data。
- special judge、交互题、Quine 都属于 custom；verdict 同样用 exit code + stderr。
- custom A 镜像构建策略：按 `bundleHash` 触发构建，评测资源变化时重建；构建成功后删除同题旧 A 镜像，不做全局 prune；A 镜像不计入 `AGENT_CACHE_GB`。镜像尽量保持小（multi-stage、只保留运行所需）。

### exit code 与判定

A 容器 exit code 固定映射：

- `0` AC、`1` WA、`2` PE、`3` TLE（checker 判定超时类）、`4` MLE、`5` OLE、其他非 0 SE。

B 容器判定优先级：

- B build 失败为 CE。
- B 运行超时为 TLE。
- B OOM 或 exit 137 为 MLE。
- B stdout/stderr 超过 output limit 为 OLE。
- B 非 0 退出为 RE。
- B 正常退出后，以 A 的 exit code 判定该 case。

stderr / message：

- A stderr 作为该 case 的 message。
- 写 PostgreSQL 前按 UTF-8 安全截断到 4096 字符，截断追加 `\n[message truncated]`。
- B build logs 作为 CE 的 submission message，同样截到 4096 字符。
- WebSocket progress message 最多 4096 字符，不推完整 64 MiB 输出。
- 是否对普通用户展示由权限和题目隐藏策略决定。

### 分数

- server 是分数唯一事实源，`JudgeResult` 不含 score。
- 先按 `caseNo` 升序排序 `JudgeCaseResult`。
- 每个 AC case 得该 case 分，非 AC 得 0；score 是整数，题目总分固定 100，首版不做 per-case 权重。
- 均分时 `base=floor(100/caseCount)`，前 `100-base*caseCount` 个 case 各多 1 分，保证总和 100。
- 最终 status 全 AC 才 AC，否则取 `caseNo` 升序中第一个非 AC status。
- Agent 返回 CE 时 server 写 `score=0`、cases 空、status CE。
- 基础设施错误导致无有效 case result 时，server 写 `score=0`、cases 空、status SE，`judge_tasks.status=FAILED`。
- 单个 case 的 A 以未知错误退出映射为 SE 是有效结果，写入该 case SE，按 `caseNo` 聚合最终 status，`judge_tasks.status=DONE`。

## 沙箱限制

B 容器：禁网、非 root、只读 rootfs、drop capabilities、`no-new-privileges`、PID 限制、内存限制、CPU 限制、stdout/stderr 输出限制、tmpfs 工作目录。

A 容器：同样禁网、非 root、drop caps、`no-new-privileges`，有 CPU/memory/PID/output 限制；资源可比 B 宽但必须有默认值；如果 `custom` Dockerfile 需要 root，应在构建阶段完成，运行阶段仍按受限用户执行。

固定资源参数（`JudgeLimit` 只下发 `timeLimit`、`memoryLimit`，其余全部是 Agent 侧常量或由这两个值推导）：

- B 的 CPU time 限制用 `timeLimit`，内存限制用 `memoryLimit`。
- A 的 CPU time 限制固定 `max(2 * timeLimit, timeLimit + 1000)`，内存限制固定 `max(2 * memoryLimit, memoryLimit + 128MiB)`。
- wall-clock safety cap 固定 `max(3 * timeLimit, timeLimit + 5000, 10000)`。
- A 镜像 build wall-clock timeout 固定 300 秒；B 镜像 build wall-clock timeout 固定 60 秒（仅 custom 才有 A build）。
- A/B 单容器 PID 上限固定 128。
- A/B 各自 stdout+stderr 合计输出上限固定 64 MiB；B 超限判 OLE，A 超限判 SE，A 也可用 exit code 5 明确返回 OLE。

时间与内存：

- 用 CPU time 判 TLE；B 超 `timeLimit` 判 TLE；A 超 `max(2 * timeLimit, timeLimit + 1000)` 判 SE，除非 A 自行 exit 3 返回 TLE。
- wall-clock safety cap 用上面固定的公式；任一容器触发即结束该 case，B 未结束判 TLE，只有 A 阻塞且 B 已正常结束时判 SE。
- 内存用 Docker stats 峰值；OOMKilled 或 exit 137 判 MLE；Docker Desktop/Colima 可能无法精确读取极短峰值，文档和测试需记录此限制。

### B 带编译环境的安全性结论

B 镜像通常带编译/运行时环境（如 gcc、python）。这不增加**容器逃逸**面：逃逸取决于内核/runtime 漏洞，与镜像内是否有编译器无关；真正兜底的是上面的沙箱（禁网、只读 rootfs、非 root、drop caps、seccomp、资源限制）。编译环境的实际成本是镜像体积和编译阶段可能滥用资源/网络，因此语言 Dockerfile 在 build 阶段禁网、不做在线依赖安装，运行阶段禁网。B 用完即删，不做缓存。

## 语言镜像契约

每种语言保存 `source` 文件名和 Dockerfile。server 准备 `run.payload.submission` 时：

- build context 根目录只含用户源码文件和语言 Dockerfile。
- 用户源码文件名必须等于语言 `source`，如 `cpp` 用 `main.cc`。
- Dockerfile 必须在 build 阶段完成编译，并通过 `CMD` 或 `ENTRYPOINT` 定义运行入口。
- 最终镜像必须含 `/bin/sh`（用于 Agent 接管 FIFO）。
- B 运行时不传额外 command，Agent 通过 image inspect 取默认入口后用 shell 包裹接管 FIFO。
- Docker build B 失败即 CE，build logs 作为 message，cases 空，score 0。
- build 超过固定的 B build timeout（60 秒）也按 CE，message 写 build timeout。
- B 运行阶段非 0 退出是 RE，不是 CE。
- 语言 Dockerfile 不得依赖网络下载；首版不做在线依赖安装。

## Progress 与浏览器 WebSocket

- v3 的实时进度必须保留；前端通过 WebSocket 接收 per-case progress；PG 不保存 live progress。
- 浏览器连 server `/api/ws`，鉴权固定读 `Sec-WebSocket-Protocol: doj-auth.<token>`，不用 query string token。
- 用户订阅具体 submission id 或当前用户提交 feed。
- server 在内存/Redis 保存短 TTL progress，Redis progress TTL 固定 10 分钟。
- 断线重连后前端先 HTTP 拉 submission detail，再恢复 WS 订阅；评测完成后主要读 PG final cases。
- 用户只能订阅自己有权查看的 submission；管理员可订阅任意 submission；比赛封榜/隐藏信息仍由 server 裁剪。

接口字段（`BrowserToServer`/`ServerToBrowser`、心跳）见 `03-api.md`。

## 缓存与临时文件

- `data/` 缓存：Agent 按 `bundleHash` 在本地缓存目录保存解压后的 `data/`，运行时只读挂载到 `/data`；上限由 `AGENT_CACHE_GB` 控制，`0` 无限制，未配置按 `min(4GB, 0.5 * availableDiskGB)` 估算，LRU 清理最久未用的 `data/`。
- 不缓存预构建判题镜像（由 Docker 自身保留）和 B 镜像（用户源码镜像，用完即删）。
- `custom` A 镜像按 `bundleHash` 关联，资源变化重建并删旧，不计入 `AGENT_CACHE_GB`，不做全局 prune。
- 下载的评测文件 tar 不持久保留：A 镜像 build 完成或 `data/` 落缓存后，立即删除下载的 tar 和临时解包目录。
- 每个 job 结束后清理 run slot 目录、FIFO、临时容器和 volume。

## Status

数据库 submission status 固定为 `WAITING`、`JUDGING`、`AC`、`WA`、`PE`、`TLE`、`MLE`、`OLE`、`RE`、`CE`、`SE`。

- `FROZEN` 不是数据库 status，只是比赛查询/展示态。
- PE 只在 `strict` 和 `custom` checker 中出现。
- 部分分提交不可能是 AC；solved 只看最终 `status='AC'`。
- OLE 来自 B 输出超限或 A 明确 exit 5。
- 最终 status 聚合规则是“第一个非 AC case status”，全 AC 才 AC。
- OI 用 `score` 排名；作业按 AC/完成率展示，可附带 best score。

## 统计修复

首版不提供 rejudge 功能，也不提供 rejudge API，只需考虑 Redis 被 clear 后的统计修复。

- cron fix 定期从 submissions 修复 Redis solved/ranking/count。
- 管理页展示上次修复时间、状态、耗时；管理端可手动触发统计修复。
- 未来增加 rejudge 时，必须先补 API 契约，再定义重判完成后的局部 repair。
- 统计可不强实时，优先节省算力。
