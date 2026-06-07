# 评测内核设计

## 代码组织结论

Runner 作为概念保留，但实现应合入 `apps/agent`。不要长期保留独立 `packages/runner` workspace package，也不要把 Runner 逻辑散进 Agent 入口文件。推荐结构：

```text
apps/agent/src/
  index.ts
  connection.ts
  jobRunner.ts
  cache/
    imageCache.ts
  runner/
    dockerRunner.ts
    judge.ts
    sandbox.ts
    verdict.ts
    testdata.ts
```

新代码文件名尽量不要使用连字符 `-`；优先使用 camelCase 或简洁单词命名。


## 总体形态

这一版使用合并 `server` 架构：

- server 提供 HTTP API。
- server 消费评测任务。
- server 接受浏览器 WebSocket。
- server 接受 Agent WebSocket。
- Agent 只连接 server，在本机 Docker 执行评测。
- Agent 不连接 PostgreSQL、Redis、S3。

这样避免“单 Worker 但独立 Worker 包”造成的架构歧义。如果未来需要多 server 横向扩展，必须重新设计共享 Agent registry、job lease、WebSocket fanout 和任务调度。

## 提交流程

1. 用户在统一题目详情页提交代码。
2. server 校验登录、限流、语言、题目可见性和软删除状态。
3. server 根据题目、用户、当前时间窗口和范围自动判断 `contestId` 与 `assignmentId`，二者可以同时存在。
4. server 插入 `submissions`，状态为 `WAITING`。
5. server 创建待评测任务，未完成任务对同一 `submissionId` 保持唯一。
6. server 分配任务给可用 Agent 并把 `judge_tasks.status` 改为 `RUNNING` 时，同一事务把 `submissions.status` 改为 `JUDGING`。
7. 评测完成后写最终 submission 和 submission_cases。
8. 如果最终 AC，写 Redis solved/ranking/count 派生状态。
9. WebSocket 推送状态和 per-case progress；PG 不保存 live progress。

失败策略：

- 初版不自动重试 system error。
- 失败一次即最终 `SE`。
- 不在重试中让用户看到 `SE -> JUDGING -> 最终状态` 的跳变。

## Agent 连接

Agent 配置：

- `SERVER`：Agent 连接 server 的地址；默认 `http://127.0.0.1:7974`，跨机器部署必须配置。
- `SECRET`：server/Agent 共享 secret。
- `AGENT_NAME`：展示名；默认主机名，取不到主机名时为 `agent`。
- `AGENT_CONCURRENCY`：并发数；默认 `1`。

认证：

- 使用 `Authorization: Bearer <secret>`。
- `SECRET` 来自环境变量。
- 不使用 PG `judge_agents`。
- 不使用 query string token。
- 不做 labels。
- 不做在线刷新 secret；需要轮换时修改 server/agent env 并重启。

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
- Agent 启动时生成 `key = sha256(AGENT_NAME + hostname + processStartTime + 16 random bytes).slice(0, 16)`。
- 同一 Agent 进程重连时沿用本次启动生成的 `key`。
- `version` 固定取 Agent 包的 `package.json version`；如果构建产物无法读取 package version，则使用根仓库 version；仍取不到时固定为 `unknown`。
- server 要求当前在线连接的 `key` 唯一；发生碰撞时拒绝新连接并记录日志，Agent 退避后重新生成 `key` 再连接。
- 管理端展示 `name` 和 `key`，但用户识别机器主要看 `name`。

Admin Agents 页面展示运行态：name、key、concurrency、active jobs、version、connectedAt、lastHeartbeat 和接入指引。

## Server 与 Agent 协议

Agent 主动连接 server。连接后：

- Agent 发 `hello`。
- server 每 10 秒发送 `ping`；Agent 收到 `ping` 后必须立即发送 `pong`，携带 activeJobs。
- Agent 也可以每 10 秒主动发送一次 `pong` 作为 heartbeat；server 只以最近一次 `pong` 时间判断在线状态。
- server 超过 30 秒未收到 `pong` 视为离线。
- server 发 `run`。
- Agent 发 `progress`。
- Agent 最终只发一次 `result`。
- 不单独发 `error`；错误合并为 `result.status='SE'` 和 message。
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
  inputPath: string
  answerPath: string
}

