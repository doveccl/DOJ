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

## Judge Runner

Keep three boundaries:

- API server: business APIs and queueing.
- Judge worker: task lifecycle and result writes.
- Runner: container build/run/cleanup, IO piping, limits, metrics.

Docker is the first backend. Podman should only be added after Docker behavior is stable.

Remote runner modes:

- The current remote runner support talks directly to a Docker API endpoint. The worker streams build context, source files, stdin, stdout, stderr, and Docker control traffic over that connection. This is convenient for one or a few trusted remote Docker hosts, especially behind HTTPS with URL credentials, but it is not the same deployment model as v3's remote worker.
- For serious remote judging, deploy `apps/worker` on the judge host instead. The worker claims tasks from PostgreSQL, downloads S3 testdata locally, talks to the local Docker socket, and writes final results back. Set `DOJ_RUNNER_KEYS=judge-a` on that worker so it only uses the runner row for that host. This keeps large testdata/stdin/stdout traffic off the Docker API WAN path and matches the old v3 remote-worker deployment shape more closely.

## Memory Metrics

MVP:

- Docker memory limit;
- OOMKilled or exit code 137 means MLE;
- poll Docker stats every 100ms;
- record highest observed memory value.

This is simple and acceptable for the first implementation. It can miss short spikes, but memory-limit enforcement still catches fatal spikes.

Enhancement:

- add calibration tests for short memory spikes;
- read cgroup v2 `memory.peak` when available, otherwise fall back to Docker stats;
- document deployment modes because DOJ itself may run inside Docker and may not see the host cgroup hierarchy.

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
- Configurable Docker runners with local socket or HTTP(S) endpoint; URL credentials are supported for reverse-proxied Docker APIs.
- Worker-side `DOJ_RUNNER_KEYS` filtering for remote judge hosts that run their own worker against local Docker.
- Submission creation.
- PostgreSQL-backed judge task queue.
- Docker runner MVP.
- Inline test cases and per-case submission results.
- Assignments with student views and progress reports.
- Contest basics with ICPC scoreboard freeze/reveal.
- Lightweight BBS.
- Rank list and first-AC solve tracking.
- Non-AC AI coaching with owner/admin authorization, hidden-test redaction, local-stub, and optional OpenAI Responses API provider.
- v3 JSON migration tool.
- One-command local development via Bun's native parallel script runner.

## Next Risks

- S3-backed `.in/.out` testdata zip import is implemented; checker/interactor package support remains a follow-up.
- Direct remote Docker API remains a convenience mode; production remote judge hosts should run `apps/worker` locally with `DOJ_RUNNER_KEYS` to avoid network IO affecting time limits.
- Runner memory metrics use Docker stats plus best-effort cgroup `memory.peak`; platform calibration remains a follow-up.
- AI coaching has hidden-test redaction coverage; provider-specific rate limits and operational controls remain follow-ups.
- Checker/interactor support is still a follow-up for non-standard judging.
- Contest scoreboard freeze/reveal and assignment reports are implemented; rolling reveal animation remains a future UX enhancement.
