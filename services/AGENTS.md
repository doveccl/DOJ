# Server API Rules

- `services/web` handles guest and signed-in user APIs.
- `services/admin` handles system administration APIs.
- `services/judger` handles APIs consumed by `doj-judger`.
- Handlers may call GORM and small helpers directly until the logic is large enough to justify another layer.

## Product Invariants

- Discussion tags are soft text associations. Hiding or deleting a problem must not hide discussions or block comments.
- Contest problem visibility is derived:
  - before contest start: hidden from the problem set and detail pages;
  - during contest: hidden from the public problem set, visible through contest detail;
  - after contest end: follows the problem's own visibility.
- Contest rank must follow the contest kind:
  - OI: last score per problem;
  - ICPC: accepted count and penalty from the first AC;
  - ICPC freeze: non-admin rank uses submissions before `freeze_at`; admin rank is live.
- Assignments are visible only to assigned users/groups and administrators.
- Context submissions must be written with `assignment_id` / `contest_id`; do not infer context from problem membership.

## Judger API Invariants

- Judger never reads PostgreSQL, Redis/Valkey, or object storage directly.
- Server delivers problem assets, source code, language config, and task metadata to judger.
- Task lease lives on `submissions.judger_id`, `submissions.lease_until`, and `submissions.attempt`.
- Result writes must validate task id, submission id, and attempt. Stale attempts must not overwrite newer results.
- Judger online status is runtime state in Redis/Valkey, not columns on `judgers`.
- Direct loopback judgers may skip token auth for single-machine deployment. Requests carrying proxy headers must not use this bypass.
- Submission creation, task lease, and result callbacks should publish lightweight events for `/api/events`.
