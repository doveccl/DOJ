# 实施计划与验收

## 总原则

- 这是正式发布前的早期重写分支，可以一次性破坏性改 schema 和 baseline migration。
- 不做新旧兼容层。
- 不保留旧 PG 文件索引 fallback。
- 旧 smoke 可以全部删除重写。
- 文档是实现 authority；如果代码与 docs 冲突，以 docs 为准。

## 第 1 阶段：基础模型和服务形态

目标：确立合并 server 和目标 schema。

任务：

- 合并 API、评测调度、浏览器 WS、Agent WS 到 server 职责。
- 重写 `packages/db/src/schema.ts`。
- 删除目标外表：files、problem_files、judge_agents、solved_problems、ai_coaching_sessions。
- 删除 legacyId、PG count cache、inline test cases、caseCount、judgeProgress、per-problem score、late/AI 作业字段。
- 重写 baseline migration。
- 重写 seed：只创建管理员、`cpp/main.cc`、`P1000 A+B Problem`；P1000 固定 `visible=true`；管理员默认 `admin/admin@example.test/admin12345`，允许部署环境变量覆盖。
- seed 同步写入 P1000 的 S3 `data/in1.txt`、`data/ans1.txt`、`data/in2.txt`、`data/ans2.txt`，确保 smoke 可直接评测 A+B。
- settings 改为 general/smtp/ai 聚合 JSON。

验收：

- DB reset 后 schema 与 `02-data-and-storage.md` 一致。
- 没有旧表和旧字段残留。
- 默认数据只有目标对象，并且 P1000 的 S3 data 完整。

## 第 2 阶段：S3 题目资产

目标：题目文件完全由 S3 路径约定承载。

任务：

- 实现 S3 list/read/write/delete/head。
- 实现路径校验。
- 实现 admin problem assets API。
- 删除 media/files API。
- 普通用户只允许 GET 公开题目的 `assets/{filename}`。
- data 上传时校验输入/输出配对。
- PE checker 按钮写入预设 `Dockerfile` 和 `checker.cc`。

验收：

- 管理员可上传 data、assets、评测资源。
- 普通用户不能 list，不能读 data 或 Dockerfile。
- 隐藏/软删除题目的 assets 不对普通用户开放。
- data 不完整时管理端能发现。

## 第 3 阶段：评测链路

目标：Agent 只连 server；server 把所有题目归一化为可构建 A 镜像的题目评测包；Runner 作为 Agent 内部模块执行统一 A/B 评测。

任务：

- Agent/server 共享 secret 使用 `SECRET`。
- 将 `packages/runner` 的实现合入 `apps/agent/src/runner/`，保留清晰模块边界，删除独立 workspace package。
- 新增或迁移代码文件名尽量不使用连字符，优先使用 camelCase。
- 删除 labels。
- 按 `03-judge-core.md` 固定实现 `JudgePayload`、`JudgeProgress`、`JudgeResult`、`AgentHello` 类型。
- B 构建上下文固定内联在 `run.payload.submission`，不新增 B 包请求消息。
- 实现 `judge_tasks.lockedUntil`、Agent heartbeat、续租、断线回收、迟到 result 丢弃和 server 重启后的 RUNNING 任务回收。
- server 从 S3 list 并 exclude assets，内部计算稳定的唯一 `problemBundleHash`，但不把文件列表下发给 Agent。
- Agent 先根据 A 镜像 metadata 判断缓存是否命中；命中则不请求题目评测文件包，未命中才向 server 请求 `tgz`。
- 删除 Agent 直连 S3 能力。
- 删除 inline cases 和 caseCount。
- server 对无 Dockerfile 题目在打包阶段补入内置默认 OI trim checker Dockerfile。
- 内置默认 OI trim checker 模板版本固定 `oi-trim-v1`，PE checker 预设模板版本固定 `pe-checker-v1`。
- 默认 OI、PE、自定义 Dockerfile 都归一化为同一套 A/B 模型；Agent 不做题目形态分支。
- A 容器按固定 env/stdin/stdout/stderr 契约读取 `DOJ_CASE_NO`、`DOJ_INPUT_PATH`、`DOJ_ANSWER_PATH` 并和 B 通信。
- `problemBundleHash` 必须按 `03-judge-core.md` 的文件条目算法计算；case 信息不作为独立字段额外加入 hash，而是由 `data/` 文件路径和内容隐含参与。
- 固定双 FIFO A/B 编排：每 case 启动 A/B，A stdout -> B stdin，B stdout -> A stdin。
- 固定语言 Dockerfile 契约：源码按 `source` 文件名放入 build context，Dockerfile 定义 CMD/ENTRYPOINT，B build 失败为 CE。
- verdict 使用 exit code + stderr。
- A/B 都禁网并按 `03-judge-core.md` 的固定 CPU/memory/PID/output/wall-clock 参数设置资源限制。
- 只缓存 A 镜像，不缓存 B。
- 实现 A 镜像缓存的 `AGENT_CACHE_GB` LRU 上限；A 镜像 metadata 写入 `problemBundleHash`；题目评测文件包 build 后立即清理，不持久缓存。

验收：

