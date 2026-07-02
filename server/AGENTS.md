# Server API Rules

- `server/web` handles guest and signed-in user APIs.
- `server/admin` handles system administration APIs.
- `server/judger` handles APIs consumed by `doj judger`.
- Handlers may call GORM and small helpers directly until the logic is large enough to justify another layer.

## Product Invariants

- Discussion tags are soft text associations. Hiding or deleting a problem must not hide discussions or block comments.
- Contest problem visibility is derived:
  - before contest start: hidden from the problem set and detail pages;
  - during contest: hidden from the public problem set, visible through contest detail and direct problem detail links;
  - after contest end: follows the problem's own visibility.
- Contest rank must follow the contest kind:
  - OI: last score per problem;
  - ICPC: accepted count and penalty from the first AC;
  - ICPC freeze: non-admin rank uses submissions before `freeze_at`; admin rank is live.
- Submission result visibility is independent from source visibility:
  - submission list/detail access is independent from problem visibility; hidden problem titles may appear on submissions, while problem detail links still enforce problem visibility;
  - normal submissions and ended assignments/contests expose results to everyone;
  - running assignments expose results to everyone but source only to the owner/admin;
  - running OI contests hide results from non-admin users, including the owner;
  - running ICPC contests expose results unless a non-owner/non-admin views a post-freeze submission;
  - hidden results must be represented as `pending` and must not expose score, time, memory, messages, cases, or progress.
- Submission source is visible to the owner/admin; other users only see it for normal or ended-context submissions when `public` is true.
- Submission lists, assignment progress and done counts, profile activities, problem latest/mine, and contest ranks must not expose hidden submission results through alternate DTOs.
- `/api/events` only publishes lightweight invalidation signals. Do not put submission status, score, cases, progress, messages, or source in SSE payloads.
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
