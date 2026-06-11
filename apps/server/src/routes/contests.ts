import { Hono } from 'hono'
import { and, desc, eq, inArray, isNull, sql } from 'drizzle-orm'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { authMiddleware, denyGuestAccess, getOptionalAuthUser, requireGroup } from '../auth'
import { apiError, notFound } from '../errors'
import { getContestDetail, getContestScoreboard } from '../services/contests'
import { countRows } from '../services/stats'
import { dateString, listQuerySchema, numericId, pageOffset } from '../validation'

const createContestSchema = z.object({
  title: z.string().min(1).max(100),
  description: z.string().max(20_000).default(''),
  type: z.enum(['OI', 'ICPC']).default('OI'),
  startAt: dateString,
  endAt: dateString,
  freezeAt: dateString.nullable().optional(),
  problemIds: z.array(numericId).min(1)
})
const updateContestSchema = createContestSchema.partial().refine((value) => Object.keys(value).length > 0, {
  message: 'At least one field must be updated'
})

export function registerContestRoutes(app: Hono) {
  app.get('/api/contests', async (c) => {
    const denied = await denyGuestAccess(c, 'Sign in to view contests')
    if (denied) return denied

    const authUser = await getOptionalAuthUser(c)
    const { page, pageSize, status } = contestListQuerySchema.parse(c.req.query())
    const where = contestStatusWhere(status, authUser?.admin === true)
    const total = await countRows(schema.contests, where)
    const items = await db
      .select()
      .from(schema.contests)
      .where(where)
      .orderBy(desc(schema.contests.startAt))
      .limit(pageSize)
      .offset(pageOffset(page, pageSize))
    return c.json({ items, page, pageSize, total })
  })

  app.post('/api/admin/contests', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const body = createContestSchema.parse(await c.req.json())
    const startAt = new Date(body.startAt)
    const endAt = new Date(body.endAt)
    const freezeAt = body.type === 'ICPC' && body.freezeAt ? new Date(body.freezeAt) : null
    if (endAt <= startAt) {
      return apiError(c, 400, 'INVALID_CONTEST_TIME', 'endAt must be after startAt')
    }
    if (body.problemIds.length > 100) return apiError(c, 400, 'TOO_MANY_PROBLEMS', 'Contests can contain at most 100 problems')
    if (freezeAt && (freezeAt <= startAt || freezeAt >= endAt)) {
      return apiError(c, 400, 'INVALID_CONTEST_FREEZE', 'freezeAt must be between startAt and endAt')
    }

    const problemIds = [...new Set(body.problemIds)]
    const problems = await db
      .select({ id: schema.problems.id })
      .from(schema.problems)
      .where(and(inArray(schema.problems.id, problemIds), eq(schema.problems.visible, true), isNull(schema.problems.deletedAt)))
    if (problems.length !== problemIds.length) {
      return apiError(c, 400, 'PROBLEM_NOT_FOUND', 'One or more problems do not exist')
    }

    const result = await db.transaction(async (tx) => {
      const [contest] = await tx
        .insert(schema.contests)
        .values({
          title: body.title,
          description: body.description,
          type: body.type,
          startAt,
          endAt,
          freezeAt
        })
        .returning()

      await tx.insert(schema.contestProblems).values(
        problemIds.map((problemId, index) => ({
          contestId: contest.id,
          problemId,
          key: toContestKey(index),
          sort: index
        }))
      )

      return contest
    })

    return c.json(await getContestDetail(result.id), 201)
  })

  app.patch('/api/admin/contests/:id', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const current = await getContestDetail(id)
    if (!current) return notFound(c)
    const body = updateContestSchema.parse(await c.req.json())
    const now = new Date()
    const started = now >= current.contest.startAt && now < current.contest.endAt
    if (started && (body.type !== undefined || body.startAt !== undefined || body.endAt !== undefined || body.freezeAt !== undefined || body.problemIds !== undefined)) {
      return apiError(c, 409, 'CONTEST_STARTED', 'Running contests can only update title and description')
    }
    const nextType = body.type ?? current.contest.type
    const nextStartAt = body.startAt ? new Date(body.startAt) : current.contest.startAt
    const nextEndAt = body.endAt ? new Date(body.endAt) : current.contest.endAt
    const nextFreezeAt =
      nextType === 'ICPC'
        ? body.freezeAt === undefined
          ? current.contest.freezeAt
          : body.freezeAt
            ? new Date(body.freezeAt)
            : null
        : null
    if (nextEndAt <= nextStartAt) {
      return apiError(c, 400, 'INVALID_CONTEST_TIME', 'endAt must be after startAt')
    }
    if (nextFreezeAt && (nextFreezeAt <= nextStartAt || nextFreezeAt >= nextEndAt)) {
      return apiError(c, 400, 'INVALID_CONTEST_FREEZE', 'freezeAt must be between startAt and endAt')
    }

    if (body.problemIds) {
      if (body.problemIds.length > 100) return apiError(c, 400, 'TOO_MANY_PROBLEMS', 'Contests can contain at most 100 problems')
      const problemIds = [...new Set(body.problemIds)]
      const problems = await db
        .select({ id: schema.problems.id })
        .from(schema.problems)
        .where(and(inArray(schema.problems.id, problemIds), eq(schema.problems.visible, true), isNull(schema.problems.deletedAt)))
      if (problems.length !== problemIds.length) {
        return apiError(c, 400, 'PROBLEM_NOT_FOUND', 'One or more problems do not exist')
      }
    }

    await db.transaction(async (tx) => {
      const patch: Partial<typeof schema.contests.$inferInsert> = { updatedAt: new Date() }
      if (body.title !== undefined) patch.title = body.title
      if (body.description !== undefined) patch.description = body.description
      if (body.type !== undefined) patch.type = body.type
      if (body.startAt !== undefined) patch.startAt = new Date(body.startAt)
      if (body.endAt !== undefined) patch.endAt = new Date(body.endAt)
      if (body.type === 'OI') patch.freezeAt = null
      else if (body.freezeAt !== undefined) patch.freezeAt = body.freezeAt ? new Date(body.freezeAt) : null
      await tx.update(schema.contests).set(patch).where(eq(schema.contests.id, id))
      if (body.problemIds) {
        await tx.delete(schema.contestProblems).where(eq(schema.contestProblems.contestId, id))
        const problemIds = [...new Set(body.problemIds)]
        await tx.insert(schema.contestProblems).values(problemIds.map((problemId, index) => ({
          contestId: id,
          problemId,
          key: toContestKey(index),
          sort: index
        })))
      }
    })

    return c.json(await getContestDetail(id))
  })

  app.delete('/api/admin/contests/:id', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const [updated] = await db
      .update(schema.contests)
      .set({ deletedAt: new Date(), updatedAt: new Date() })
      .where(eq(schema.contests.id, id))
      .returning()
    if (!updated) return notFound(c)
    return c.json({ ok: true })
  })

  app.post('/api/admin/contests/:id/restore', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const [updated] = await db
      .update(schema.contests)
      .set({ deletedAt: null, updatedAt: new Date() })
      .where(eq(schema.contests.id, id))
      .returning()
    if (!updated) return notFound(c)
    return c.json(await getContestDetail(id))
  })

  app.get('/api/contests/:id', async (c) => {
    const denied = await denyGuestAccess(c, 'Sign in to view contests')
    if (denied) return denied

    const authUser = await getOptionalAuthUser(c)
    const isAdmin = authUser?.admin === true
    const contest = await getContestDetail(numericId.parse(c.req.param('id')), { publicView: !isAdmin })
    if (!contest) return notFound(c)
    if (!isAdmin && contest.contest.deletedAt) return notFound(c)
    if (!isAdmin && Date.now() < contest.contest.startAt.getTime()) {
      return c.json({ ...contest, problems: [] })
    }
    return c.json(contest)
  })

  app.get('/api/contests/:id/scoreboard', async (c) => {
    const denied = await denyGuestAccess(c, 'Sign in to view contests')
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const detail = await getContestDetail(id, { includeProblems: false })
    if (!detail || detail.contest.deletedAt) return notFound(c)
    const { page, pageSize } = listQuerySchema.parse(c.req.query())
    const scoreboard = await getContestScoreboard(id, { page, pageSize })
    if (!scoreboard) return notFound(c)
    if (Date.now() < new Date(scoreboard.contest.startAt).getTime()) {
      return apiError(c, 403, 'CONTEST_NOT_STARTED', 'Contest has not started')
    }
    return c.json(scoreboard)
  })

  app.get('/api/admin/contests/:id/scoreboard/full', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const { page, pageSize } = listQuerySchema.parse(c.req.query())
    const scoreboard = await getContestScoreboard(numericId.parse(c.req.param('id')), {
      reveal: true,
      page,
      pageSize
    })
    if (!scoreboard) return notFound(c)
    return c.json(scoreboard)
  })

}

const contestListQuerySchema = listQuerySchema.extend({
  status: z.enum(['current', 'upcoming', 'past']).optional()
})

function contestStatusWhere(status: 'current' | 'upcoming' | 'past' | undefined, includeDeleted: boolean) {
  const notDeleted = includeDeleted ? undefined : isNull(schema.contests.deletedAt)
  const statusWhere =
    status === 'current'
      ? and(sql`${schema.contests.startAt} <= now()`, sql`${schema.contests.endAt} > now()`)
      : status === 'upcoming'
        ? sql`${schema.contests.startAt} > now()`
        : status === 'past'
          ? sql`${schema.contests.endAt} <= now()`
          : undefined
  return and(notDeleted, statusWhere)
}

function toContestKey(index: number) {
  let n = index
  let key = ''
  do {
    key = String.fromCharCode(65 + (n % 26)) + key
    n = Math.floor(n / 26) - 1
  } while (n >= 0)
  return key
}
