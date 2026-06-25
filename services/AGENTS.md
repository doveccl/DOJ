# services 规则

- `services/` 是 server 侧接口实现层。
- `web/` 放游客和普通用户 API。
- `admin/` 放管理员 API。
- `judger/` 放 server 提供给 judger 的 API。
- 管理员能力优先贴近对应业务页面，例如题目、作业、比赛、讨论。
- 讨论 tags 是软关联文本，不因为题目隐藏、删除或赛前不可见而强制隐藏讨论、禁止发帖或禁止评论。
- 比赛题目可见性是派生状态：赛前题库和题目详情都不可见，赛中题库不可见但题目详情可见，赛后回到题目自身 visible 口径。
- 比赛榜单必须按赛制计算：OI 按每题最高分求和，ICPC 按首 AC 数和罚时；封榜后的非管理员榜单和比赛提交列表只计算 freezeAt 前提交。
- judger API 负责任务领取、题包交付、结果回传和心跳，结果写入必须校验 task attempt。
- task lease 属于提交执行状态，保存在 `submissions.judger_id` / `submissions.lease_until` / `submissions.attempt`；不要再引入独立 tasks 表或 Redis lease。
- judger 在线态、连接时间、活跃时间和在线时长是运行态，走统一 Redis/Valkey 缓存源；`REDIS` 为空时连接本地默认 Redis，不要双写双读，也不要加到 `judgers` 表。
- 提交创建、judger 领取任务、judger 回传结果等会改变提交可见状态的路径，都要通过 `services/events` 广播轻量事件供 `/api/events` SSE 刷新前端。
- 直连 loopback judger 可无 token，用于单机部署和本地开发；带 `Forwarded` / `X-Forwarded-*` / `X-Real-IP` 的反代请求不能走本地免 token。远程 judger 使用管理页配置的 auth。管理页删除 judger 后不再保留可见状态。
- lease 使用长轮询；没有任务时由 server 等待一小段时间，judger 客户端不要再做秒级空转轮询。
- lease 必须使用锁或等价机制避免同一任务被多个 judger 同时领取。
- result 写入必须同时校验 task id、submission id 和 attempt；旧 attempt 的回包不能覆盖 submission，也不能删除新 attempt 的 case 结果。
- 题目数据、附件、checker/interactor 等对象都应由 server 从对象存储读取后交付给 judger，judger 不直连数据库或对象存储。
