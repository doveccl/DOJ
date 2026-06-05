import { and, asc, eq, lt, or, sql } from 'drizzle-orm'
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
        and(
          or(
            eq(schema.judgeTasks.status, 'WAITING'),
            and(eq(schema.judgeTasks.status, 'RUNNING'), lt(schema.judgeTasks.lockedUntil, now))
          ),
          lt(schema.judgeTasks.attempts, schema.judgeTasks.maxAttempts)
        )
      )
      .orderBy(asc(schema.judgeTasks.priority), asc(schema.judgeTasks.createdAt))
      .limit(1)
      .for('update', { skipLocked: true })

    if (!task) return null

    const [claimed] = await tx
      .update(schema.judgeTasks)
      .set({
        status: 'RUNNING',
        lockedBy: options.workerId,
        lockedUntil: leaseUntil,
        attempts: sql`${schema.judgeTasks.attempts} + 1`,
        updatedAt: now
      })
      .where(eq(schema.judgeTasks.id, task.id))
      .returning()

    return claimed
  })
}

export async function completeJudgeTask(id: number) {
  await db
    .update(schema.judgeTasks)
    .set({ status: 'DONE', lockedBy: null, lockedUntil: null, updatedAt: new Date() })
    .where(eq(schema.judgeTasks.id, id))
}

export async function failJudgeTask(id: number, error: unknown) {
  const [task] = await db
    .select({ attempts: schema.judgeTasks.attempts, maxAttempts: schema.judgeTasks.maxAttempts })
    .from(schema.judgeTasks)
    .where(eq(schema.judgeTasks.id, id))
    .limit(1)
  const retry = task ? task.attempts < task.maxAttempts : false

  await db
    .update(schema.judgeTasks)
    .set({
      status: retry ? 'WAITING' : 'FAILED',
      lockedBy: null,
      lockedUntil: null,
      lastError: error instanceof Error ? error.message : String(error),
      updatedAt: new Date()
    })
    .where(eq(schema.judgeTasks.id, id))
}
