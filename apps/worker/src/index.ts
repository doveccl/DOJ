import { eq } from 'drizzle-orm'
import { db, schema } from '@doj/db/client'
import { claimJudgeTask, completeJudgeTask, failJudgeTask } from '@doj/db/queue'
import { DockerRunner } from '@doj/runner/docker-runner'

const workerId = `worker-${crypto.randomUUID()}`
const concurrency = Number(process.env.DOJ_JUDGE_CONCURRENCY ?? 2)
const runner = new DockerRunner()

async function handleOne() {
  const task = await claimJudgeTask({ workerId, leaseSeconds: 120 })
  if (!task) return false

  try {
    const [submission] = await db
      .select()
      .from(schema.submissions)
      .where(eq(schema.submissions.id, task.submissionId))
      .limit(1)

    if (!submission) throw new Error(`submission not found: ${task.submissionId}`)

    await db
      .update(schema.submissions)
      .set({ status: 'JUDGING', updatedAt: new Date() })
      .where(eq(schema.submissions.id, submission.id))

    await runner.run({
      scopeId: submission.id,
      imageId: 'not-built-yet',
      limits: {
        timeMs: 1000,
        memoryBytes: 256 * 1024 * 1024,
        outputBytes: 64 * 1024 * 1024
      }
    })

    await db
      .update(schema.submissions)
      .set({
        status: 'SE',
        message: 'Docker runner is scaffolded but not implemented yet.',
        updatedAt: new Date()
      })
      .where(eq(schema.submissions.id, submission.id))

    await completeJudgeTask(task.id)
    await runner.cleanup({ scopeId: submission.id })
  } catch (error) {
    await failJudgeTask(task.id, error)
  }

  return true
}

async function loop(slot: number) {
  console.log(`judge worker ${workerId} slot ${slot} started`)
  for (;;) {
    const claimed = await handleOne()
    if (!claimed) await Bun.sleep(1000)
  }
}

await Promise.all(Array.from({ length: concurrency }, (_, i) => loop(i)))
