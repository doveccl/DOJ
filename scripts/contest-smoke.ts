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
      languageId: 'py',
      sourceCode: 'print("contest")\n'
    })
  })

  if (submission.contestId !== detail.contest.id) {
    throw new Error(`submission lost contestId: ${JSON.stringify(submission)}`)
  }

  const judged = await waitForJudgement(submission.id)
  if (judged.status !== 'AC') {
    throw new Error(`expected AC, got ${judged.status}: ${judged.message}`)
  }

  const coachResponse = await fetch(`${apiBase}/api/submissions/${submission.id}/coach`, {
    method: 'POST'
  })
  if (coachResponse.status !== 403) {
    throw new Error(
      `expected contest coaching 403, got ${coachResponse.status}: ${await coachResponse.text()}`
    )
  }

  console.log({
    contestId: detail.contest.id,
    problemKey: detail.problems[0]?.key,
    submissionId: submission.id,
    status: judged.status,
    coachStatus: coachResponse.status
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
