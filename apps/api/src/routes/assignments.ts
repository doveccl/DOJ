import { Hono } from 'hono'
import { desc, inArray } from 'drizzle-orm'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { authMiddleware, requireAuthUser, requireGroup } from '../auth'
import {
  getAssignmentDetail,
  getAssignmentReport,
  getUserAssignmentDetail,
  getUserAssignments
} from '../services/assignments'
import { countRows } from '../services/stats'
import { dateString, listQuerySchema, numericId, pageOffset } from '../validation'

const createAssignmentSchema = z.object({
  title: z.string().min(1).max(160),
  description: z.string().max(10_000).default(''),
  startAt: dateString.optional(),
  dueAt: dateString.optional(),
  allowLate: z.boolean().default(false),
  aiCoachingEnabled: z.boolean().default(true),
  groupIds: z.array(numericId).min(1),
  problems: z
    .array(
      z.object({
        problemId: numericId,
        score: z.number().int().positive().default(100)
      })
    )
    .min(1)
})

export function registerAssignmentRoutes(app: Hono) {
  app.get('/api/assignments', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const { page, pageSize } = listQuerySchema.parse(c.req.query())
    const total = await countRows(schema.assignments)
    const list = await db
      .select()
      .from(schema.assignments)
      .orderBy(desc(schema.assignments.createdAt))
      .limit(pageSize)
      .offset(pageOffset(page, pageSize))
    return c.json({ total, page, pageSize, list })
  })

  app.post('/api/assignments', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const body = createAssignmentSchema.parse(await c.req.json())
    const groupIds = [...new Set(body.groupIds)]
    const problemIds = [...new Set(body.problems.map((problem) => problem.problemId))]

    const [groups, problems] = await Promise.all([
      db
        .select({ id: schema.groups.id })
        .from(schema.groups)
        .where(inArray(schema.groups.id, groupIds)),
      db
        .select({ id: schema.problems.id })
        .from(schema.problems)
        .where(inArray(schema.problems.id, problemIds))
    ])

    if (groups.length !== groupIds.length) {
      return c.json({ code: 'GROUP_NOT_FOUND', message: 'One or more groups do not exist' }, 400)
    }
    if (problems.length !== problemIds.length) {
      return c.json(
        { code: 'PROBLEM_NOT_FOUND', message: 'One or more problems do not exist' },
        400
      )
    }

    const result = await db.transaction(async (tx) => {
      const [assignment] = await tx
        .insert(schema.assignments)
        .values({
          title: body.title,
          description: body.description,
          startAt: body.startAt ? new Date(body.startAt) : null,
          dueAt: body.dueAt ? new Date(body.dueAt) : null,
          allowLate: body.allowLate,
          aiCoachingEnabled: body.aiCoachingEnabled
        })
        .returning()

      await tx.insert(schema.assignmentGroups).values(
        groupIds.map((groupId) => ({
          assignmentId: assignment.id,
          groupId
        }))
      )

      await tx.insert(schema.assignmentProblems).values(
        body.problems.map((problem, index) => ({
          assignmentId: assignment.id,
          problemId: problem.problemId,
          score: problem.score,
          sortOrder: index
        }))
      )

      return assignment
    })

    return c.json(await getAssignmentDetail(result.id), 201)
  })

  app.get('/api/assignments/:id', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const assignment = await getAssignmentDetail(numericId.parse(c.req.param('id')))
    if (!assignment) return c.notFound()
    return c.json(assignment)
  })

  app.get('/api/assignments/:id/report', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const report = await getAssignmentReport(numericId.parse(c.req.param('id')))
    if (!report) return c.notFound()
    return c.json(report)
  })

  app.get('/api/my/assignments', authMiddleware, async (c) => {
    const user = await requireAuthUser(c)
    const list = await getUserAssignments(user.id, 50)

    return c.json({ total: list.length, list })
  })

  app.get('/api/my/assignments/:id', authMiddleware, async (c) => {
    const user = await requireAuthUser(c)
    const assignment = await getUserAssignmentDetail(user.id, numericId.parse(c.req.param('id')))
    if (!assignment) return c.notFound()
    return c.json(assignment)
  })
}
