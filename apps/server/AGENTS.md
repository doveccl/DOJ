# Server AI 规则

## 范围

本文件约束 `apps/server/`。这里是合并 server：HTTP API、评测调度、浏览器 WebSocket、Agent WebSocket 都属于同一服务进程职责。

## 总体原则

- HTTP API 返回 shape 要稳定、简洁、领域化。
- 服务端裁剪敏感字段；前端隐藏不是安全边界。
- 管理权限统一使用 `users.admin`。
- 输入校验使用 Zod；错误响应使用统一 `{ error: { code, message, issues? } }`。
- 列表分页参数固定为 `page`、`pageSize`，默认 50，上限 100，响应固定包含 `items`、`page`、`pageSize`、`total`。
- 时间戳输出 ISO 8601 UTC 字符串，字段名以 `At` 结尾；heatmap 可以按 `tz` 聚合成本地日期桶。
- 普通用户访问隐藏、软删除或无权资源时返回 404，避免暴露存在性；管理员接口可返回更具体配置错误。
- `429` 必须带 `Retry-After`，错误 code 固定 `RATE_LIMITED`；Zod 错误 code 固定 `VALIDATION_ERROR`。

## Auth 与设置

- 浏览器 HTTP API 使用 `Authorization: Bearer <token>`；浏览器 WebSocket 使用同一 token，放在 `Sec-WebSocket-Protocol: doj-auth.<token>`。
- session token 为 32 字节随机数 base64url；Redis key 为 `session:{token}`，TTL 30 天并滑动续期。
- 注册默认关闭；开放注册必须配置 SMTP 和邮箱验证码。
- 注册和修改邮箱验证码为 6 位数字，有效期 10 分钟，重发间隔 60 秒。
- 禁用用户不能登录；已签发 session 在下一次鉴权时失效。
- `mustChangePassword=true` 时，除 self 读取和修改密码外，其他登录态接口返回 403。
- settings 分区固定为 `general`、`smtp`、`ai`；`_` 开头私有字段读接口只返回是否已设置。

## 业务边界

- 题目正文存 S3 `problems/{problemId}/statement.md`，不进 PostgreSQL。
- 题目资产只管理 `data/`、`assets/` 和 custom 根目录评测资源；公开 assets 只允许普通用户读取 `assets/`，不暴露 `data/` 和根目录。
- `GET /api/problems` 对普通用户和访客只返回可见且未删除题目；管理员可读隐藏/删除题。
- `POST /api/submissions` 只允许提交可见且未删除题目，管理员也遵守。
- server 按当前时间窗口和范围自动写入 submission 的 `contestId`、`assignmentId`，二者可以同时存在。
- 提交源码只对本人、管理员或 public 可见；比赛/作业裁剪规则必须在服务端执行。
- 作业只有 `endAt`，没有 late/allowLate；截止前可编辑结构字段，截止后只允许标题、描述等非结构字段。
- 比赛时间窗口固定 `[startAt, endAt)`；OI 使用最后一次提交分数，ICPC 使用通过数和罚时。
- 讨论区使用 `topics` + `posts`；创建 topic 时事务性创建首楼 post，回复时更新 `topics.updatedAt`。

## 评测调度

- 创建提交时写 `submissions(status='WAITING')` 并入队 `judge_tasks`。
- claim 任务时把 submission 改为 `JUDGING`，租约 60 秒，执行中每 20 秒续租。
- 可用 Agent 条件：WebSocket 在线、30 秒内 heartbeat、`activeJobs < concurrency`。
- Agent 选择规则：负载比最小，再按 heartbeat 最早，再按 key 字典序。
- Agent 断开或 heartbeat 超时后停止续租；租约过期后任务可重新调度。
- 收到有效 result 后事务性写 submission、submission_cases，并把 judge task 标记 DONE。
- 基础设施不可恢复错误写最终 `SE`，首版不自动重试，不向用户展示 `SE -> JUDGING` 跳变。