- 无 Dockerfile 的 A+B 题通过 server 补入默认 checker 后可 AC/WA，并且 A 镜像可按 `problemBundleHash` 缓存。
- data 缺失时不允许提交或给管理员配置错误。
- PE checker 预设可产生 PE。
- 自定义 checker 可通过 exit code 返回 verdict。
- 多 case 混合失败时最终 status 等于第一个非 AC case status。
- case 均分整数总和为 100。
- Agent 不需要 S3 env。
- A 镜像命中时，无论题目原始是否有 Dockerfile，Agent 都不会请求题目评测文件包。

## 第 4 阶段：实时状态和 Redis 派生

目标：WebSocket progress 和 Redis solved/ranking/count。

任务：

- 浏览器 `/api/ws`。
- submission progress 通过 WS 推送。
- Redis 保存短 TTL progress；未配置 `REDIS` 时走统一 in-memory fallback。
- 最终 cases 写 PG。
- AC 后写 Redis solved/ranking/count。
- cron fix 定期修复 Redis。
- 管理页展示上次修复时间。

验收：

- 提交详情无需轮询即可看到 per-case progress。
- Redis clear 后 cron 可恢复统计。
- 排行/题库 solved/count 不依赖 PG count cache。

## 第 5 阶段：API 和权限

目标：所有 API 使用目标字段和服务端裁剪。

任务：

- `requireAdmin()` 替代 group admin。
- auth self 返回 admin。
- 注册、邮箱修改接入邮件验证码。
- settings 私有字段裁剪。
- submissions 上下文由后端判断。
- contests/assignments 删除 per-problem score/late/AI。
- OI/ICPC 榜单公式、封榜裁剪、排行/heatmap/tags/recommended API 按 docs 固定实现；heatmap 访客固定 401，通过率除零固定 0。
- discussion 改 topics/posts。

验收：

- 前端不再依赖 groups 判断 admin。
- 普通用户无法看到无权源码、隐藏题、封榜详情、非公开 assets。
- API 字段名与 docs 一致。

## 第 6 阶段：前端

目标：UI 与 `05-ui-manual.md` 一致。

任务：

- 首页三栏 + 热力图。
- 题库 solved 图标 + 通过率。
- 接入 `unplugin-auto-import` 和 `unplugin-vue-components`，自动导入 Vue/Router/Pinia/Naive UI/本地通用组件。
- 题目详情统一页面，提交区下方整宽。
- 提交列表/详情接入 WS。
- 作业列表分页。
- 比赛列表筛选和详情 tabs。
- Profile Gravatar + 邮箱验证码。
- 管理端按新结构重做设置、成员、题目、作业、比赛、语言、评测机。
- 引入 `md-editor-v3` 作为固定 Markdown 编辑器。

验收：

- 默认分页 50。
- 常见导航、状态、操作、筛选、solved 等 UI 元素优先使用图标增强识别度。
- 管理端题目三步编辑可用。
- Markdown 展示态无 editor 痕迹。
- 讨论无站内图上传。

## 第 7 阶段：Smoke

旧 smoke 可以全部删除重写，保持克制。

固定 smoke：

- `auth`：注册关闭、管理员登录、邮箱验证码基础路径。
- `settings`：general/smtp/ai 私有字段裁剪。
- `members`：admin 字段、用户创建、组成员。
- `problem-assets`：data/assets/评测资源上传、权限、data 完整性。
- `judge-default`：server 补默认 checker 后的统一 A/B OI trim AC/WA，验证 A 镜像缓存命中不再请求 tgz。
- `judge-custom`：预设 PE checker、自定义 exit code checker。
- `submission-security`：源码和隐藏题裁剪。
- `realtime-progress`：WS progress。
- `redis-derived`：solved/ranking/count repair。
- `assignments`：groups + individual users、报告。
- `contests`：OI 最后提交、ICPC 罚时、封榜裁剪、admin 查看真实榜单。
- `limits-and-hash`：验证固定资源限制、output limit、wall-clock cap、`problemBundleHash` sha256 输入和 A 镜像 label 命中。
- `discussion`：topics/posts。

命名原则：smoke name 与文件名保持一致，不再维护额外映射。

## 不做事项

- 不做全局 media。
- 不做站点 logo。
- 不做邀请码。
- 不做题目版本化。
- 不做多 server 调度。
- 不做 Agent labels。
- 不做 Agent 直连 S3。
- 不做 B 镜像缓存。
- 不做用户取消评测，后续需要再加。
- 不做讨论站内图上传。
- 不做复杂讨论编辑历史。

## 最终验收清单

- 新 AI 对话只读 `docs/` 能复述产品、schema、评测、API、UI。
- 根 README 仍只面向人说明项目、开发、部署。
- `AGENTS.md` 只指向 docs，不保存长 done list。
- Runner 实现位于 Agent 内部模块，不再作为独立 workspace package。
- 新代码文件名尽量没有连字符。
- DB schema 无旧模型残留。
- server 合并 API/调度/WS/Agent 连接职责。
- Agent 无 DB/Redis/S3 凭据。
- 题目资产只走 `problems/{id}/...`。
- 首页、题目、提交、管理端符合 UI 手册。
- 核心 smoke 通过。
