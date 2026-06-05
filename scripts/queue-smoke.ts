import { eq } from 'drizzle-orm'
import { closeDb, db, schema } from '../packages/db/src/client'
import { claimJudgeTask, enqueueJudgeTask, failJudgeTask } from '../packages/db/src/queue'

const workerId = `queue-smoke-${crypto.randomUUID()}`

try {
  const task = await enqueueJudgeTask(1)
  const first = await claimJudgeTask({ workerId, leaseSeconds: 60 })
  if (!first || first.id !== task.id || first.attempts !== 1 || first.status !== 'RUNNING') {
    throw new Error(`first claim mismatch: ${JSON.stringify(first)}`)
  }

  await failJudgeTask(first.id, new Error('transient failure'))
  const second = await claimJudgeTask({ workerId, leaseSeconds: 60 })
  if (!second || second.id !== task.id || second.attempts !== 2 || second.status !== 'RUNNING') {
    throw new Error(`retry claim mismatch: ${JSON.stringify(second)}`)
  }

  await failJudgeTask(second.id, new Error('still failing'))
  const third = await claimJudgeTask({ workerId, leaseSeconds: 60 })
  if (!third || third.id !== task.id || third.attempts !== 3 || third.status !== 'RUNNING') {
    throw new Error(`final claim mismatch: ${JSON.stringify(third)}`)
  }

  await failJudgeTask(third.id, new Error('final failure'))
  const [exhausted] = await db
    .select()
    .from(schema.judgeTasks)
    .where(eq(schema.judgeTasks.id, task.id))
    .limit(1)
  if (exhausted?.status !== 'FAILED')
    throw new Error(`task was not exhausted: ${JSON.stringify(exhausted)}`)

  console.log({
    taskId: task.id,
    attempts: [first.attempts, second.attempts, third.attempts],
    exhausted: true
  })
} finally {
  await closeDb()
}
