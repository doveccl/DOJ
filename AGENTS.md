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
   - Done: allow uploading uncompressed (loose) testdata files; server repackages them into a stored ZIP so the agent's existing ZIP parse path is unchanged (`parseLooseTestCases`/`buildStoredZip` in `packages/shared/src/testdata.ts`).
   - Done: improve Docker sandbox security and record the resource-metrics conclusion.
   - Done: replace raw test-case JSON UI with upload/inspection-oriented tooling.
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

- [HIGH] Submission detail/list endpoints IDOR. Detail (`apps/api/src/routes/submissions.ts:181-235`) is now gated: `message`/`cases` require owner/admin (`canInspect`), `sourceCode` requires owner/admin or `open`, hidden-problem submissions 404 for non-owners. List (`:29-71`) now also gates `message` to owner/admin. REMAINING: assignment submissions are still readable by walking ids if the problem is visible (no assignment-membership scope on the public list/detail); decide whether to scope assignment submissions to members.
- [MED] Login rate-limit key includes username (`apps/api/src/routes/auth.ts:58-67`), so credential-stuffing across usernames from one IP bypasses it. Add an IP-wide login limit.
- [MED] Testdata upload buffers the whole file into memory before size check (`apps/api/src/routes/problems.ts:159-171`). Stream or check `Content-Length` first.
- [MED] Rate-limit/session degrade to per-process in-memory `Map` when Redis is briefly down (`session.ts:11-16`, `rate-limit.ts:34-55`, `redis.ts`), bypassable across instances in multi-instance deploys.
- [LOW] 500 responses echo internal `error.message` (`apps/api/src/index.ts:31-38`). Register race returns 500 instead of 409 (`routes/auth.ts:30-50`). Contest problem list is public before `startAt` (`routes/contests.ts:96-100`). Login user-enumeration via timing + 409 on register. Rate-limit `incr`+`expire` non-atomic (`redis.ts:50-55`).

### Performance

- [DONE] Admin problem list N+1: replaced per-problem latest-version+file queries with two batched queries (`getLatestProblemVersions` uses `DISTINCT ON` + a single `files` lookup) in `apps/api/src/routes/problems.ts`.
- [HIGH] Scoreboard/assignment report fetch ALL submissions into memory with no limit/cache, on a public hot path (`services/contests.ts:38-50`, `services/assignments.ts:59-70`). Aggregate in DB or cache.
- [HIGH] Global submission list + dashboard sort by `createdAt` but composite indexes lead with other columns, causing full sort (`routes/submissions.ts:56-62`, `routes/public.ts:35-40`). Add `submissions(created_at desc)` index.
- [PARTIAL] Fake pagination: `problems`/`admin-problems` now do real count + offset pagination (`routes/problems.ts`). STILL TODO: contests/assignments/admin-users/admin-groups/discussion.
- [MED] Dashboard runs 5 independent queries serially (`routes/public.ts:22-73`); parallelize. Submission list count+data are serial (`routes/submissions.ts:37-62`); `Promise.all`.
- [MED] Missing indexes for common sorts: `bbs_topics.updated_at`, `contests.start_at`, `assignments.created_at`, `problems.created_at`, `users.created_at`.
- [MED] `countVisibleSubmissions` does `count(*)` over `submissions ⋈ problems` on hot paths (`services/stats.ts:10-16`).

### Judging core (vs v4 experimental work)

- [HIGH] Time limit is **wall-clock**, not CPU time (`packages/runner/src/docker-runner.ts:81,154-166`); unstable/non-reproducible under agent load. Move to cgroup CPU accounting or cpu-time enforcement.
- [HIGH] No checker / special judge / interactor. `compareOutput` only exact/whitespace compare (`packages/runner/src/judge.ts:112-126`). NOTE: schema already has `checkerFileId`/`interactorFileId` (`packages/db/src/schema.ts:212-213`) — declared but unimplemented (doc/impl gap).
- [HIGH] No subtask / partial score. First non-AC breaks the loop (`judge.ts:95-100`); result has a single status, no score field. OI partial scoring is effectively missing despite `contestTypes=['OI','ICPC']`.
- [MED] No real-time judging progress: agent returns one final `result`, no per-case progress message in the protocol (`packages/shared/src/agent.ts:66-83`). Submission sits at JUDGING with no "testing case N".
- [MED] No compile cache (rebuilds image per submission, `judge.ts:10-20`) and no agent-side testdata cache (re-downloads + re-parses zip every job, `apps/agent/src/index.ts:93-106`).
- [MED] Compile success/failure detected by regex on build logs `/error|failed/i` (`docker-runner.ts:71`) instead of build exit code — false positives/negatives.
- [MED] Agent concurrency selection has TOCTOU: `pickAvailableAgent` reads `activeJobs` before `runJob` increments it (`apps/worker/src/index.ts:27-30` vs `agent-server.ts:147`); can over-dispatch.
- [MED] Job timeout (`agentJobTimeoutMs`=120000) equals claim lease (120s); near-timeout judging can be re-claimed -> duplicate judging. Job timeout does not tell the agent to cancel, leaking the agent-side container (`agent-server.ts:149-152`).
- [LOW] Hand-written zip parser rejects data-descriptor streams and relies on filename heuristics (`packages/shared/src/testdata.ts`). Inline vs zip test-case visibility semantics differ.
- [LOW] Dead/duplicate type `JudgeTaskPayload` (`packages/shared/src/judge.ts:32-40`) — actual flow uses `JudgeAgentPayload`. `cgroup` peak memory unreadable on Docker Desktop/remote (`docker-runner.ts:371-401`), falls back to sampling (may miss peaks). `patchDestroySoon` monkey-patch is fragile (`docker-runner.ts:245-252`). Single-process in-memory agent registry assumes one worker (`agent-server.ts:44`).

### Structure / naming

- Frontend routes are inlined in `apps/web/src/main.ts` (no separate router module). `pages/Admin*.vue` are flat, not under `pages/admin/`. Leftover `/admin/runners -> /admin/agents` redirect.
- (Done this iteration) Renamed Bbs -> Discussion across the whole stack: schema tables `discussion_topics`/`discussion_replies` (migration regenerated as `0000_baseline.sql`), API routes `/api/discussion/*` (`routes/discussion.ts`, `services/discussion.ts`), frontend pages `DiscussionList.vue`/`DiscussionDetail.vue` with `/discussion` routes, i18n `discussion.*`, and scripts (`discussion-smoke.ts`, rate-limit/migrate refs).

### UI findings from live walkthrough (anonymous + admin)

- Dashboard exposes user-count / assignment-count stats to anonymous visitors; these should be role-gated (server-side).
- Stat numbers rendered in default blue, clashing with teal theme (fixed once moved to NStatistic + theme).
- Problem list is bare: no search/tag-filter/pagination, no per-user solved status. Admin sees no in-place manage actions on public pages.
- [Deferred] Registration hardening: add CAPTCHA and/or email verification (needs SMTP or third-party). Recorded for a later iteration; not done.
