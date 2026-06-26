# Web API Audit

This file records the Web/Admin API boundary review after the 2026-06 audit.

## Boundary Rules

- List endpoints return summary fields only. Detail endpoints own large text, source code, cases, comments, rank tables, and storage-derived data.
- Patch endpoints are real partial updates. A UI action that changes one field should send only that field.
- Default page loads must not fetch unrelated admin, rank, assignment, contest, backup, or option lists.
- Per-row reference data such as user names, problem titles, reply counts, list counts, and user progress must be batched.
- Object storage access is detail-only unless the endpoint is explicitly an asset endpoint.
- Admin utility endpoints should match the local UI need. For example, assignment member selectors use `GET /api/admin/members`, not the full admin overview.
- Mutation responses should match the UI scope that triggered them: list actions return list/detail rows only when the caller needs them; configuration patches return the updated configuration object.

## Fixes From This Pass

- `PATCH /api/me` now accepts partial `mail`, `bio`, or `avatar` updates. Avatar upload no longer resends mail/bio.
- `PATCH /api/problems/{id}` now accepts partial problem updates. Statement edits no longer resend visibility/mode, and mode changes no longer resend title/statement/tags/limits.
- `PATCH /api/discussion/{id}` now accepts partial discussion updates. Pin/lock actions no longer fetch detail and resend title/content/tags.
- `GET /api/assignments` now returns `AssignmentListItem` without `users/groups`, and the backend skips assignment member queries for the list/home path.
- Added `GET /api/admin/members` for user/group selectors so assignment editing does not fetch settings, languages, judgers, or queue data.
- Added tests for partial profile/problem/discussion updates, assignment list member omission, and the admin members endpoint.

## Endpoint Matrix

