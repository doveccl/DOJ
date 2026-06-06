# DOJ Agent Notes

This file is the working memory for autonomous coding on this branch. Keep it current when broad user requirements change. Do not use Codex goals for this project; track durable work here instead.

## Current Direction

- The current `main` branch is an early rewrite branch. Database history can be reset freely until a formal release.
- Prefer Bun for backend runtime and tooling, Vue 3 + Naive UI for the frontend, PostgreSQL for durable state, Redis for session/rate-limit/cache concerns, and S3-compatible object storage for large artifacts.
- Judging now uses a central worker plus separately deployed judge agents over WebSocket. Production remote judging should avoid remote Docker APIs; agents fetch testdata from S3-compatible storage and execute on local Docker.
- The UI should be compact and task-focused. Avoid page-header boilerplate, redundant explanatory subtitles, and admin-looking dashboard copy on student-facing pages.

## Milestones

1. Repository hygiene and developer experience.
   - Done: rewrite README as bilingual project/dev/deploy docs.
   - Done: keep agent-only notes in `AGENTS.md`.
   - Done: remove noisy root smoke scripts; use a single smoke runner with named targets.
   - Done: remove subpackage `version` fields.
   - Done: add ESLint alongside Prettier.
   - Done: reset Drizzle migration history to a fresh baseline while the schema is still fluid.
   - Done: remove `packages/storage`; keep storage runtime helpers close to shared backend code.
   - Done: simplify dev compose/env defaults.
2. Backend architecture and infrastructure.
   - Done: split API routes/services out of the monolithic `apps/api/src/index.ts`.
   - Done: add DB-backed runtime settings for non-secret behavior flags.
   - Done: add Redis/Valkey for sessions and API rate limits through Bun native Redis.
   - Done: use PostgreSQL `LISTEN/NOTIFY` plus `FOR UPDATE SKIP LOCKED` for judge task wakeups.
   - Done: design and implement the judge-agent connection model.
3. Judge and problem model.
   - Done: collapse seeded C/C++ into one default language (`cc` / `main.cc`).
   - Done: remove per-problem max output configuration in favor of global/default behavior.
   - Done: accept looser testdata file naming such as `1.in`, `input1.txt`, `ans01.txt` (`classifyCaseEntry` in `packages/shared/src/testdata.ts`; covered by `testdata` smoke).
   - Done: remove `problem_versions`; statement/limits/caseCount/inline `testCases` live directly on `problems`, and submissions no longer store `problemVersionId`.
   - Done: problem packages are loose S3-backed files via `problem_files(problemId, path, fileId)` (e.g. `Dockerfile`, `data/1.in`, `assets/*`), surfaced in `AdminProblems.vue` with text editing and upload.
   - Done: improve Docker sandbox security and record the resource-metrics conclusion.
   - Done: replace raw test-case JSON UI with package/inline-case editing tooling.
4. Frontend and UX.
   - Done: configure Naive UI locale/date locale.
   - Done: hide auth-required menu items for anonymous users instead of showing dead-end pages (`menuOptions` in `apps/web/src/App.vue`: problems gated by `guestProblemsetVisible`, assignments require sign-in, admin requires the admin group).
   - Done: remove redundant discussion sign-in copy and use componentized topic tags.
   - Done: rework home (role-gated stat board, only admins see aggregate stats), discussion (Markdown editor + searchable problem/contest link picker), submission detail/list (show language name not id, drop judge message from list), and admin layout (no per-page title boilerplate).
   - Done: rename page components without the redundant `Page` suffix.
   - Done: use CDN/dynamic language loading for editor/highlight support.
   - Done: add a profile page (`/profile`) with editable introduction/password (`PATCH /api/auth/self`); restore the v3 user introduction field.
   - Done: merge admin Users + Groups into a single Members area with sub-tabs (`AdminMembers.vue`).
   - Done: move AI config to a dedicated AI settings tab (provider/baseUrl/model/api key, key never returned to client — only `aiApiKeySet`).
   - Done: shared Markdown editor (`MarkdownEditor.vue`) used by discussion topic/reply and problem statement editing, with image upload via `POST /api/media`.

## User Preferences

