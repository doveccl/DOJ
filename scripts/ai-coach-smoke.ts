import { eq } from 'drizzle-orm'
import { closeDb, db, schema } from '../packages/db/src/client'

const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const runId = crypto.randomUUID()

try {
  const authResponse = await fetch(`${apiBase}/api/auth/register`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json'
    },
    body: JSON.stringify({
      name: `coach_${runId.slice(0, 8)}`,
      email: `coach_${runId}@example.test`,
      password: 'password123'
    })
  })

  if (!authResponse.ok) {
    throw new Error(`auth API failed: ${authResponse.status} ${await authResponse.text()}`)
  }

  const auth = (await authResponse.json()) as { token: string }

  const problemResponse = await fetch(`${apiBase}/api/problems`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json'
    },
    body: JSON.stringify({
      title: `Coach Problem ${runId.slice(0, 8)}`,
      slug: `coach-${runId}`,
      statementMarkdown: '# Coach Problem\n\nFail intentionally.',
      timeLimitMs: 5000,
      memoryLimitBytes: 128 * 1024 * 1024,
      outputLimitBytes: 1024 * 1024
    })
  })

  if (!problemResponse.ok) {
    throw new Error(`problem API failed: ${problemResponse.status} ${await problemResponse.text()}`)
  }

  const { problem, version } = (await problemResponse.json()) as {
    problem: { id: string }
    version: { id: string }
  }

  const submissionResponse = await fetch(`${apiBase}/api/submissions`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      authorization: `Bearer ${auth.token}`
    },
    body: JSON.stringify({
      problemId: problem.id,
      problemVersionId: version.id,
      languageId: 'sh',
      sourceCode: '#!/bin/sh\necho boom >&2\nexit 42\n'
    })
  })

  if (!submissionResponse.ok) {
    throw new Error(
      `submission API failed: ${submissionResponse.status} ${await submissionResponse.text()}`
    )
  }

  const submission = (await submissionResponse.json()) as { id: string }
  const judged = await waitForJudgement(submission.id)
  if (judged.status !== 'RE') {
    throw new Error(`expected RE, got ${judged.status}: ${judged.message}`)
  }

  const coachResponse = await fetch(`${apiBase}/api/submissions/${submission.id}/coach`, {
    method: 'POST'
  })

  if (!coachResponse.ok) {
    throw new Error(`coach API failed: ${coachResponse.status} ${await coachResponse.text()}`)
  }

  const session = (await coachResponse.json()) as {
    id: string
    model: string
    responseMarkdown: string
  }

  console.log({
    sessionId: session.id,
    model: session.model,
    preview: session.responseMarkdown.split('\n').slice(0, 3).join(' ')
  })
} finally {
  await closeDb()
}

async function waitForJudgement(submissionId: string) {
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
