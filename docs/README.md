# DOJ 产品与系统文档

`docs/` 是面向 AI 和产品/技术讨论的 canonical 文档集。目标是：即使新开一个 AI 对话、没有任何代码上下文，也能仅凭这些文档复刻 DOJ 的产品形态、数据模型、评测内核、API 边界和 UI 页面。

根目录 `README.md` 仍是面向人的项目使用说明；本目录不是安装手册，而是 PRD + 系统设计 + 用户手册。

## 文档结构

四个集中入口，按需查阅，不再拆成大量小文件：

1. `01-overview-and-ui.md`：产品定位、角色、全局原则，以及各页面（首页、题库、题目详情、提交、作业、比赛、排行、讨论、个人资料、管理端）的内容与职责。
2. `02-data-and-storage.md`：PostgreSQL 完整目标 schema、S3 题目资产路径约定、Redis 派生状态、固定安全参数和环境变量。
3. `03-api.md`：HTTP/WS 接口路由、请求/响应字段表、权限与裁剪规则、错误与分页规范。
4. `04-judge-core.md`：评测内核实现细节，server/Agent 协议、调度租约、题目资产下发、A/B 评测模型、沙箱限制、缓存与实时进度。

跨文档引用以上述文件名为准，保持单一事实源，不在多个文件重复维护同一结论。

## 当前硬性结论

- 当前分支仍是正式发布前的早期重写分支，可以重置数据库迁移历史，不做新旧兼容层。
- 技术栈固定为 Bun workspace、Hono server、Drizzle ORM、PostgreSQL、Redis/内存 fallback、S3/MinIO、Docker API、Vite + Vue 3 + Naive UI。
- 这一版采用合并 `server` 形态：API、评测调度、浏览器 WebSocket、Agent WebSocket 都属于同一个后端服务包/进程职责，不保留“单 Worker 但又独立 Worker 包”的中间态。
- Agent 只连接 server，不直连 PostgreSQL、Redis 或 S3；server 在 `run` 中先给 Agent 唯一 `bundleHash`，Agent 缓存未命中时再走 HTTP（`GET /api/agents/bundle/:problemId`，SECRET 鉴权）下载评测文件 tar。WebSocket 只走 JSON 控制消息，二进制走 HTTP。
- PostgreSQL 只保存核心业务事实；文件索引、Agent registry、solved/count 缓存不进 PostgreSQL。
- S3 只保存题目资产 `problems/{problemId}/{path}`，没有全局 media 概念；题面正文存 `problems/{problemId}/statement.md`，不进 PostgreSQL。
- Redis 保存 session、限流、WebSocket progress、solved/ranking/count 派生状态和短期缓存；Redis 状态必须可以从 PostgreSQL 重建。
- 评测行为由 `problems.mode = default | strict | custom` 决定：`default`/`strict` 共用预构建判题镜像 `doveccl/doj:judger`（`FROM busybox`），由 server 下发的 `CHECK` 环境变量切换 trim/PE 比较，普通题目不再生成或构建任何 A 镜像；只有 `custom` 才用题目根目录 `Dockerfile` 构建 A 镜像。Agent 不感知 `mode`，统一执行 A/B 模型。
- A/B 容器通信固定为双 FIFO + shell 重定向，由 Agent 创建并接管 `/pipe/a2b`、`/pipe/b2a`，A/B 程序心智上只是纯 stdio；A/B 镜像必须含 `/bin/sh`，不支持 `FROM scratch`。该方案经实测确定，Docker attach/exec 半关无法对容器 stdin 单独投递 EOF。
- verdict 通过 A 容器 exit code + stderr 返回；Agent 只上报 case status 和资源，分数由 server 按 case AC 情况统计，题目容器不自主产出 score。
- 页面默认分页大小统一为 50。
- 代码文件名尽量不使用连字符 `-`，新文件优先使用 camelCase 或简洁单词命名。
- Runner 作为概念保留，但实现归属 `apps/agent` 内部模块，不再作为长期独立 workspace package。
- 所有 Markdown 编辑场景固定使用 `md-editor-v3`，展示态不能像 editor。
- 当前不保留预写模块级 `AGENTS.md`；后续按模块实施改造时，再为正在落地的模块生成或更新局部 `AGENTS.md`，只记录该模块当前实现注意事项，并指向本 docs 作为权威来源。

## 术语

- `server`：合并后的后端服务，包含 API、评测调度、浏览器 WebSocket、Agent WebSocket。
- `Agent`：独立部署的评测机进程，只连接 server，在本机 Docker 执行评测。
- `Runner`：Agent 内部的 Docker 执行库，目标实现位置在 `apps/agent/src/runner/`。
- `题目资产`：S3 中某个题目下的所有文件，包括 `statement.md`、`data/`、`assets/` 以及 `custom` 题目的 `Dockerfile` 和评测辅助文件。
- `statement.md`：题面 Markdown 正文，仅根目录单文件。
- `assets/`：题面展示图片和附件，仅展示用途，不参与评测。
- `data/`：评测输入输出文件，所有评测模式都必须有完整 data。
- `评测资源`：`custom` 题目根目录下参与构建的 `Dockerfile`、checker/interactor 源码等文件。

## 实施与验收要点

文档是实现 authority；如果代码与 docs 冲突，以 docs 为准。这是早期重写分支，可以一次性破坏性改 schema 和 baseline migration，不保留旧 PG 文件索引 fallback，旧 smoke 可全部删除重写。

落地顺序建议：基础 schema 与合并 server 形态 → S3 题目资产读写与权限 → 评测链路（Agent 协议、调度租约、A/B 模型、预构建判题镜像、custom 构建、缓存）→ WebSocket progress 与 Redis 派生统计 → API 字段与服务端裁剪 → 前端页面。

核心验收：DB reset 后 schema 与 `02-data-and-storage.md` 一致且无旧表/旧字段；`default`/`strict` 题目用预构建镜像可 AC/WA/PE 且 Agent 不构建 A 镜像，`custom` 题目可按 `bundleHash` 构建并复用；多 case 混合失败时最终 status 等于第一个非 AC case，均分整数总和为 100；普通用户无法看到无权源码、隐藏题、封榜详情、非公开 assets；Agent 无 DB/Redis/S3 凭据；提交详情无需轮询即可看到 per-case progress，Redis clear 后 cron 可恢复统计；新 AI 对话只读 `docs/` 能复述产品、schema、评测、API、UI。

建议的最小 smoke 覆盖：auth（注册关闭、管理员登录、邮箱验证码）、settings 私有字段裁剪、members、problem-assets（含题面 S3 与 data 完整性）、judge-default（预构建镜像 trim AC/WA）、judge-strict（PE）、judge-custom（自定义 exit code checker/interactor）、submission-security、realtime-progress、redis-derived、assignments、contests（OI 最后提交、ICPC 罚时、封榜裁剪、真实榜单）、limits-and-hash（固定资源限制、output limit、wall-clock cap、`bundleHash`）、discussion。smoke name 与文件名保持一致。

## 不做事项

- 不做全局 media 文件库、站点 logo 上传、注册邀请码。
- 不做 Agent labels、Agent 直连 S3、B 镜像缓存。
- 不做多 server/多 worker 调度，除非后续专门设计共享 registry/lease。
- 不做题目版本化、每题分值配置（题目满分固定 100）。
- 不做站内通知、讨论审核、讨论编辑历史、讨论站内图上传。
- 不做用户取消评测，后续需要再加。
