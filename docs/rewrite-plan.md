# Rewrite Plan

## Baseline

The new `main` branch starts from a blank implementation. v3 is the product and migration baseline. v4 is only a useful reference for Docker-based judging.

## Queue

Do not introduce a separate MQ for the first milestone. Use PostgreSQL as the queue backend because Postgres is already required and deploys cleanly with Docker Compose.

The queue model is:

- `submissions` stores product truth;
- `judge_tasks` stores operational state;
- workers claim tasks with a lease;
- expired leases are recoverable;
- startup recovery remains a safety net.

If the queue later becomes an independent bottleneck, move to BullMQ + Valkey. Until then, keep the deployment small.

## Groups

Use a Linux-like model:

- `users`
- `groups`
- `user_groups`

Preseed `admin`, `user`, and optionally `guest`. Assignments and contests can target groups directly. Avoid separate class/course/member-group concepts until the product proves it needs them.

## Foreign Keys

Use physical foreign keys for stable metadata and mapping tables:

- `user_groups`
- `contest_problems`
- `assignment_groups`
- `assignment_problems`

Use logical references plus indexes for historical high-write records:

- `submissions`
- `submission_cases`
- `judge_tasks`
- AI logs
- audit logs

This preserves historical records and makes migration easier while still letting the database protect core metadata.

## Judge Agents

Keep three boundaries:

- API server: business APIs and queueing.
- Judge worker: task lifecycle, PG notifications, agent WebSocket coordination, and result writes.
- Judge agent: separately deployed process that connects to the worker, downloads testdata from S3-compatible storage, and executes on local Docker.
- Runner: in-process library for container build/run/cleanup, IO piping, limits, and metrics.

Docker is the first backend. Podman should only be added after Docker behavior is stable.

Remote judging conclusion:

- Do not use a remote Docker API as the production deployment model. It moves Docker control traffic and build/run IO onto the network path and can make time-limit behavior harder to reason about.
- Run `apps/worker` centrally and deploy one or more `apps/agent` processes on judge machines. Each agent authenticates by key/token over WebSocket, fetches testdata from S3/MinIO before the timed run, and talks only to its local Docker socket.
- Keep `packages/runner` as a library rather than another deployed service. The agent owns networking, authentication, heartbeats, and task protocol; the runner owns sandbox execution.

## Memory Metrics

Current implementation:

- Docker memory limit;
- OOMKilled or exit code 137 means MLE;
- Docker stats stream records the highest observed memory value;
- local Linux deployments additionally try cgroup v2 `memory.peak` or cgroup v1 `memory.max_usage_in_bytes`.

Conclusion:

- Docker stats is the portable baseline and works for Docker Desktop, remote daemons, and DOJ itself running inside Docker.
- Host cgroup peak files are more exact on local Linux when the agent can see the daemon's container cgroups, but they are best-effort only.
- Keep timeout and memory limit enforcement in Docker. Short peaks that exceed the configured limit are still caught by OOMKilled even if the sampled peak value is lower than the true instantaneous peak.

## AI

First feature: non-AC coaching outside contests.

Rules:

- disabled during contests by default;
- configurable per assignment or group;
- no hidden tests or official answers in student-facing prompts;
- every request logs user, problem, submission, prompt version, model, and response.

Later ideas:

- common wrong-answer clusters for teachers;
- code style and complexity feedback;
- statement translation and polishing;
- checker/interactor review checklist;
- edge-case suggestions;
- plagiarism clustering summaries;
- assignment reports.

## BBS

Do not port old low-usage `posts` directly. If discussion is needed, build a simple BBS with topics, replies, tags, and optional problem/contest links.

## Implemented Milestone

- Bun workspace.
- PostgreSQL schema and migrations.
- Auth, group management, and admin user suspension.
- Admin problem creation, visibility control, versioned updates, and S3 testdata upload.
- Configurable judge languages with C/C++ seeded by default.
- WebSocket judge agents with key/token authentication, online status, token rotation, and local Docker execution.
- Submission creation.
- PostgreSQL-backed judge task queue.
- Docker runner MVP.
- Inline test cases and per-case submission results.
- Assignments with student views and progress reports.
- Contest basics with ICPC scoreboard freeze/reveal.
- Lightweight BBS.
- Rank list and first-AC solve tracking.
- Non-AC AI coaching with owner/admin authorization, hidden-test redaction, local-rules, and optional OpenAI Responses API provider.
- v3 JSON migration tool.
- One-command local development via Bun's native parallel script runner for API, web, worker, and local agent.

## Storage

Object reads and writes use Bun's native `S3Client`; the AWS SDK is intentionally not included. The small SigV4 helper in `@doj/shared/storage` is only for bucket control-plane calls used by `s3:ensure-bucket` (`HEAD`/`PUT` bucket), because the runtime S3 client covers object operations but not bucket creation. In normal judging flow, agents fetch testdata through the Bun client with the configured endpoint, access key, secret key, region, and bucket before timed execution starts.

## Next Risks

- S3-backed `.in/.out` testdata zip import is implemented; checker/interactor package support remains a follow-up.
- Runner memory metrics use Docker stats plus best-effort cgroup `memory.peak`; platform calibration remains a follow-up.
- AI coaching has hidden-test redaction coverage; provider-specific rate limits and operational controls remain follow-ups.
- Checker/interactor support is still a follow-up for non-standard judging.
- Contest scoreboard freeze/reveal and assignment reports are implemented; rolling reveal animation remains a future UX enhancement.
