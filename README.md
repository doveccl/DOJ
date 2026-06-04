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
```

Generate and apply database migrations:

```sh
bun run db:generate
bun run db:migrate
```

Run services:

```sh
bun run dev:api
bun run dev:worker
bun run dev:web
```

Default URLs:

- API: `http://localhost:7974`
- Web: `http://localhost:28080`
- MinIO console: `http://localhost:9001`

## Current State

This is an initial scaffold. API routing, schema, queue leasing, worker loop, and runner boundaries exist. The Docker runner still needs the real build/run implementation.
