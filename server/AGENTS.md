# Server API Rules

- `server/api/public` handles guest and signed-in user APIs.
- `server/api/admin` handles system administration APIs.
- `server/api/worker` handles APIs consumed by `doj judger`.
- `server/api/openapi.go` is the Go-first web/admin API contract.
- `server/{auth,cache,storage,validate,settings,judge,backup,events,middleware}` contains server runtime helpers shared by API packages.
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
- An unfinished contest is non-deleted with `end_at > now`, including scheduled contests. Assignments only group problems/submissions; they never gate submission access, results, or source.
- Submission result visibility depends only on its direct `contest_id`: admins see all; owners see all except unfinished OI results; others also lose post-freeze ICPC results. No contest, ended/deleted contest, and pre-freeze ICPC results are visible.
- Submission source visibility is separate: admins and owners always see source; others require `public` and the problem must belong to no unfinished contest. Unfinished contests on a problem never hide unrelated submission results.
- Raw submission counts and heatmaps include hidden results. AC, score, solved, done, rank, status, and result-status filters use only viewer-visible results; hidden/live results contribute `pending`. Explicit assignment/contest statistics scope by context id; global statistics include all contexts.
- Hidden results must not expose score, time, memory, messages, cases, or progress. Hidden source is an empty `code`.
- Submission lists, assignment progress and done counts, profile activities, problem latest/mine, and contest ranks must not expose hidden submission results through alternate DTOs.
- `/api/events` only publishes lightweight invalidation signals. Do not put submission status, score, cases, progress, messages, or source in SSE payloads.
- Assignments are visible only to assigned users/groups and administrators.
- Context submissions must be written with `assignment_id` / `contest_id`; do not infer context from problem membership.
- Problem package updates write the new content-addressed ZIP before atomically swapping the JSONB manifest. Keep the compare-and-swap check; failed or replaced blobs are left for package GC rather than deleted in the request path.

## Judger API Invariants

- Judger never reads PostgreSQL, Redis/Valkey, or object storage directly.
- Server delivers problem assets, source code, language config, and task metadata to judger.
- Task lease lives on `submissions.judger_id`, `submissions.lease_until`, and `submissions.attempt`.
- Result writes must validate task id, submission id, and attempt. Stale attempts must not overwrite newer results.
- Judger online status is runtime state in Redis/Valkey, not columns on `judgers`.
- Direct loopback judgers may skip token auth for single-machine deployment. Requests carrying proxy headers must not use this bypass.
- Submission creation, task lease, and result callbacks should publish lightweight events for `/api/events`.