| Endpoint | Role | Boundary | Review |
| --- | --- | --- | --- |
| `GET /api/health` | Probe | Small health object. | OK. No DB-heavy behavior. |
| `GET /api/ready` | Probe | Small dependency status object. | OK. Intentionally checks readiness. |
| `GET /api/events` | Live | SSE event stream. | OK. Separate from list data. |
| `GET /api/site` | Config read | Public site settings only. | OK. Small and cacheable by client. |
| `GET /api/me` | Session read | Current user or guest DTO. | OK. No profile activity payload. |
| `PATCH /api/me` | Partial mutation | Optional `mail`, `bio`, `avatar`. | Fixed. Only provided fields are validated and saved. |
| `PATCH /api/me/password` | Mutation | Password fields only. | OK. No profile payload coupling. |
| `POST /api/auth/login` | Auth mutation | Login credentials in, `Me` out. | OK. |
| `POST /api/auth/register` | Auth mutation | Registration fields in, `Me` out. | OK. |
| `POST /api/auth/logout` | Auth mutation | Clears session. | OK. |
| `GET /api/languages` | Option list | Submit language id/name/source. | OK for submit form; image/compile/run stay admin-only. |
| `POST /api/uploads/images` | Asset write | User image upload result only. | OK. Storage-owning endpoint. |
| `GET /api/admin` | Admin overview | Users, groups, languages, judgers, queue. | Accepted as tab-scoped management overview; frontend avoids loading it on settings/backup/default pages. |
| `GET /api/admin/settings` | Admin config read | Site settings only. | OK. Used by settings tab. |
| `PATCH /api/admin/settings` | Partial config mutation | Optional setting keys only. | OK. Empty patch rejected; unrelated settings preserved. |
| `GET /api/admin/members` | Admin option list | Users and groups only. | Added. Used by assignment member selectors instead of full overview. |
| `POST /api/admin/users` | Admin mutation | User create request, refreshed admin overview. | Accepted for current users/groups management tab; revisit if tables exceed current 200-row admin bound. |
| `PATCH /api/admin/users/{name}` | Admin mutation | Role/groups only. | OK request boundary. Response is overview for current tab refresh. |
| `DELETE /api/admin/users/{name}` | Admin mutation | Path id only. | OK request boundary. Response is overview for current tab refresh. |
| `POST /api/admin/users/{name}/password` | Admin mutation | Password reset result only. | OK. Does not reload overview. |
| `POST /api/admin/groups` | Admin mutation | Group fields only. | OK request boundary. Response is overview for current tab refresh. |
| `PATCH /api/admin/groups/{id}` | Admin mutation | Group name/users only. | OK request boundary. |
| `DELETE /api/admin/groups/{id}` | Admin mutation | Path id only. | OK request boundary. |
| `POST /api/admin/languages` | Admin mutation | Language definition only. | OK request boundary. Response is overview for current tab refresh. |
| `PATCH /api/admin/languages/{id}` | Admin mutation | Language definition only. | OK request boundary. |
| `DELETE /api/admin/languages/{id}` | Admin mutation | Path id only. | OK request boundary. |
| `POST /api/admin/judgers` | Admin mutation | Judger name only. | OK. Creation returns token in overview row because it is displayed once. |
| `PATCH /api/admin/judgers/{id}` | Admin mutation | Judger name/auth only. | OK request boundary. |
| `DELETE /api/admin/judgers/{id}` | Admin mutation | Path id only. | OK request boundary. |
| `GET /api/admin/backups/settings` | Backup config read | Backup settings only. | OK. Loaded only on backup tab. |
| `PATCH /api/admin/backups/settings` | Backup config mutation | Backup settings form fields. | OK. All fields are visible in that form. |
| `GET /api/admin/backups` | Backup list | Storage-derived backup list. | OK. Backup page owns it. |
| `POST /api/admin/backups` | Backup mutation | Manual backup command. | OK. Running lock is Redis-backed. |
| `GET /api/admin/backups/{name}/download` | Backup asset read | Streams selected backup. | OK. Explicit download endpoint. |
| `DELETE /api/admin/backups/{name}` | Backup mutation | Deletes selected backup. | OK. |
| `GET /api/home` | Home aggregate | Notice, heatmap, recent problem summaries, assignment/contest summary items. | OK. Uses batched counts; no problem statement/storage reads. |
| `PATCH /api/home/notice` | Admin mutation | Notice content only. | OK. Separate from admin settings page. |
| `GET /api/users/{id}/{year}/{month}/{day}/{name}` | Asset read | User uploaded media. | OK. Explicit media endpoint. |
| `GET /api/problems` | List | `ProblemListItem` summary. | OK. No statement or asset storage access; stats/discussions/mine are batched. |
| `POST /api/problems` | Admin mutation | Creation fields only, list item out. | OK. Statement is created by detail edit, not list create. |
| `GET /api/problems/{id}` | Detail | Full `Problem`, statement, storage-derived cases/data size. | OK. Detail endpoint owns storage-derived data. |
| `PATCH /api/problems/{id}` | Partial mutation | Optional title/statement/tags/visible/mode/time/memory. | Fixed. Small UI actions submit only their changed fields. |
| `PATCH /api/problems/{id}/visibility` | List-row mutation | Visibility only, list item out. | OK. Does not touch statement storage. |
| `DELETE /api/problems/{id}` | Admin mutation | Path id only. | OK. |
| `GET /api/problems/{id}/assets` | Asset list | Problem asset inventory. | OK. Loaded only when admin asset modal opens. |
| `POST /api/problems/{id}/assets/images` | Asset write | Problem statement image upload. | OK. |
| `GET /api/problems/{id}/assets/{name}` | Asset read | Public problem asset stream. | OK. |
| `GET /api/problems/{id}/data/{name}` | Asset read | Private data download. | OK. Admin-only. |
| `GET /api/problems/{id}/judge/{name}` | Asset read | Private judge download. | OK. Admin-only. |
| `POST /api/problems/{id}/assets/files` | Asset write | Upload one asset file. | OK. |
| `DELETE /api/problems/{id}/assets/files` | Asset mutation | Delete one asset key. | OK. |
| `GET /api/problems/{id}/assets/files/content` | Asset detail | One editable asset content. | OK. Loaded only when editing selected asset. |
| `PATCH /api/problems/{id}/assets/files/content` | Asset mutation | One editable asset content. | OK. |
| `POST /api/problems/{id}/assets/cases` | Asset mutation | Creates one input/output case pair. | OK. |
| `POST /api/problems/{id}/assets/template` | Asset mutation | Fills judge template. | OK. |
| `GET /api/problems/{id}.zip` | Asset download | Streams problem asset zip. | OK. Explicit download endpoint. |
| `GET /api/assignments` | List | `AssignmentListItem` without members. | Fixed. Visibility/count/done are batched; no member query for list. |
| `POST /api/assignments` | Admin mutation | Full assignment editor fields. | OK. All fields are visible in create modal. |
| `GET /api/assignments/{id}` | Detail | Assignment, linked problem summaries, submissions, progress, admin members. | OK. Detail owns progress/submission data. |
| `PATCH /api/assignments/{id}` | Admin mutation | Full assignment editor fields. | OK for current edit modal because every submitted field is visible/editable together. |
| `DELETE /api/assignments/{id}` | Admin mutation | Path id only. | OK. |
| `GET /api/contests` | List | Contest summary rows with problem count. | OK. Counts are batched. |
| `POST /api/contests` | Admin mutation | Full contest editor fields. | OK. All fields are visible in create modal. |
| `GET /api/contests/{id}` | Detail | Contest, linked problem summaries, rank, context submissions. | OK. Detail owns rank/submission expansion. |
| `PATCH /api/contests/{id}` | Admin mutation | Full contest editor fields. | OK for current edit modal because every submitted field is visible/editable together. |
| `DELETE /api/contests/{id}` | Admin mutation | Path id only. | OK. |
| `GET /api/submissions` | List | Capped submission rows with problem/user names. | OK. Problem/user references are batched; source/cases are detail-only. |
| `POST /api/submissions` | Mutation | Submit code payload, submission row out. | OK. |
| `GET /api/submissions/{id}` | Detail | Submission row, source code, case results. | OK. Detail owns source/cases and visibility checks. |
| `PATCH /api/submissions/{id}` | Partial mutation | Public flag only. | OK. |
| `GET /api/rank` | List | Capped user rank rows. | OK. User stats are batched. |
| `GET /api/users/{name}` | Detail aggregate | Profile, heatmap, solved problem summaries, recent activity. | OK. Solved problems use `ProblemListItem`; activity references are batched. |
| `GET /api/discussion` | List | Discussion summary rows. | OK. Search supported; authors/reply counts are batched; content stays detail-only. |
| `POST /api/discussion` | Mutation | Create title/content/tags. | OK. |
| `GET /api/discussion/{id}` | Detail | Discussion summary, content, comments. | OK. Detail owns content/comments and batches authors. |
| `PATCH /api/discussion/{id}` | Partial mutation | Optional title/content/tags/pinned/locked. | Fixed. Pin/lock submit only the changed flag. |
| `DELETE /api/discussion/{id}` | Admin mutation | Path id only. | OK. |
| `POST /api/discussion/{id}/comments` | Mutation | Comment content only. | OK. |

## Accepted Bounds

- Public list endpoints are currently capped at 50 rows, rank at 100 users, and admin management tables at 200 rows.
- Admin CRUD responses still refresh the management overview for the active management tabs. This is acceptable while admin tables are capped and loaded only in those tabs; if the product grows beyond those bounds, split responses by tab (`AdminMembers`, language list, judger panel) rather than raising limits.
- Assignment/contest edit requests remain full-form mutations because their current modals expose all submitted fields together. If inline single-field controls are added later, add partial endpoints or partial patch semantics before wiring the UI.
- If current caps become product limits rather than practical admin bounds, the next step is cursor pagination or remote search, not raising limits.
