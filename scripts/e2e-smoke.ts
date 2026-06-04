import { eq } from 'drizzle-orm'
import { closeDb, db, schema } from '../packages/db/src/client'

const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const runId = crypto.randomUUID()

try {
  const [user] = await db
    .insert(schema.users)
    .values({
      name: `smoke_${runId.slice(0, 8)}`,
      email: `smoke_${runId}@example.test`,
      passwordHash: 'smoke'
    })
    .returning()

  const [problem] = await db
    .insert(schema.problems)
    .values({
      title: `Smoke Problem ${runId.slice(0, 8)}`,
      slug: `smoke-${runId}`
    })
    .returning()

  const [version] = await db
    .insert(schema.problemVersions)
    .values({
      problemId: problem.id,
      version: 1,
      statementMarkdown: '# Smoke Problem\n\nExit successfully.',
      timeLimitMs: 5000,
      memoryLimitBytes: 128 * 1024 * 1024,
      outputLimitBytes: 1024 * 1024
    })
    .returning()

  const response = await fetch(`${apiBase}/api/submissions`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json'
    },
    body: JSON.stringify({
      userId: user.id,
      problemId: problem.id,
      problemVersionId: version.id,
      languageId: 'sh',
      sourceCode: '#!/bin/sh\necho accepted\nsleep 0.3\n'
    })
  })

  if (!response.ok) {
    throw new Error(`submission API failed: ${response.status} ${await response.text()}`)
  }

  const submission = (await response.json()) as { id: string }
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

  const [judged] = await db
    .select()
    .from(schema.submissions)
    .where(eq(schema.submissions.id, submission.id))
    .limit(1)

  if (!judged) throw new Error(`submission disappeared: ${submission.id}`)
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