- Problem IDs start at 1000. Other numeric IDs can start at 1.
- Default starter problem should be only `P1000 A+B Problem`.
- Default language should be one C/C++ entry unless there is a strong reason to split.
- README should describe the project and development/deployment only, English first and Chinese second.
- Push after every completed milestone or coherent sub-milestone.
- Same page, role-based progressive disclosure: the **server** decides what data a viewer receives (return `canManage`/role and trim sensitive fields server-side); the frontend only decides how to render. Never rely on frontend-only hiding for sensitive data.
- Frontend uses SCSS. Design tokens (CSS variables) live in global styles so runtime theme switching works; component-private styles live in scoped SFC blocks. Do NOT override Naive UI defaults (incl. corner radius) without a concrete reason — prefer Naive's own look. Prefer composing Naive components (NLayout, NStatistic, NGrid, NDataTable, NThing, NDescriptions, NTabs, NResult) over hand-written markup.
- Markdown editor: build on the existing CodeEditor (Monaco, CDN-loaded) with a split write/preview that reuses MarkdownView (markdown-it + katex + prism) so editing and display stay 100% aligned. Images/attachments upload through our own S3 endpoint (`POST /api/media`, served from `GET /api/files/:id`), not a third-party editor's built-in renderer. Component is `apps/web/src/components/MarkdownEditor.vue`.
- Don't write per-page title/subtitle boilerplate when the selected nav menu already conveys location (applies to admin pages too).
- AI configuration (api key / endpoint / per-feature toggles like AI coaching) should be DB-backed runtime settings, editable in a dedicated AI settings tab.
- Single C/C++ language id is `cc` with source file `main.cc`.

## Review Backlog (2026-06 full-codebase audit)

These were found during a whole-system review. The current iteration focuses on **UI refactor + componentization** and **Bbs -> Discussion rename**. The items below are NOT yet done and are recorded so any agent can continue. Severity in brackets.

### Security (do first when picking up backend work)

- [DONE] Submission detail/list endpoints IDOR. Detail (`apps/api/src/routes/submissions.ts`) gates `message`/`cases` to owner/admin for contests, `sourceCode` to owner/admin or open, hidden-problem submissions 404 for non-owners, and assignment submissions 404 unless owner/admin. Public submission list hides judge messages from non-owner/non-admin and excludes assignment submissions unless owner/admin. Covered by `submission-security` smoke.
- [DONE] Added an IP-wide login rate limit in addition to the existing IP+username bucket (`routes/auth.ts`), so credential-stuffing across usernames from one IP is throttled. Covered by `rate-limit` smoke.
- [DONE] Package upload now checks `Content-Length` before parsing multipart form data and still validates summed uploaded file sizes after parse (`routes/problems.ts`). Covered by `testdata` smoke.
- [MED] Rate-limit/session degrade to per-process in-memory `Map` when Redis is briefly down (`session.ts:11-16`, `rate-limit.ts:34-55`, `redis.ts`), bypassable across instances in multi-instance deploys.
- [PARTIAL] Low-risk auth/API hardening: 500 responses now return a generic message, register unique races map to 409, missing-user login attempts run a dummy Argon2 verify to reduce timing enumeration, contest detail hides problem list before `startAt` for non-admin viewers, and rate-limit `incr`+`expire` is atomic via Redis Lua. REMAINING: register still intentionally returns 409 for duplicate name/email.

### Performance

- [DONE] Admin problem list N+1 eliminated by the versionless problem model; admin list reads `problems` directly and package files are loaded only for detail/package editor requests (`routes/problems.ts`, `AdminProblems.vue`).
- [PARTIAL] Scoreboard/assignment report fetch ALL submissions into memory. Public contest scoreboard now has a 5s Redis cache (`getContestScoreboard` in `services/contests.ts`, key `scoreboard:<id>`, admin reveal bypasses it) to collapse live-contest refresh bursts. Assignment report left as a direct admin-only computation (low concurrency). STILL TODO if scale demands: DB-side aggregation instead of loading all rows.
- [DONE] Added btree sort index `submissions(created_at)` (plus problems/contests/assignments/users/discussion_topics) so dashboard/list `ORDER BY created_at` no longer full-sorts.
- [DONE] Real pagination (count + offset) now on problems/admin-problems/contests/assignments/admin-users/discussion via shared `listQuerySchema`+`pageOffset` (`apps/api/src/validation.ts`). Frontend tables use Naive remote pagination. STILL TODO: admin-groups list (small, capped) and the admin contest/assignment management tables only read page 1 in the UI.
- [DONE] Dashboard recent* queries and submission list count+data now run in `Promise.all` (`routes/public.ts`, `routes/submissions.ts`); stat counts were already parallel.
- [DONE] Missing sort indexes added: `contests.start_at`, `assignments.created_at`, `problems.created_at`, `users.created_at`, `discussion_topics.updated_at`.
- [DONE] `countVisibleSubmissions` no longer counts over `submissions ⋈ problems`; dashboard stats sum `problems.submissionCount` for visible problems (`services/stats.ts`).

### Judging core (vs v4 experimental work)