interface JudgeLimit {
  timeLimitMs: number
  memoryLimitBytes: number
  aTimeLimitMs: number
  aMemoryLimitBytes: number
  aBuildTimeoutMs: 300000
  bBuildTimeoutMs: 60000
  pidLimit: 128
  outputLimitBytes: number
  wallClockLimitMs: number
}


`JudgeLimit.wallClockLimitMs` 固定等于 `max(3 * timeLimitMs, timeLimitMs + 5000ms, 10000ms)`，由 server 计算后下发。
interface SubmissionPackage {
  languageId: string
  source: string
  code: string
  dockerfile: string
  contextHash: string
}

interface JudgePayload {
  submissionId: number
  problemId: number
  problemBundleHash: string
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
  | { type: 'problem-package'; jobId: string; tgz: ArrayBuffer }
  | { type: 'ping' }

type AgentToServer =
  | { type: 'hello'; info: AgentHello }
  | { type: 'problem-package-request'; jobId: string }
  | { type: 'pong'; activeJobs: number }
  | { type: 'progress'; jobId: string; progress: JudgeProgress }
  | { type: 'result'; jobId: string; result: JudgeResult }
```

字段规则：

- `JudgeCase.inputPath` 和 `answerPath` 是 A 镜像内的 POSIX 相对路径，固定指向 `data/` 下文件。
- `JudgePayload.limits` 由 server 按题目限制和固定公式计算后下发，Agent 必须按该值执行。
- `SubmissionPackage.source` 是源文件名，必须等于语言 `source`，例如 `main.cc`。
- `SubmissionPackage.code` 是用户源码原文。
- `SubmissionPackage.dockerfile` 是该语言的 Dockerfile；Agent 用 `source/code/dockerfile` 组装 B build context。
- `SubmissionPackage.contextHash = sha256(source + "\n" + sha256(code) + "\n" + sha256(dockerfile) + "\n")`，只用于日志和排错，不作为缓存 key。
- B 构建上下文不通过单独消息下发，也不进入题目 `tgz`；它固定内联在 `run.payload.submission` 中。
- `JudgeCaseResult.timeMs` 和 `memoryBytes` 固定表示用户程序 B 容器的 CPU time 和峰值内存；A 容器资源只用于判定和排障，不写入 case 资源字段。
- `JudgeProgress` 是增量展示事件；server 可以覆盖保存同一 submission 的最新 progress。
- `JudgeResult` 是最终结果，Agent 对每个 job 只能发送一次。
- Agent 只上报 status、time、memory、message 和 cases，不上报最终分数；server 是 score、case score 和最终 status 的唯一事实计算方。

## 调度与租约

`judge_tasks` 是调度事实源，在线 Agent 连接是可用执行资源。

固定流程：

1. 创建提交时，server 在同一事务中插入 `submissions(status='WAITING')` 和 `judge_tasks(status='WAITING')`。
2. 调度循环固定每 1 秒运行一次；新任务创建或 Agent 连接/空闲时也必须立即触发一次调度。
3. 调度循环按 `judge_tasks.createdAt ASC` 取 WAITING 或锁过期的 RUNNING 任务。
4. 可用 Agent 条件是 WebSocket 在线、最近 30 秒有 heartbeat、`activeJobs < concurrency`。
5. Agent 选择规则固定为 `activeJobs/concurrency` 最小优先，再按 `lastHeartbeatAt` 最早、`key` 字典序升序。
6. 分配任务时，server 将 `judge_tasks.status` 改为 `RUNNING`，设置 `lockedUntil = now + 60s`，并记录内存态 `jobId -> agentKey`。
7. server 每 20 秒为仍在线且仍在执行的任务续租一次，把 `lockedUntil` 延长到 `now + 60s`。
8. Agent heartbeat 中的 `activeJobs` 只作为调度容量信号；任务事实仍以 `judge_tasks` 和 server 内存态为准。
9. Agent 断开或 30 秒无 heartbeat 后，server 停止续租；`lockedUntil` 到期后任务重新进入可调度状态。
10. 同一任务重新调度时使用新的 `jobId`；旧 Agent 如果迟到返回 result，server 必须因 `jobId` 不匹配或任务已完成而丢弃。
11. 收到有效 `result` 后，server 在事务中写 submission、submission_cases，并把 `judge_tasks.status` 改为 `DONE`。
12. server 处理 job 发生不可恢复基础设施错误时，server 写最终 SE、cases 空，并把 `judge_tasks.status` 改为 `FAILED`，首版不自动重试；Agent 返回含 case 的 SE 属于有效评测结果，任务状态为 `DONE`。
13. server 进程重启后，内存态丢失；启动扫描 `RUNNING` 且 `lockedUntil <= now` 的任务并重新调度，未过期的 RUNNING 等到过期后再回收。

状态映射：

- `judge_tasks.WAITING` 对应 `submissions.WAITING`。
- `judge_tasks.RUNNING` 对应 `submissions.JUDGING`。
- `judge_tasks.DONE` 对应 `submissions` 的最终非运行态。
- `judge_tasks.FAILED` 对应 `submissions.SE`，且 cases 为空。

## 题目文件下发

Agent 不直连 S3，也不接收文件列表。server 内部负责 list、校验、计算唯一 hash，并按需打包。

server 内部行为：

1. list `problems/{problemId}/`。
2. 排除 `assets/`。
3. 校验 `data/` 完整性；如果存在根目录 Dockerfile，则校验其依赖的评测资源。
4. 如果不存在根目录 Dockerfile，生成内置默认 OI trim checker Dockerfile 和辅助文件，形成归一化构建上下文。
5. 基于归一化后的评测相关文件计算唯一 `problemBundleHash`。

`problemBundleHash`：

- 由 server 计算，对 Agent 是不透明的唯一内容指纹。
- 参与输入必须覆盖归一化构建上下文中所有评测相关文件的路径和内容变化。
- `assets/` 不参与计算。
- 哈希算法固定为 SHA-256，输出小写 hex。
- 每个文件条目固定计算 `contentHash = sha256(file bytes)`。
- `metadataHash` 固定为 `sha256("mode=" + mode + "\ntemplateVersion=" + templateVersion + "\n")`；S3 对象的 etag、lastModified、contentType、user metadata 不参与 `metadataHash`。
- `mode` 固定为 `file`；内置生成文件的 `templateVersion` 固定写入对应模板版本，普通 S3 文件的 `templateVersion` 为空字符串。
- server 内部按 `path` 字典序升序排序后，将每个条目按 `path\nsize\ncontentHash\nmetadataHash\n` 拼接，再对整体文本做 SHA-256 得到 `problemBundleHash`。
- `path` 使用归一化后的 POSIX 相对路径，禁止前导 `/`，禁止 `.` 和 `..`。
- 无 Dockerfile 题目的内置默认 checker 模板版本必须通过 `metadataHash.templateVersion=oi-trim-v1` 参与计算。
- PE checker 预设写入 S3 后就是普通题目文件，`metadataHash.templateVersion` 固定为空字符串；`pe-checker-v1` 必须作为 `checker.cc` 文件内容中的常量参与 content hash。
- 如果对象存储只提供 multipart etag，server 不能直接假设它等价于内容 hash；必要时在上传时自行计算内容 hash 并保存到对象 metadata。

Agent 下载策略：

- server 在 `run` 消息中下发 `problemBundleHash`、case 列表、limits 和 `run.payload.submission`，不立即发送题目评测文件包。
- Agent 自行检查本地 A 镜像 metadata 中是否已有同一个 `problemBundleHash`。
- A 镜像命中时，Agent 不请求题目评测文件包，直接使用缓存 A 镜像评测。
- A 镜像未命中时，Agent 再向 server 单独请求题目评测文件包。
- Agent 不需要知道 S3 endpoint、bucket、etag、文件列表、题目原始是否有 Dockerfile，只和 server 交互。

题目评测文件包格式：

- 固定使用 `tgz`。
- Agent 优先把 `tgz` 作为 Docker build context 传入；如果当前 Docker API 封装不直接接受 gzip tar stream，则解压到临时目录后 build。
- 无论是否解压，A 镜像 build 完成后都必须删除下载包和解压目录。

## 题目完整性

所有题目都必须有完整 `data/`，并且在交给 Agent 前都被 server 归一化为同一种 A/B 评测模型。

- 无 Dockerfile：server 在打包时补入内置默认 OI trim checker Dockerfile。
- PE：管理端按钮向题目根目录写入预设 PE checker `Dockerfile` 和 `checker.cc`；之后仍走同一套 A/B 模型。
- 自定义 Dockerfile：使用题目根目录 Dockerfile 和评测资源。
- Agent 不感知无 Dockerfile、默认 OI、PE、SPJ、interactor 这些题目原始形态；Agent 只看 `problemBundleHash`、case 列表、A 镜像和 `run.payload.submission` 中的 B 构建上下文。
- 不能认为交互题只有一个 case；case 由 data 决定。
- 如果 data 不完整，管理端上传/编辑时就应暴露，不应允许普通用户提交后才发现。

宽松命名规则：

- 输入文件：文件名包含 `in` 且包含数字。
- 输出文件：文件名包含 `out` 或 `ans` 且包含数字。
- 从文件名提取数字作为 caseNo，固定取第一个数字串并转数字。
- 每个 caseNo 必须恰好一个输入和一个输出。
- 同 caseNo 多输入/多输出、缺输入/缺输出都是题目配置错误。

## 统一 A/B 评测模型

server 必须保证发给 Agent 的题目评测文件包永远能构建 A 镜像。

默认 OI：

- 题目源资产没有根目录 `Dockerfile` 时，server 不把这个差异暴露给 Agent。
- server 在临时打包阶段补入内置默认 OI trim checker Dockerfile，模板版本固定 `oi-trim-v1`。
- 默认 checker 读取 bake 进 A 镜像的 `data/`，与 B 的输出做 trim 比较。
- 默认 checker 固定执行字节级 trim 比较，不做 token 比较。
- 比较前分别规范化答案文件和 B stdout：先把 CRLF 与单独 CR 都转换为 LF；再按 LF 分行；每行只移除行尾 ASCII space `0x20` 和 tab `0x09`；只删除末尾连续空行；中间空行和行首空白全部保留；最后用单个 LF 连接剩余行且不追加结尾 LF。
- 除 CR/LF、行尾 space/tab 和尾部空行外，其他字节必须完全一致才 AC；不一致 WA。
- B stderr 不参与默认 OI 比较，但仍计入 output limit。
- 默认 checker 不产生 PE。
- 内置默认 checker 模板版本必须参与 `problemBundleHash`，避免 checker 逻辑升级后错误命中旧 A 镜像。

PE 支持：

- PE 是 ICPC 风格概念，不作为默认模式。
- 管理端提供“一键启用 PE checker”。
- 点击后向题目根目录写入预设 `Dockerfile` 和 `checker.cc`，模板版本固定 `pe-checker-v1`，前端给简短说明。
- 写入后它和普通自定义 Dockerfile 没有 Agent 侧区别。

内置模板产物：

- 默认 OI 和 PE 预设都固定生成两个文件：`Dockerfile` 和 `checker.cc`。
- 内置 checker 不允许使用脚本语言实现；固定使用 C++ 编译产物。
- 这两个文件的字节内容参与 `problemBundleHash`；实现不能自行换语言、换基础镜像或改文件名，除非同步升级模板版本。
- 默认 OI 与 PE 共用固定 Dockerfile：

```Dockerfile
FROM gcc:14
WORKDIR /judge
COPY data /judge/data
COPY checker.cc /judge/checker.cc
RUN g++ -std=c++20 -O2 -pipe -static -s -o /judge/checker /judge/checker.cc
CMD ["/judge/checker"]
```

- 默认 OI `checker.cc` 固定内容：

```cpp
#include <cstdlib>
#include <fstream>
#include <iostream>
#include <sstream>
#include <string>
#include <vector>

static constexpr const char* DOJ_CHECKER_TEMPLATE_VERSION = "oi-trim-v1";

static std::string readFile(const char* path) {
  std::ifstream in(path, std::ios::binary);
  std::ostringstream ss;
  ss << in.rdbuf();
  return ss.str();
}

static std::string normalize(std::string data) {
  std::string lf;
  lf.reserve(data.size());
  for (size_t i = 0; i < data.size(); ++i) {
    if (data[i] == '\r') {
      if (i + 1 < data.size() && data[i + 1] == '\n') ++i;
      lf.push_back('\n');
    } else {
      lf.push_back(data[i]);
    }
  }

  std::vector<std::string> lines;
  size_t start = 0;
  while (start <= lf.size()) {
    size_t pos = lf.find('\n', start);
    std::string line = pos == std::string::npos ? lf.substr(start) : lf.substr(start, pos - start);
    while (!line.empty() && (line.back() == ' ' || line.back() == '\t')) line.pop_back();
    lines.push_back(line);
    if (pos == std::string::npos) break;
    start = pos + 1;
  }
  while (!lines.empty() && lines.back().empty()) lines.pop_back();

  std::string out;
  for (size_t i = 0; i < lines.size(); ++i) {
    if (i) out.push_back('\n');
    out += lines[i];
  }
  return out;
}

int main() {
  const char* inputPath = std::getenv("DOJ_INPUT_PATH");
  const char* answerPath = std::getenv("DOJ_ANSWER_PATH");
  if (!inputPath || !answerPath) {
    std::cerr << "missing checker environment\n";
    return 6;
  }

  std::cout << readFile(inputPath);
  std::cout.flush();

  std::ostringstream actualStream;
  actualStream << std::cin.rdbuf();
  std::string expected = readFile(answerPath);

  if (normalize(actualStream.str()) == normalize(expected)) return 0;
  std::cerr << "wrong answer\n";
  return 1;
}
```

- PE `checker.cc` 固定内容：

```cpp
#include <cctype>
#include <cstdlib>
#include <fstream>
#include <iostream>
#include <sstream>
#include <string>
#include <vector>

static constexpr const char* DOJ_CHECKER_TEMPLATE_VERSION = "pe-checker-v1";

static std::string readFile(const char* path) {
  std::ifstream in(path, std::ios::binary);
  std::ostringstream ss;
  ss << in.rdbuf();
  return ss.str();
}

static std::string normalize(std::string data) {
  std::string lf;
  lf.reserve(data.size());
  for (size_t i = 0; i < data.size(); ++i) {
    if (data[i] == '\r') {
      if (i + 1 < data.size() && data[i + 1] == '\n') ++i;
      lf.push_back('\n');
    } else {
      lf.push_back(data[i]);
    }
  }

  std::vector<std::string> lines;
  size_t start = 0;
  while (start <= lf.size()) {
    size_t pos = lf.find('\n', start);
    std::string line = pos == std::string::npos ? lf.substr(start) : lf.substr(start, pos - start);
    while (!line.empty() && (line.back() == ' ' || line.back() == '\t')) line.pop_back();
    lines.push_back(line);
    if (pos == std::string::npos) break;
    start = pos + 1;
  }
  while (!lines.empty() && lines.back().empty()) lines.pop_back();

  std::string out;
  for (size_t i = 0; i < lines.size(); ++i) {
    if (i) out.push_back('\n');
    out += lines[i];
  }
  return out;
}

static std::vector<std::string> tokens(const std::string& data) {
  std::vector<std::string> result;
  std::string current;
  for (unsigned char ch : data) {
    if (std::isspace(ch)) {
      if (!current.empty()) {
        result.push_back(current);
        current.clear();
      }
    } else {
      current.push_back(static_cast<char>(ch));
    }
  }
  if (!current.empty()) result.push_back(current);
  return result;
}

int main() {
  const char* inputPath = std::getenv("DOJ_INPUT_PATH");
  const char* answerPath = std::getenv("DOJ_ANSWER_PATH");
  if (!inputPath || !answerPath) {
    std::cerr << "missing checker environment\n";
    return 6;
  }

  std::cout << readFile(inputPath);
  std::cout.flush();

  std::ostringstream actualStream;
  actualStream << std::cin.rdbuf();
  std::string actual = actualStream.str();
  std::string expected = readFile(answerPath);

  if (normalize(actual) == normalize(expected)) return 0;
  if (tokens(actual) == tokens(expected)) {
    std::cerr << "presentation error\n";
    return 2;
  }
  std::cerr << "wrong answer\n";
  return 1;
}
```

自定义评测：

- 存在根目录 `Dockerfile` 时，server 使用题目 Dockerfile 和评测资源构建 A 镜像。
- 自定义 Dockerfile 必须把 `data/` 复制到镜像内 `/judge/data`，或保证运行时 `/judge/{DOJ_INPUT_PATH}` 和 `/judge/{DOJ_ANSWER_PATH}` 可读；server/Agent 不会额外挂载 data。
- 自定义 Dockerfile 的运行阶段必须以 `/judge` 为工作目录，或自行处理相对路径；推荐显式 `WORKDIR /judge`。
- Special Judge、交互题、PE checker 预设都属于同一类。

统一规则：

- 仍以 `data/` 决定 case 数。
- A 是题目容器，B 是提交容器。
- A 容器运行时固定工作目录为 `/judge`。
- A 容器每个 case 都通过环境变量接收 case 选择器：`DOJ_CASE_NO`、`DOJ_INPUT_PATH`、`DOJ_ANSWER_PATH`、`DOJ_TIME_LIMIT_MS`、`DOJ_MEMORY_LIMIT_BYTES`。
- `DOJ_INPUT_PATH` 和 `DOJ_ANSWER_PATH` 是 `/judge` 下 POSIX 相对路径，例如 `data/in1.txt` 和 `data/ans1.txt`；A 程序必须自行读取这些文件。
- A 容器 stdin 固定连接 B stdout，A stdout 固定连接 B stdin，A stderr 固定作为 case message。
- 默认 OI/PE checker 的行为是先把 `DOJ_INPUT_PATH` 文件原样写入 stdout 并关闭写端，再从 stdin 读取 B stdout 到 EOF，最后读取 `DOJ_ANSWER_PATH` 判定。
- 交互题 A 程序可以在 stdin/stdout 上与 B 多轮交互，但仍必须通过 env 读取 `DOJ_CASE_NO`、`DOJ_INPUT_PATH` 和 `DOJ_ANSWER_PATH`。
- A/B 都禁网，都有资源限制。
- B 的 CPU time 限制使用题目 `timeLimitMs`，内存限制使用题目 `memoryLimitBytes`。
- A 的 CPU time 限制固定为 `max(2 * timeLimitMs, timeLimitMs + 1000ms)`，内存限制固定为 `max(2 * memoryLimitBytes, memoryLimitBytes + 128MiB)`。
- A 镜像 build wall-clock timeout 固定 300 秒；B 镜像 build wall-clock timeout 固定 60 秒。
- A/B 单容器 PID 上限固定 128。
- A/B 每个容器各自的 stdout+stderr 合计输出上限固定 64 MiB；B 超限判 OLE，A 超限判 SE，A 也可用 exit code 5 明确返回 OLE。
- 每个 case 启动一组新的 A/B 容器，共享一个临时 volume。
- A/B 通过两个 FIFO 通信：A stdout -> `a2b` -> B stdin，B stdout -> `b2a` -> A stdin。
- 启动顺序固定为先创建 FIFO，再启动 A，再启动 B；二者并发运行，任一方超时/异常时结束该 case。
- A 负责默认 OI 输入输出、PE、自定义 checker 和 interactor；B 只运行用户程序。
- A 不自主产出 score，Agent 也不产出 score；score 由 server 按 case status 统计。
- verdict 使用 exit code + stderr。

A 容器 exit code 固定映射：

- `0`：AC。
- `1`：WA。
- `2`：PE。
- `3`：TLE 或 checker 判定超时类错误。
- `4`：MLE。
- `5`：OLE。
- 其他非 0：SE。

B 容器判定优先级：

- B build 失败为 CE。
- B 运行超时为 TLE。
- B OOM 或 exit 137 为 MLE。
- B stdout/stderr 超过限制为 OLE。
- B 非 0 退出为 RE。
- B 正常退出后，以 A 的 exit code 判定该 case。

stderr：

- 作为该 case 的 message。
- 写入 PostgreSQL 前必须按 UTF-8 安全截断到 4096 字符；截断时追加 `\n[message truncated]`。
- B build logs 作为 CE 的 submission message，写入 PostgreSQL 前同样按 4096 字符截断。
- WebSocket progress message 同样最多 4096 字符，不能推送完整 64 MiB 输出。
- 对普通用户是否展示由权限和题目隐藏策略决定。

分数：

- server 是分数计算唯一事实源；Agent 返回的 `JudgeResult` 不包含 score。
- server 先按 `caseNo` 升序排序 `JudgeCaseResult`。
- 每个 case AC 得该 case 分。
- 非 AC case 得 0 分。
- score 是整数。题目总分固定为 100，首版不做每 case 自定义权重。
- case 均分时，`base=floor(100/caseCount)`，前 `100-base*caseCount` 个 case 各多 1 分，确保总和等于 100。
- 最终 status 只有所有 case AC 才是 AC；否则取 `caseNo` 升序中的第一个非 AC status。
- 若 Agent 返回 CE，server 固定写 `score=0`、cases 为空、submission status CE。
- 若基础设施错误导致整 job 无法形成有效 case result，server 固定写 `score=0`、cases 为空、submission status SE，并把 `judge_tasks.status` 置为 `FAILED`。
- 若某个 case 的 A 容器以未知错误退出并映射为 SE，这是有效评测结果；server 写入该 case SE，按 `caseNo` 聚合最终 submission status，`judge_tasks.status` 置为 `DONE`。

## Agent 内部 Runner 沙箱

B 容器：

- 禁网。
- 非 root。
- 只读 rootfs。
- drop capabilities。
- `no-new-privileges`。
- PID 限制。
- 内存限制。
- CPU 限制。
- stdout/stderr 输出限制。
- tmpfs 工作目录。

A 容器：

- 同样禁网。
- 同样需要 CPU/memory/PID/output 限制。
- 资源可比 B 宽，但必须有默认值。
- A 也使用非 root、drop caps 和 no-new-privileges；如果题目 Dockerfile 需要 root 权限，应在构建阶段完成，运行阶段仍按受限用户执行。

时间：

- 使用 CPU time 判断 TLE。
- B 的 CPU time 超过 `timeLimitMs` 判 TLE。
- A 的 CPU time 超过 `max(2 * timeLimitMs, timeLimitMs + 1000ms)` 判 SE，除非 A 自行以 exit code 3 返回 TLE。
- wall-clock safety cap 固定使用 `JudgeLimit.wallClockLimitMs`。
- 任一容器触发 wall-clock safety cap 时结束该 case；B 未结束判 TLE，只有 A 阻塞且 B 已正常结束时判 SE。

内存：

- 使用 Docker stats 峰值。
- OOMKilled 或 exit 137 判 MLE。
- Docker Desktop/Colima 可能无法精确读取极短峰值，文档和测试需记录这个限制。

## Progress 和浏览器 WebSocket

目标：

- v3 有实时进度，新版本必须保留。
- 前端通过 WebSocket 接收 per-case progress。
- PG 不保存 live progress。

固定实现：

- 浏览器连 server `/api/ws`，使用登录 token 鉴权；实现固定读取 `Sec-WebSocket-Protocol: doj-auth.<token>`，不使用 query string token。
- 用户订阅具体 submission id 或当前用户提交 feed。
- server 在内存/Redis 保存短 TTL progress。
- 断线重连后前端先 HTTP 拉 submission detail，再恢复 WS 订阅。
- Redis progress TTL 固定 10 分钟。
- 评测完成后主要读 PG final cases。

权限：

- 用户只能订阅自己有权查看的 submission。
- 管理员可订阅任意 submission。
- 比赛封榜/隐藏信息仍由 server 裁剪。

## 缓存与临时文件

题目评测文件包：

- 题目评测文件包由 Agent 按需向 server 请求，固定格式为 `tgz`。包内必须包含 `data/`、A 容器 Dockerfile 和评测资源，但不包含 `assets/`。
- Agent 只把题目评测文件包作为本次 job 的临时 build/run 输入。能直接作为 Docker build context 传入时不必落盘解压；需要解压时只解压到临时工作目录。
- A 镜像 build 超过 `aBuildTimeoutMs=300000` 时视为系统错误，server 最终写 submission SE、cases 空、score 0，`judge_tasks.status=FAILED`。
- 题目评测文件包不做持久缓存，不占用 `AGENT_CACHE_GB`。
- A 镜像 build 完成后立即删除下载得到的 `tgz` 和解包目录。
- 每个 job 结束后清理工作目录、临时容器和 volume。

A 镜像缓存：

- 只缓存 A，不缓存 B。
- B 是用户源码镜像，不缓存。
- A 镜像 cache key 使用 `problemBundleHash`。
- 上限由 `AGENT_CACHE_GB` 控制。
- `0` 表示无限制。
- 未配置时按 `min(4GB, 0.5 * availableDiskGB)` 估算，其中 `availableDiskGB` 是 Agent 可用于 Docker 镜像/缓存目录的可用磁盘空间。
- LRU 清理，优先删除最久未使用的 A 镜像。
- 必须把 `problemBundleHash` 写入 A 镜像 Docker image label，用于后续命中判断。
- Agent 本地 metadata 记录 `lastUsedAt` 和 `sizeEstimate`，用于 LRU 清理。

## 语言镜像契约

每种语言保存 `source` 文件名和 Dockerfile。server 准备 `run.payload.submission` 时：

- build context 根目录只包含用户源码文件和语言 Dockerfile。
- 用户源码文件名必须等于语言的 `source`，例如 `cpp` 使用 `main.cc`。
- Dockerfile 必须在 build 阶段完成编译，并通过 `CMD` 或 `ENTRYPOINT` 定义运行入口。
- B 容器运行时不传额外 command，只执行镜像默认 `CMD`/`ENTRYPOINT`。
- Docker build B 失败即 CE，build logs 作为 message，cases 为空，score 为 0。
- Docker build B 超过 `bBuildTimeoutMs` 也按 CE 处理，message 写 build timeout。
- B 运行阶段非 0 退出是 RE，不是 CE。
- 语言 Dockerfile 中不得依赖网络下载；首版不做在线依赖安装。

## Status

数据库中的 submission status 固定为：

- `WAITING`
- `JUDGING`
- `AC`
- `WA`
- `PE`
- `TLE`
- `MLE`
- `OLE`
- `RE`
- `CE`
- `SE`

说明：

- `FROZEN` 不是数据库 status，只是比赛查询/展示态。
- PE 只在 PE checker/custom checker 中出现。
- 部分分提交不可能是 AC。
- solved 只看最终 `status='AC'`。
- OLE 来自 B stdout/stderr 超过 output limit，或 A 明确以 exit code 5 判定。
- 最终 status 聚合规则是“第一个非 AC case status”，全 AC 才 AC。
- OI 使用 `score` 排名。
- 作业按 AC/完成率展示，可附带 best score。

## 统计修复

首版不提供 rejudge 功能，也不提供 rejudge API；只需要考虑 Redis 被 clear 后的统计修复。

- cron fix 定期从 submissions 修复 Redis solved/ranking/count。
- 管理页展示上次修复时间、状态、耗时。
- 管理端可手动触发统计修复。
- 未来增加 rejudge 时，必须先补 API 契约，再定义重判完成后的局部 repair。
- 统计可以不强实时，优先节省算力。
