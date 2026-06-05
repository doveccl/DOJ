import { eq } from 'drizzle-orm'
import { closeDb, db, schema } from '../packages/db/src/client'

const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const adminUser = process.env.DOJ_ADMIN_NAME ?? 'admin'
const adminPassword = process.env.DOJ_ADMIN_PASSWORD ?? 'admin12345'
const runId = crypto.randomUUID()

try {
  const admin = await api<{ token: string }>('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify({ user: adminUser, password: adminPassword })
  })
  const adminHeaders = authHeaders(admin.token)

  const { problem, version } = await api<{ problem: { id: number }; version: { id: number } }>(
    '/api/problems',
    {
      method: 'POST',
      headers: adminHeaders,
      body: JSON.stringify({
        title: `Contest Smoke Problem ${runId.slice(0, 8)}`,
        slug: `contest-smoke-${runId}`,
        statementMarkdown: '# Contest Smoke\n\nExit successfully.'
      })
    }
  )

  const now = Date.now()
  const detail = await api<{
    contest: { id: number }
    problems: Array<{ id: number; key: string }>
  }>('/api/contests', {
    method: 'POST',
    headers: adminHeaders,
    body: JSON.stringify({
      title: `Contest Smoke ${runId.slice(0, 8)}`,
      description: 'Smoke contest',
      type: 'ICPC',
      startAt: new Date(now - 60_000).toISOString(),
      freezeAt: new Date(now - 30_000).toISOString(),
      endAt: new Date(now + 60 * 60_000).toISOString(),
      problems: [{ problemId: problem.id, key: 'A', score: 100 }]
    })
  })

  const user = await api<{ token: string }>('/api/auth/register', {
    method: 'POST',
    body: JSON.stringify({
      name: `contest_${runId.slice(0, 8)}`,
      email: `contest_${runId}@example.test`,
      password: 'password123'
    })
  })

  const submission = await api<{ id: number; contestId: number }>('/api/submissions', {
    method: 'POST',
    headers: authHeaders(user.token),
    body: JSON.stringify({
      problemId: problem.id,
      problemVersionId: version.id,
      contestId: detail.contest.id,
      languageId: 'cpp',
      sourceCode:
        '#include <bits/stdc++.h>\nusing namespace std;\nint main(){ cout << "contest\\n"; }\n'
    })
  })

  if (submission.contestId !== detail.contest.id) {
    throw new Error(`submission lost contestId: ${JSON.stringify(submission)}`)
  }

  const judged = await waitForJudgement(submission.id)
  if (judged.status !== 'AC') {
    throw new Error(`expected AC, got ${judged.status}: ${judged.message}`)
  }

  const publicDetail = await api<{
    sourceCode: string
    message: string
    cases: unknown[]
    restricted: boolean
  }>(`/api/submissions/${submission.id}`)
  if (
    !publicDetail.restricted ||
    publicDetail.sourceCode ||
    publicDetail.message ||
    publicDetail.cases.length
  ) {
    throw new Error(`contest submission leaked public detail: ${JSON.stringify(publicDetail)}`)
  }

  const ownerDetail = await api<{ sourceCode: string; restricted: boolean }>(
    `/api/submissions/${submission.id}`,
    { headers: { authorization: `Bearer ${user.token}` } }
  )
  if (ownerDetail.restricted || !ownerDetail.sourceCode.includes('contest')) {
    throw new Error(`contest owner detail was not available: ${JSON.stringify(ownerDetail)}`)
  }

  const coachResponse = await fetch(`${apiBase}/api/submissions/${submission.id}/coach`, {
    method: 'POST',
    headers: {
      authorization: `Bearer ${user.token}`
    }
  })
  if (coachResponse.status !== 403) {
    throw new Error(
      `expected contest coaching 403, got ${coachResponse.status}: ${await coachResponse.text()}`
    )
  }

  const scoreboard = await api<{
    frozen: boolean
    revealed: boolean
    rows: Array<{
      userId: number
      solved: number
      problems: Record<string, { attempts: number; frozenAttempts: number }>
    }>
  }>(`/api/contests/${detail.contest.id}/scoreboard`)
  const frozenRow = scoreboard.rows.find((item) => item.userId === judged.userId)
  if (
    !scoreboard.frozen ||
    scoreboard.revealed ||
    !frozenRow ||
    frozenRow.solved !== 0 ||
    frozenRow.problems.A?.frozenAttempts !== 1
  ) {
    throw new Error(`scoreboard did not hide frozen submission: ${JSON.stringify(scoreboard)}`)
  }

  const revealed = await api<{
    revealed: boolean
    rows: Array<{ userId: number; solved: number }>
  }>(`/api/contests/${detail.contest.id}/scoreboard/reveal`, { headers: adminHeaders })
  const revealedRow = revealed.rows.find((item) => item.userId === judged.userId)
  if (!revealed.revealed || !revealedRow || revealedRow.solved !== 1) {
    throw new Error(`scoreboard reveal missing solved row: ${JSON.stringify(revealed)}`)
  }

  console.log({
    contestId: detail.contest.id,
    problemKey: detail.problems[0]?.key,
    submissionId: submission.id,
    status: judged.status,
    coachStatus: coachResponse.status,
    frozenSolved: frozenRow.solved,
    revealedSolved: revealedRow.solved
  })
} finally {
  await closeDb()
}

async function api<T>(path: string, init: RequestInit = {}) {
  const headers = new Headers(init.headers)
  if (!headers.has('content-type') && init.body) headers.set('content-type', 'application/json')
  const response = await fetch(`${apiBase}${path}`, { ...init, headers })
  if (!response.ok) throw new Error(`${path} failed: ${response.status} ${await response.text()}`)
  return (await response.json()) as T
}

function authHeaders(token: string) {
  return {
    'content-type': 'application/json',
    authorization: `Bearer ${token}`
  }
}

async function waitForJudgement(submissionId: number) {
  for (let attempt = 0; attempt < 20; attempt += 1) {
    await runWorkerOnce()

    const [judged] = await db
      .select()
      .from(schema.submissions)
      .where(eq(schema.submissions.id, submissionId))
      .limit(1)

    if (!judged) throw new Error(`submission disappeared: ${submissionId}`)
    if (!['WAITING', 'JUDGING'].includes(judged.status)) return judged
    await Bun.sleep(200)
  }

  throw new Error(`submission did not finish judging: ${submissionId}`)
}

async function runWorkerOnce() {
  const worker = Bun.spawn(['bun', 'run', '--cwd', 'apps/worker', 'dev'], {
    env: {
      ...process.env,
      DOJ_WORKER_ONCE: '1'
    },
    stdout: 'pipe',
    stderr: 'pipe'
  })

  const exitCode = await worker.exited
  if (exitCode !== 0) {
    throw new Error(
      [
        `worker failed with exit ${exitCode}`,
        await new Response(worker.stdout).text(),
        await new Response(worker.stderr).text()
      ].join('\n')
    )
  }
}
