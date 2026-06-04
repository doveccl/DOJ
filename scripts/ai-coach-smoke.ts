import { closeDb, db, schema } from '../packages/db/src/client'

const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const runId = crypto.randomUUID()

try {
  const [user] = await db
    .insert(schema.users)
    .values({
      name: `coach_${runId.slice(0, 8)}`,
      email: `coach_${runId}@example.test`,
      passwordHash: 'smoke'
    })
    .returning()

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
      'content-type': 'application/json'
    },
    body: JSON.stringify({
      userId: user.id,
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
