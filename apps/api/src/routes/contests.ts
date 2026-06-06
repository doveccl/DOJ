import { Hono } from 'hono'
import { desc, inArray } from 'drizzle-orm'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { authMiddleware, getOptionalAuthUser, requireGroup } from '../auth'
import { getContestDetail, getContestScoreboard } from '../services/contests'
import { countRows } from '../services/stats'
import { dateString, listQuerySchema, numericId, pageOffset } from '../validation'

const createContestSchema = z.object({
  title: z.string().min(1).max(160),
  description: z.string().max(10_000).default(''),
  type: z.enum(['OI', 'ICPC']).default('OI'),
  startAt: dateString,
  endAt: dateString,
  freezeAt: dateString.optional(),
  problems: z
    .array(
      z.object({
        problemId: numericId,
        key: z.string().min(1).max(32),
        score: z.number().int().positive().default(100)
      })
    )
    .min(1)
})

export function registerContestRoutes(app: Hono) {
  app.get('/api/contests', async (c) => {
    const { page, pageSize } = listQuerySchema.parse(c.req.query())
    const total = await countRows(schema.contests)
    const list = await db
      .select()
      .from(schema.contests)
      .orderBy(desc(schema.contests.startAt))
      .limit(pageSize)
      .offset(pageOffset(page, pageSize))
    return c.json({ total, page, pageSize, list })
  })

  app.post('/api/contests', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const body = createContestSchema.parse(await c.req.json())
    const startAt = new Date(body.startAt)
    const endAt = new Date(body.endAt)
    const freezeAt = body.freezeAt ? new Date(body.freezeAt) : null
    if (endAt <= startAt) {
      return c.json({ code: 'INVALID_CONTEST_TIME', message: 'endAt must be after startAt' }, 400)
    }
    if (freezeAt && (freezeAt <= startAt || freezeAt >= endAt)) {
      return c.json(
        { code: 'INVALID_CONTEST_FREEZE', message: 'freezeAt must be between startAt and endAt' },
        400
      )
    }

    const problemIds = [...new Set(body.problems.map((problem) => problem.problemId))]
    const problems = await db
      .select({ id: schema.problems.id })
      .from(schema.problems)
      .where(inArray(schema.problems.id, problemIds))
    if (problems.length !== problemIds.length) {
      return c.json(
        { code: 'PROBLEM_NOT_FOUND', message: 'One or more problems do not exist' },
        400
      )
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
        body.problems.map((problem, index) => ({
          contestId: contest.id,
          problemId: problem.problemId,
          key: problem.key,
          score: problem.score,
          sortOrder: index
        }))
      )

      return contest
    })

    return c.json(await getContestDetail(result.id), 201)
  })

  app.get('/api/contests/:id', async (c) => {
    const authUser = await getOptionalAuthUser(c)
    const isAdmin = authUser?.groups.includes('admin') === true
    const contest = await getContestDetail(numericId.parse(c.req.param('id')))
    if (!contest) return c.notFound()
    if (!isAdmin && Date.now() < contest.contest.startAt.getTime()) {
      return c.json({ ...contest, problems: [] })
    }
    return c.json(contest)
  })

  app.get('/api/contests/:id/scoreboard', async (c) => {
    const scoreboard = await getContestScoreboard(numericId.parse(c.req.param('id')))
    if (!scoreboard) return c.notFound()
    return c.json(scoreboard)
  })

  app.get('/api/contests/:id/scoreboard/reveal', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const scoreboard = await getContestScoreboard(numericId.parse(c.req.param('id')), {
      reveal: true
    })
    if (!scoreboard) return c.notFound()
    return c.json(scoreboard)
  })
}
