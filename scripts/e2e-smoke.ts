import { eq } from 'drizzle-orm'
import { closeDb, db, schema } from '../packages/db/src/client'

const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const adminUser = process.env.DOJ_ADMIN_NAME ?? 'admin'
const adminPassword = process.env.DOJ_ADMIN_PASSWORD ?? 'admin12345'
const runId = crypto.randomUUID()

try {
  const authResponse = await fetch(`${apiBase}/api/auth/register`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json'
    },
    body: JSON.stringify({
      name: `smoke_${runId.slice(0, 8)}`,
      email: `smoke_${runId}@example.test`,
      password: 'password123'
    })
  })

  if (!authResponse.ok) {
    throw new Error(`auth API failed: ${authResponse.status} ${await authResponse.text()}`)
  }

  const auth = (await authResponse.json()) as { token: string }
  const adminResponse = await fetch(`${apiBase}/api/auth/login`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json'
    },
    body: JSON.stringify({ user: adminUser, password: adminPassword })
  })

  if (!adminResponse.ok) {
    throw new Error(`admin auth API failed: ${adminResponse.status} ${await adminResponse.text()}`)
  }

  const admin = (await adminResponse.json()) as { token: string }

  const problemResponse = await fetch(`${apiBase}/api/problems`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      authorization: `Bearer ${admin.token}`
    },
    body: JSON.stringify({
      title: `Smoke Problem ${runId.slice(0, 8)}`,
      slug: `smoke-${runId}`,
      statementMarkdown: '# Smoke Problem\n\nExit successfully.',
      timeLimitMs: 5000,
      memoryLimitBytes: 128 * 1024 * 1024,
      outputLimitBytes: 1024 * 1024
    })
  })

  if (!problemResponse.ok) {
    throw new Error(`problem API failed: ${problemResponse.status} ${await problemResponse.text()}`)
  }

  const { problem, version } = (await problemResponse.json()) as {
    problem: { id: number }
    version: { id: number }
  }

  const response = await fetch(`${apiBase}/api/submissions`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      authorization: `Bearer ${auth.token}`
    },
    body: JSON.stringify({
      problemId: problem.id,
      problemVersionId: version.id,
      languageId: 'cpp',
      sourceCode:
        '#include <bits/stdc++.h>\nusing namespace std;\nint main(){ cout << "accepted\\n"; return 0; }\n'
    })
  })

  if (!response.ok) {
    throw new Error(`submission API failed: ${response.status} ${await response.text()}`)
  }

  const submission = (await response.json()) as { id: number }
  const judged = await waitForJudgement(submission.id)

  if (judged.status !== 'AC') {
    throw new Error(`expected AC, got ${judged.status}: ${judged.message}`)
  }

  console.log({
    submissionId: judged.id,
    status: judged.status,
    message: judged.message.trim(),
    timeMs: judged.timeMs,
    memoryBytes: judged.memoryBytes
  })
} finally {
  await closeDb()
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
