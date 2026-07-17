# DOJ AI Rules

## Direction

- Frontend: React + Vite + TypeScript + antd.
- Server: Go + Echo + GORM + OpenAPI.
- Judging: Go `doj judger` on the host and Go `doj runner` in language containers.
- Data services: PostgreSQL, Redis/Valkey, and local or S3-compatible object storage.

## Layout

- `main.go` is the `doj` entrypoint for `server`, `judger`, and `runner` subcommands.
- `contract/` contains cross-boundary DTOs and generated contract artifacts.
- `models/` contains GORM models and database helpers.
- `server/` contains the server entrypoint, server-side API handlers, middleware, and server runtime helpers.
- `server/api/public` handles guest and signed-in user APIs.
- `server/api/admin` handles system administration APIs.
- `server/api/worker` handles APIs consumed by `doj judger`.
- `judger/` contains host-side judger worker and container orchestration.
- `judger/runner` contains container-side runner implementation.
- `index.html` is the Vite HTML entry.
- `web/` contains frontend source.

## Project Rules

- Keep the implementation aligned with the latest agreed schema and API. During active development, do not add backward-compatibility branches for old local data.
- `server/api/openapi.go` is the Go-first web/admin API contract. Run `pnpm api:gen` to regenerate the ignored `contract/web/openapi.yaml` artifact and `web/client/schema.ts`, and update handlers, UI, and tests together.
- `contract/judger` contains the shared server ↔ judger JSON contract types.
- Prefer short table, field, and file names when the meaning stays clear.
- Problem IDs start at 1000.
- Problem memory limits are MB. Submission results, case results, and judger resource usage are KB.
- Problem tags are stored on the problem row, not in a separate tag table.
- Discussions are intentionally soft-linked to problems only through tags such as `P1000`. Do not add a problem foreign key or derive discussion authorization/visibility from those tags; contest spoiler control is moderation and entry-point UX.
- Administrators are trusted operators. For edits to started/used assignments and contests, use a confirmation warning rather than server-side lifecycle locks; retain only trust-boundary validation and data-integrity checks.
- A fresh database intentionally seeds `admin/admin`. The API derives the default-password warning from the password hash, and the UI keeps it visible until the password changes.
- Assignment and contest descriptions live on their database rows and are edited with the rest of the object through one edit action. Their detail header and optional description share one Card; omit the Card body when the description is empty.
- Do not store S3/object keys in the database when the key can be derived from a business id.
- Problem `data/` and `judge/` files share one content-addressed ZIP at `problems/{id}/packages/{hash}.zip`. `problems.package` JSONB is its manifest: ZIP hash/size, file offsets, and cases. Public statement assets stay under `problems/{id}/assets/`.
- A missing case score means 10; case scores and their total are not capped at 100. Do not eagerly delete replaced package ZIPs because an active lease may still use the old hash; package GC removes unreferenced blobs after its grace period.
- Deleted rows are not visible to administrators in the product UI; DBA recovery is outside the app.

## Quality Gate

- For permissions, visibility, assignment membership, contest rules, submission ownership, and statistics, write down the invariant in tests before or with the implementation.
- Assignment visibility must cover guest, unassigned user, assigned user, and admin behavior.
- Submission-related statistics must use submission context fields such as `assignment_id` and `contest_id`, not reverse inference from problem membership.
- Submission result visibility and source visibility are separate. Hidden results are returned as `pending` with score, time, memory, message, cases, and progress cleared; hidden source is returned as an empty `code`.
- Submission list/detail access is independent from problem visibility; hidden problem titles may appear on submissions, while problem detail links still enforce problem visibility.
- Submission lists, assignment progress and done counts, profile activities, problem latest/mine, ranks, and live refreshes must not bypass submission result visibility. SSE events must stay lightweight invalidation signals and must not carry result, case, progress, or source payloads.
- Detail edit modals must mount after data is ready, or use a stable `key`; do not let empty async data seed `initialValues`.
- Prefer fixing dirty development data by resetting it instead of adding compatibility logic.
