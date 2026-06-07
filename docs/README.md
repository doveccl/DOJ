# DOJ 产品与系统文档

`docs/` 是面向 AI 和产品/技术讨论的 canonical 文档集。目标是：即使新开一个 AI 对话、没有任何代码上下文，也能仅凭这些文档复刻 DOJ 的产品形态、数据模型、评测内核、API 边界和 UI 页面。

根目录 `README.md` 仍是面向人的项目使用说明；本目录不是安装手册，而是 PRD + 系统设计 + 用户手册。

## 阅读顺序

1. `01-product-prd.md`：产品定位、角色、核心流程、功能边界。
2. `02-data-and-storage.md`：PostgreSQL、S3、Redis、运行配置和完整目标 schema。
3. `03-judge-core.md`：评测内核、server/agent 协议、题目资产、Runner、缓存和状态规则。
4. `04-api-and-permissions.md`：API 契约、权限裁剪、WebSocket、settings、错误和分页规范。
5. `05-ui-manual.md`：页面信息架构、分页、展示内容、管理端交互和 Markdown 编辑器要求。
6. `06-implementation-plan.md`：一次性改造计划、验收标准、smoke 策略和不做事项。

## 当前硬性结论

- 当前分支仍是正式发布前的早期重写分支，可以重置数据库迁移历史。
- 技术栈固定为 Bun workspace、Hono server、Drizzle ORM、PostgreSQL、Redis/内存 fallback、S3/MinIO、Docker API、Vite + Vue 3 + Naive UI。
- 这一版采用合并 `server` 形态：API、评测调度、浏览器 WebSocket、Agent WebSocket 都属于同一个后端服务包/进程职责。不要保留“单 Worker 但又独立 Worker 包”的中间态。
- Agent 只连接 server，不直连 PostgreSQL、Redis 或 S3。题目评测文件由 server 从 S3 读取；server 先给 Agent 唯一 `problemBundleHash`，Agent 按需请求 `tgz` 题目评测文件包。
- PostgreSQL 只保存核心业务事实。文件索引、Agent registry、solved/count 缓存不进 PostgreSQL。
- S3 只保存题目资产：`problems/{problemId}/{path}`。没有全局 media 概念。
- Redis 保存 session、限流、WebSocket progress、solved/ranking/count 派生状态和短期缓存；Redis 状态必须可以从 PostgreSQL 重建。
- 默认评测没有 Dockerfile 时，由 server 在打包阶段补入内置 OI trim checker Dockerfile；如果管理员需要 PE，可在题目资产界面点击按钮写入预设 PE checker Dockerfile。Agent 不感知默认 OI/PE/SPJ/interactor，统一执行 A/B 模型。
- 自定义评测使用题目根目录 `Dockerfile`，verdict 通过 exit code + stderr 返回；Agent 只上报 case status 和资源，分数由 server 按 case AC 情况统计，题目容器不自主产出 score。
- 页面默认分页大小统一为 50。
- 代码文件名尽量不要使用连字符 `-`；新文件优先使用 camelCase 或简洁单词命名。
- Runner 作为概念保留，但实现归属 `apps/agent` 内部模块，不再作为长期独立 workspace package。
- 所有 Markdown 编辑场景固定使用 `md-editor-v3`；展示态不能像 editor。
- 当前不保留预写模块级 `AGENTS.md`；后续按模块实施改造时，尽量为每个正在落地的模块生成或更新局部 `AGENTS.md`，只记录该模块当前实现注意事项，并指向本 docs 作为权威来源。

## 术语

- `server`：合并后的后端服务，包含 API、评测调度、浏览器 WebSocket、Agent WebSocket。
- `Agent`：独立部署的评测机进程，只连接 server，在本机 Docker 执行评测。
- `Runner`：Agent 内部的 Docker 执行库，目标实现位置在 `apps/agent/src/runner/`。
- `题目资产`：S3 中某个题目下的所有文件，包括 `data/`、`assets/`、`Dockerfile` 和评测辅助文件。
- `assets/`：题面展示图片和附件，仅展示用途，不参与评测。
- `data/`：评测输入输出文件，所有评测模式都必须有完整 data。
- `评测资源`：题目根目录下参与自定义评测构建的 `Dockerfile`、checker/interactor 源码等文件。

## 不做事项

- 不做全局 media 文件库。
- 不做站点 logo 上传。
- 不做注册邀请码。
- 不做 Agent labels。
- 不做 Agent 直连 S3。
- 不做多 server/多 worker 调度，除非后续专门设计共享 registry/lease。
- 不做题目版本化。
- 不做站内通知、讨论审核、讨论编辑历史。
- 不做每题分值配置，题目满分预期为 100。
