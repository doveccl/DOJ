# services 规则

- `services/` 是 server 侧接口实现层。
- `web/` 放游客和普通用户 API。
- `admin/` 放管理员 API。
- `judger/` 放 server 提供给 judger 的 API。
- 管理员能力优先贴近对应业务页面，例如题目、作业、比赛、讨论。
- judger API 负责任务领取、题包交付、结果回传和心跳，结果写入必须校验 task attempt。
- 提交创建、judger 领取任务、judger 回传结果等会改变提交可见状态的路径，都要通过 `services/events` 广播轻量事件供 `/api/events` SSE 刷新前端。
- 本地 loopback judger 可无 token 快速开发；远程 judger 使用管理页配置的 auth。管理页删除 judger 后不再保留可见状态。
- lease 使用长轮询；没有任务时由 server 等待一小段时间，judger 客户端不要再做秒级空转轮询。
- lease 必须使用锁或等价机制避免同一任务被多个 judger 同时领取。
- result 写入必须同时校验 task id、submission id 和 attempt；旧 attempt 的回包不能覆盖 submission，也不能删除新 attempt 的 case 结果。
- 题目数据、附件、checker/interactor 等对象都应由 server 从对象存储读取后交付给 judger，judger 不直连数据库或对象存储。
