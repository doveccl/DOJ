import { and, asc, eq, lt, or } from 'drizzle-orm'
import { db, schema, sqlClient } from './client'

export const judgeTaskChannel = 'doj_judge_tasks'

export interface ClaimJudgeTaskOptions {
  workerId: string
  leaseSeconds: number
}

export async function enqueueJudgeTask(submissionId: number) {
  const [task] = await db.insert(schema.judgeTasks).values({ submissionId }).returning()
  await notifyJudgeTask(task.id, submissionId)

  return task
}

export async function notifyJudgeTask(taskId: number, submissionId: number) {
  try {
    await sqlClient.notify(judgeTaskChannel, JSON.stringify({ taskId, submissionId }))
  } catch (error) {
    console.warn(`failed to notify judge task ${taskId}:`, error)
  }
}

export async function claimJudgeTask(options: ClaimJudgeTaskOptions) {
  const now = new Date()
  const leaseUntil = new Date(now.getTime() + options.leaseSeconds * 1000)

  return db.transaction(async (tx) => {
    const [task] = await tx
      .select()
      .from(schema.judgeTasks)
      .where(
        or(
          eq(schema.judgeTasks.status, 'WAITING'),
          and(eq(schema.judgeTasks.status, 'RUNNING'), lt(schema.judgeTasks.lockedUntil, now))
        )
      )
      .orderBy(asc(schema.judgeTasks.createdAt))
      .limit(1)
      .for('update', { skipLocked: true })

    if (!task) return null

    const [claimed] = await tx
      .update(schema.judgeTasks)
      .set({
        status: 'RUNNING',
        lockedUntil: leaseUntil,
        updatedAt: now
      })
      .where(eq(schema.judgeTasks.id, task.id))
      .returning()

    await tx
      .update(schema.submissions)
      .set({ status: 'JUDGING', message: 'Judging', updatedAt: now })
      .where(eq(schema.submissions.id, claimed.submissionId))

    return claimed
  })
}

export async function renewJudgeTaskLease(id: number, leaseSeconds: number) {
  const now = new Date()
  const leaseUntil = new Date(now.getTime() + leaseSeconds * 1000)
  const [task] = await db
    .update(schema.judgeTasks)
    .set({ lockedUntil: leaseUntil, updatedAt: now })
    .where(and(eq(schema.judgeTasks.id, id), eq(schema.judgeTasks.status, 'RUNNING')))
    .returning()
  return task ?? null
}

export async function completeJudgeTask(id: number) {
  await db
    .update(schema.judgeTasks)
    .set({ status: 'DONE', lockedUntil: null, updatedAt: new Date() })
    .where(eq(schema.judgeTasks.id, id))
}

export async function failJudgeTask(id: number, error: unknown) {
  await db
    .update(schema.judgeTasks)
    .set({
      lockedUntil: null,
      status: 'FAILED',
      lastError: error instanceof Error ? error.message : String(error),
      updatedAt: new Date()
    })
    .where(eq(schema.judgeTasks.id, id))
}
