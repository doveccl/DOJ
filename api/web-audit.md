# Web API Audit

This file records the current API boundary rules after the 2026-06 audit.

## Boundary Rules

- List endpoints return summary fields only. Detail endpoints own large text, code, cases, and storage-derived data.
- Default page loads must not fetch unrelated admin, rank, assignment, contest, or backup lists.
- Per-row reference data such as user names, problem titles, reply counts, and list statistics must be batched.
- Object storage access is detail-only unless the endpoint is explicitly an asset endpoint.
- Mutation responses should match the UI scope that triggered them: list actions return summary rows, detail saves return detail rows.

## Endpoint Review

- `GET /api/site`, auth, `GET /api/me`, and language endpoints are small configuration/session reads.
- `GET /api/home` returns notice, user heatmap, recent problem summaries, and assignment/contest summaries. Assignment and contest counts are batched.
- `GET /api/problems` returns `ProblemListItem`; it does not read statement files or asset storage. `GET /api/problems/{id}` returns full `Problem` and owns statement plus asset stats.
- Problem asset endpoints are separate and remain the only problem routes that list, read, write, zip, or delete object storage data.
- Assignment and contest list endpoints use batched visibility/count helpers. Detail endpoints batch linked problem loading and then compute context-specific submissions/progress/rank.
- `GET /api/submissions` is capped and batches problem/user references. `GET /api/submissions/{id}` owns source code and case details.
- `GET /api/rank` batches user statistics instead of counting per user. User profile solved/activity sections batch problem and submission reference data.
- Discussion list supports query/tag search and batches authors plus reply counts. Discussion detail batches comment authors and owns content/comments.
- `GET /api/admin/settings` is the lightweight settings read. `GET /api/admin` remains the management overview for users, groups, languages, judgers, and queue, and the frontend only loads it when those tabs need it.
- Backup settings/list/manual/download/delete routes are separate from the admin overview and only load on the backup tab.

## Accepted Bounds

- Public list endpoints are currently capped at 50 rows, rank at 100 users, and admin management tables at 200 rows.
- If those caps become product limits rather than practical admin bounds, the next step is cursor pagination or remote search, not raising limits.
