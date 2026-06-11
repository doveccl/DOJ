# DB AI 规则

## 范围

本文件约束 `packages/db/`。这里维护 Drizzle schema、数据库 client、队列、seed 和运行时 settings 默认值。

## 存储分工

- PostgreSQL 只保存核心业务事实和评测任务。
- S3 保存题目资产：`problems/{problemId}/statement.md`、`data/`、`assets/`、custom 根目录评测资源。
- Redis 保存 session、限流、WebSocket progress、solved/ranking/count 派生状态和短期缓存；Redis 状态必须能从 PostgreSQL 重建。
- Agent registry、solved/count cache、文件索引不进 PostgreSQL。

## Schema 原则

- 当前 `main` 仍是正式发布前早期重写分支，可以重置迁移历史，不做旧 schema 兼容层。
- schema 固定维护在 `packages/db/src/schema.ts`，迁移使用 `drizzle-kit generate` 和 `drizzle-kit migrate`。
- 自增主键使用 integer identity；外键 ID 类型与被引用主键一致。
- `languages.id` 使用 `varchar(32)`；`timeMs` 用 integer；`memoryBytes` 用 bigint；`score` 用 integer。
- 时间戳使用 `timestamptz`。
- boolean 字段必须有明确默认值：`admin=false`、`mustChangePassword=false`、`visible=false`、`public=false`、`pinned=false`。
- 不保留旧表/旧字段：`files`、`problem_files`、`judge_agents`、`solved_problems`、`ai_coaching_sessions`、`legacyId`、PostgreSQL count cache、注册邀请码字段。

## 固定业务事实

- 用户 admin 权限由 `users.admin` 表达，group 只表示教学/成员分组。
- settings key 固定为 `general`、`smtp`、`ai`；`_` 开头字段是私有配置。
- languages seed 默认 `cpp`，source 为 `main.cc`，Dockerfile 使用 C++20 静态编译。
- seed 必须创建 admin 和 `P1000 A+B Problem`，并写入 P1000 的 statement 与两组 `data/`。
- submissions 保存最终状态、最终资源和上下文；live progress 不写 PostgreSQL。
- judge_tasks 是调度事实源，未完成任务对同一 `submissionId` 保持唯一。
