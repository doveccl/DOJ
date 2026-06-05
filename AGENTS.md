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
   - Collapse seeded C/C++ into one default language.
   - Remove per-problem max output configuration in favor of global/default behavior.
   - Accept looser testdata file naming such as `1.in`, `input1.txt`, `ans01.txt`.
   - Done: improve Docker sandbox security and record the resource-metrics conclusion.
   - Replace raw test-case JSON UI with upload/inspection-oriented tooling.
4. Frontend and UX.
   - Configure Naive UI locale/date locale.
   - Hide auth-required menu items for anonymous users instead of showing dead-end pages.
   - Rework home, assignments, discussion, rank, submission detail, and admin layout.
   - Done: rename page components without the redundant `Page` suffix.
   - Use CDN/dynamic language loading for editor/highlight support.

## User Preferences

- Problem IDs start at 1000. Other numeric IDs can start at 1.
- Default starter problem should be only `P1000 A+B Problem`.
- Default language should be one C/C++ entry unless there is a strong reason to split.
- README should describe the project and development/deployment only, English first and Chinese second.
- Push after every completed milestone or coherent sub-milestone.