- [DONE] Time limit now uses **CPU time**, not wall-clock. Each run container is pinned to 1 CPU (`NanoCpus`) and CPU nanoseconds are read from the Docker stats stream (`cpu_stats.cpu_usage.total_usage`) reachable via the Colima Linux daemon (no host cgroup file access needed); TLE trips on CPU time, with a generous wall-clock cap as a safety net for sleeping stalls (`packages/runner/src/docker-runner.ts`). Reported submission time is CPU time. Runner smoke covers busy-loop TLE + sleep-as-AC.
- [DONE] A/B package judging implemented. The worker sends B (submission package) plus loose problem package files to the agent. If the problem package contains `Dockerfile`, the runner builds A (problem package: interactor/checker) and B, then connects them per case with named FIFOs (`a2b`/`b2a`) in a shared volume. A writes `/tmp/result.json` (`status`/`score`/`message`) as the primary verdict, with testlib-style exit codes as fallback. If there is no problem `Dockerfile`, default judging derives cases from `data/` files or inline `testCases` and compares stdout. Covered by `runner`, `e2e`, `testcases`, and `testdata` smokes.
- [DONE] Subtask / partial score implemented. Package/default judging runs all cases (no first-error break) and awards each a weight (explicit per-case `points`, else even split of 100). Added `score` to `submissions`/`submission_cases` (baseline migration regenerated), `score`/`maxScore` in the agent protocol (`packages/shared/src/agent.ts`), persisted in worker, exposed on submission list/detail API, shown in submission UI. Covered by runner smoke (AC/RE/AC -> 67/100). NOTE: still single overall `status` (first failure's status) rather than a true subtask grouping model.
- [MED] No real-time judging progress: agent returns one final `result`, no per-case progress message in the protocol (`packages/shared/src/agent.ts:66-83`). Submission sits at JUDGING with no "testing case N".
- [MED] No compile cache (rebuilds image per submission) and no agent-side package/testdata cache (re-downloads S3 objects every job, `apps/agent/src/index.ts`).
- [DONE] Package build success/failure uses Docker build stream errors, not regex (`buildPackage` in `packages/runner/src/docker-runner.ts`). Legacy `build()` still exists for older paths and retains the old log heuristic.
- [DONE] Agent concurrency selection TOCTOU fixed: worker reserves an agent slot before claiming a task and releases it if no task is claimed, so concurrent worker slots cannot over-dispatch the same agent (`reserveAvailableAgent`/`releaseAgent` in `apps/worker/src/agent-server.ts`).
- [PARTIAL] Job timeout no longer equals claim lease: task lease is now `agentJobTimeoutMs + 60s`, preventing near-timeout jobs from being immediately re-claimed and duplicated. REMAINING: timed-out jobs still do not send an explicit cancel to the agent, so agent-side work may continue until its local process exits.
- [DONE] Removed unused hand-written ZIP parser/ZIP repack helpers from `packages/shared/src/testdata.ts`; default-mode package cases now derive from already-fetched loose `data/` files.
- [PARTIAL] Removed dead duplicate `JudgeTaskPayload`; actual flow uses `JudgeAgentPayload`. REMAINING: `cgroup` peak memory unreadable on Docker Desktop/remote (`docker-runner.ts`), falls back to sampling (may miss peaks). `patchDestroySoon` monkey-patch is fragile. Single-process in-memory agent registry assumes one worker.

### Structure / naming

- Frontend routes are inlined in `apps/web/src/main.ts` (no separate router module). `pages/Admin*.vue` are flat, not under `pages/admin/`. Leftover `/admin/runners -> /admin/agents` redirect.
- (Done this iteration) Renamed Bbs -> Discussion across the whole stack: schema tables `discussion_topics`/`discussion_replies` (migration regenerated as `0000_baseline.sql`), API routes `/api/discussion/*` (`routes/discussion.ts`, `services/discussion.ts`), frontend pages `DiscussionList.vue`/`DiscussionDetail.vue` with `/discussion` routes, i18n `discussion.*`, and scripts (`discussion-smoke.ts`, rate-limit/migrate refs).

### UI findings from live walkthrough (anonymous + admin)

- Dashboard exposes user-count / assignment-count stats to anonymous visitors; these should be role-gated (server-side).
- Stat numbers rendered in default blue, clashing with teal theme (fixed once moved to NStatistic + theme).
- Problem list is bare: no search/tag-filter/pagination, no per-user solved status. Admin sees no in-place manage actions on public pages.
- [Deferred] Registration hardening: add CAPTCHA and/or email verification (needs SMTP or third-party). Recorded for a later iteration; not done.
