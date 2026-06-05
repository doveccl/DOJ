# DOJ

New implementation line for DOJ, built from a blank `main` branch.

## Direction

- Bun for API, worker, and tooling.
- PostgreSQL as the primary database and initial queue backend.
- MinIO/S3 for problem data, images, and attachments.
- Docker runner as the first judging backend.
- Vue 3, Vite, Naive UI, and vue-i18n for the frontend.

The v3 branch remains the migration and product baseline. The v4 branch is treated as a Docker judging experiment.

## Layout

```text
apps/
  api/       Hono API server
  web/       Vue + Naive UI frontend
  worker/    Judge task worker
packages/
  db/        Drizzle schema and PostgreSQL queue helpers
  runner/    Runner interface and Docker implementation
  storage/   S3-compatible object storage helpers
  shared/    Shared domain types
docs/        Architecture notes
```

## Local Development

Start infrastructure:

```sh
docker compose -f compose.dev.yml up -d
```

Install dependencies:

```sh
bun install
cp .env.example .env
```

Generate and apply database migrations:

```sh
bun run db:generate
bun run db:migrate
bun run s3:ensure-bucket
bun run db:seed
```

Run services:

```sh
bun run dev
```

Run checks:

```sh
bun run check
bun run smoke
```

Or run targeted smoke checks:

```sh
bun run auth:smoke
bun run admin:smoke
bun run languages:smoke
bun run runners:smoke
bun run group-membership:smoke
bun run assignment:smoke
bun run my-assignments:smoke
bun run contest:smoke
bun run bbs:smoke
bun run rank:smoke
bun run runner:smoke
bun run e2e:smoke
bun run testcases:smoke
bun run ai:smoke
```

Default URLs:

- API: `http://localhost:7974`
- Web: `http://localhost:28080`
- MinIO console: `http://localhost:9001`

## Current State

The current implementation includes:

- Auth with Argon2id password hashes and JWT sessions.
- Linux-like users/groups/admin membership with admin user suspension.
- Numeric PostgreSQL IDs, with problems starting at 1000.
- Admin-configurable judge languages, with C/C++ as the default seeded languages.
- Admin-configurable Docker runners, including local sockets and HTTP(S) Docker API endpoints. Remote endpoint credentials can be embedded in the URL, for example `https://user:pass@docker.example.com`.
- PostgreSQL-backed judge task queue with leases.
- Docker runner build/run/cleanup with time, output, and memory limits plus best-effort cgroup peak memory reads.
- Inline problem test cases, per-case submission results, and AC/WA/PE/TLE/MLE/OLE/RE/CE/SE aggregation.
- S3-backed problem testdata ZIP upload for `.in/.out` case pairs.
- Assignments for groups.
- Contest basics with timed submissions and AI coaching disabled in contests.
- Non-AC AI coaching with owner/admin authorization, hidden-test redaction, and a provider abstraction: `local-stub` by default, optional OpenAI Responses API via `AI_PROVIDER=openai`.
- Lightweight BBS topics/replies with optional problem/contest links.
- Rank list backed by first-AC solve tracking.
- Bun-native S3 object reads/writes and bucket provisioning for MinIO/S3.
- v3 JSON migration tool in `scripts/migrate-v3-json.ts`.
