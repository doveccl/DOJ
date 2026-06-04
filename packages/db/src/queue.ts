import { and, asc, eq, lt, or, sql } from 'drizzle-orm'
import { db, schema } from './client'

export interface ClaimJudgeTaskOptions {
  workerId: string
  leaseSeconds: number
}

export async function enqueueJudgeTask(submissionId: string) {
  const [task] = await db
    .insert(schema.judgeTasks)
    .values({ submissionId })
    .returning()

  return task
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

export async function completeJudgeTask(id: string) {
  await db
    .update(schema.judgeTasks)
    .set({ status: 'DONE', lockedBy: null, lockedUntil: null, updatedAt: new Date() })
    .where(eq(schema.judgeTasks.id, id))
}

export async function failJudgeTask(id: string, error: unknown) {
  await db
    .update(schema.judgeTasks)
    .set({
      status: 'FAILED',
      lockedBy: null,
      lockedUntil: null,
      lastError: error instanceof Error ? error.message : String(error),
      updatedAt: new Date()
    })
    .where(eq(schema.judgeTasks.id, id))
}
