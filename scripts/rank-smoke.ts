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
  const user = await api<{ token: string; user: { id: number; name: string } }>(
    '/api/auth/register',
    {
      method: 'POST',
      body: JSON.stringify({
        name: `rank_${runId.slice(0, 8)}`,
        email: `rank_${runId}@example.test`,
        password: 'password123'
      })
    }
  )
  const { problem, version } = await api<{ problem: { id: number }; version: { id: number } }>(
    '/api/problems',
    {
      method: 'POST',
      headers: authHeaders(admin.token),
      body: JSON.stringify({
        title: `Rank Smoke ${runId.slice(0, 8)}`,
        slug: `rank-smoke-${runId}`,
        statementMarkdown: '# Rank Smoke\n\nExit successfully.'
      })
    }
  )

  const submission = await api<{ id: number }>('/api/submissions', {
    method: 'POST',
    headers: authHeaders(user.token),
    body: JSON.stringify({
      problemId: problem.id,
      problemVersionId: version.id,
      languageId: 'py',
      sourceCode: 'print("rank")\n'
    })
  })

  const judged = await waitForJudgement(submission.id)
  if (judged.status !== 'AC') throw new Error(`expected AC, got ${judged.status}`)

  const rank = await api<{
    list: Array<{ id: number; solvedCount: number; submissionCount: number }>
  }>('/api/rank')
  const row = rank.list.find((item) => item.id === user.user.id)
  if (!row || row.solvedCount < 1 || row.submissionCount < 1) {
    throw new Error(`rank row did not update: ${JSON.stringify(row)}`)
  }

  console.log({
    userId: user.user.id,
    solvedCount: row.solvedCount,
    submissionCount: row.submissionCount
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
