# Shared AI 规则

## 范围

本文件约束 `packages/shared/`。这里放跨 server、web、agent 共享的协议类型、状态常量、分页、settings、storage/testdata 工具。

## 规则

- 共享类型是跨进程契约，修改前先确认 API、Agent、Web 三端调用点。
- `JudgeStatus` 固定包含 `WAITING`、`JUDGING`、`AC`、`WA`、`PE`、`TLE`、`MLE`、`OLE`、`RE`、`CE`、`SE`。
- Agent 协议字段要保持 JSON 可序列化，不混入 server/runner 内部实现类型。
- `JudgeLimit` 只含题目基础限制 `timeLimit` 和 `memoryLimit`；wall-clock、pid、output、build 上限由 Agent 内部推导。
- 默认分页大小固定 50，pageSize 上限固定 100。
- storage/testdata helper 必须校验 POSIX 相对路径，拒绝绝对路径、反斜杠、控制字符、`.` 和 `..` 片段。
- settings 公开类型不能包含 `_` 私有字段原值。
