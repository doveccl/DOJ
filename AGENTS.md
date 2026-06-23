# DOJ AI 规则

## 项目方向

- 前端使用 React + Vite + TypeScript + antd。
- 后端使用 Go + Echo + GORM + OpenAPI。
- 评测机使用 Go 实现，包含宿主侧 judger 和容器内 runner。
- 数据库使用 PostgreSQL；缓存/轻量状态使用 Valkey；对象存储使用 S3/MinIO。

## 当前目录约定

- `cmd/` 只放 `server.go`、`judger.go`、`runner.go` 三个入口。
- `api/` 放 OpenAPI 契约。
- `models/` 放 GORM 模型和数据库读写。
- `middleware/` 放 Echo 中间件。
- `services/` 放 server 侧接口实现，分 `web/`、`admin/`、`judger/`。
- `judger/` 放所有评测机相关代码，包含 runner。
- `utils/` 只放小工具和薄封装。
- `index.html` 是 Vite HTML 入口；`web/` 放前端源码。

## 后端约定

- `api/web.yaml` 是 web/admin API 契约，用于生成前端 TypeScript schema/client，并约束 Go Echo handler 实现。
- OpenAPI 采用小步 contract-first：先写当前功能所需接口，再实现 Go 和前端。
- `services/judger` 是 server 提供给 judger 的接口实现。
- 第一版不强制抽象业务 service；handler 可以直接调用 GORM、utils 和局部 helper，逻辑变厚后再抽。
- 配置保持显式，真实密钥、私有地址和私有路径留在本地环境。

## 设计硬约束

- judger 不直连 PostgreSQL、Valkey、S3/MinIO；judger 只访问 server 的 judger API。
- 题目第一版不做 problem version。
- 标签作为题目的字段保存，不单独建标签表。
- 管理员才能创建的题目、作业、比赛，默认不存 `created_by` 字段，除非后续出现审计需求。
- S3 路径能从业务 id 推导时，不在数据库重复保存路径。
- 题目数据在 S3 中按文件展开保存，不同时持久保存 zip 和未压缩两份。
- 提交源码第一版存数据库；评测大日志或大附件以后再决定是否进 S3。
- 第一版先用 GORM 模型表达表结构。
- 题目内存限制字段使用 MB；提交结果、case 结果和 judger 内部资源限制继续使用 KB。
- 题目编号从 1000 开始；不要让真实数据库自增回到 1。
- 快速开发期按最后一次确定的字段、表名和接口实现；不要保留旧口径兼容分支。开发库出现脏数据时优先清理或重置数据。
- 表名和字段名尽量短；单词能表达清楚时避免双单词组合名。
- 代码文件名也尽量短；单词能表达清楚时避免双单词组合名，但不能牺牲语义清晰度。
- 管理员专属创建对象默认不存 `created_by`，除非功能需要审计或展示。

## 前端硬约束

- 使用 antd 原生组件优先，避免手搓已有组件能力
- API/server state 使用 TanStack Query；不要为同一份服务端数据再维护一套独立 `useState` 缓存。
- 管理页用户统一叫“用户”，用户组统一叫“用户组”；不要混用“成员”。
- 全局公告只在首页就地编辑，管理设置页不要展示公告 textarea。
- 写 antd 代码前必要时查阅 https://ant.design/llms.txt
- Vitest 使用要克制，只测稳定纯逻辑或关键组件，不替代浏览器走查
- 页面走查优先使用 IDE 内置浏览器。若 Codex Desktop 浏览器桥自身故障导致无法执行，例如 `node_repl/js` 在用户代码执行前报 `sandboxPolicy` 元数据错误，先尝试修复本地 Codex 配置；仍不可用时使用用户 Chrome 作为第二选择。除非明确需要，不要为了走查在项目里安装 Playwright。

## 质量门禁

- 涉及权限、可见性、分配关系、提交归属、统计口径的改动，必须先写清楚业务不变量，并补后端回归测试。
- 作业等有分配关系的功能，必须同时覆盖游客、未分配普通用户、已分配普通用户、管理员四种视角。
- 涉及列表、详情、首页、提交、统计任意一处的上下文逻辑，必须主动检查同一实体的其他入口是否使用同一口径。
- 提交归属类统计必须基于提交记录上的上下文字段，例如 `assignment_id` / `contest_id`，不能只凭题目属于某集合倒推。
- 异步详情编辑弹窗必须在详情数据就绪后挂载表单，或使用稳定 `key` 重建表单；不要让空数据先触发 `initialValues`。
