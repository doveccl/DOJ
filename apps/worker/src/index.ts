import { eq } from 'drizzle-orm'
import { db, schema } from '@doj/db/client'
import { claimJudgeTask, completeJudgeTask, failJudgeTask } from '@doj/db/queue'
import { DockerRunner } from '@doj/runner/docker-runner'
import { getLanguage } from './languages'

const workerId = `worker-${crypto.randomUUID()}`
const concurrency = Number(process.env.DOJ_JUDGE_CONCURRENCY ?? 2)
const runner = new DockerRunner()

async function handleOne() {
  const task = await claimJudgeTask({ workerId, leaseSeconds: 120 })
  if (!task) return false
  let submissionId = task.submissionId

  try {
    const [submission] = await db
      .select()
      .from(schema.submissions)
      .where(eq(schema.submissions.id, task.submissionId))
      .limit(1)

    if (!submission) throw new Error(`submission not found: ${task.submissionId}`)
    submissionId = submission.id

    const [version] = await db
      .select()
      .from(schema.problemVersions)
      .where(eq(schema.problemVersions.id, submission.problemVersionId))
      .limit(1)

    if (!version) throw new Error(`problem version not found: ${submission.problemVersionId}`)

    const language = getLanguage(submission.languageId)
    const scopeId = `submission-${submission.id}`

    await db
      .update(schema.submissions)
      .set({ status: 'JUDGING', updatedAt: new Date() })
      .where(eq(schema.submissions.id, submission.id))

    const build = await runner.build({
      scopeId,
      dockerfile: language.dockerfile(language.sourceFile),
      files: {
        [language.sourceFile]: submission.sourceCode
      },
      limits: {
        timeMs: version.timeLimitMs,
        memoryBytes: version.memoryLimitBytes
      }
    })

    if (!build.ok || !build.imageId) {
      await db
        .update(schema.submissions)
        .set({
          status: 'CE',
          message: build.logs,
          updatedAt: new Date()
        })
        .where(eq(schema.submissions.id, submission.id))
      await completeJudgeTask(task.id)
      await runner.cleanup({ scopeId })
      return true
    }

    const result = await runner.run({
      scopeId,
      imageId: build.imageId,
      command: language.command.length ? language.command : undefined,
      limits: {
        timeMs: version.timeLimitMs,
        memoryBytes: version.memoryLimitBytes,
        outputBytes: version.outputLimitBytes
      }
    })

    await db
      .update(schema.submissions)
      .set({
        status: result.status,
        timeMs: result.timeMs,
        memoryBytes: result.memoryBytes,
        message: result.stderr || Buffer.from(result.stdout).toString('utf8'),
        updatedAt: new Date()
      })
      .where(eq(schema.submissions.id, submission.id))

    await completeJudgeTask(task.id)
    await runner.cleanup({ scopeId })
  } catch (error) {
    await db
      .update(schema.submissions)
      .set({
        status: 'SE',
        message: error instanceof Error ? error.message : String(error),
        updatedAt: new Date()
      })
      .where(eq(schema.submissions.id, submissionId))
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

if (process.env.DOJ_WORKER_ONCE === '1') {
  await handleOne()
  process.exit(0)
}

await Promise.all(Array.from({ length: concurrency }, (_, i) => loop(i)))
