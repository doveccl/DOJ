# DOJ AI Rules

## Direction

- Frontend: React + Vite + TypeScript + antd.
- Server: Go + Echo + GORM + OpenAPI.
- Judging: Go `doj judger` on the host and Go `doj runner` in language containers.
- Data services: PostgreSQL, Redis/Valkey, and local or S3-compatible object storage.

## Layout

- `main.go` is the `doj` entrypoint for `server`, `judger`, and `runner` subcommands.
- `common/` contains shared contracts, generated contract artifacts, and shared helper packages.
- `models/` contains GORM models and database helpers.
- `middleware/` contains Echo middleware.
- `server/` contains the server entrypoint and server-side API handlers.
- `judger/` contains all judger and runner implementation.
- `index.html` is the Vite HTML entry.
- `web/` contains frontend source.

## Project Rules

- Keep the implementation aligned with the latest agreed schema and API. During active development, do not add backward-compatibility branches for old local data.
- `server/web/openapi.go` is the Go-first web/admin API contract. Run `pnpm api:gen` to regenerate the ignored `common/web/openapi.yaml` artifact and `web/client/schema.ts`, and update handlers, UI, and tests together.
- `common/judger` contains the shared server ↔ judger JSON contract types.
- Prefer short table, field, and file names when the meaning stays clear.
- Problem IDs start at 1000.
- Problem memory limits are MB. Submission results, case results, and judger resource usage are KB.
- Problem tags are stored on the problem row, not in a separate tag table.
- Do not store S3/object keys in the database when the key can be derived from a business id.
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
