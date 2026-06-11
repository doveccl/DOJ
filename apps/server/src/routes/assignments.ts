import { Hono } from 'hono'
import { and, eq, inArray, isNull } from 'drizzle-orm'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { authMiddleware, requireAuthUser, requireGroup } from '../auth'
import { apiError, notFound } from '../errors'
import {
  getAdminAssignments,
  getAssignmentDetail,
  getAssignmentReport,
  getUserAssignmentDetail,
  getUserAssignments
} from '../services/assignments'
import { dateString, listQuerySchema, numericId, pageOffset } from '../validation'

const createAssignmentSchema = z.object({
  title: z.string().min(1).max(100),
  description: z.string().max(20_000).default(''),
  endAt: dateString,
  groupIds: z.array(numericId).default([]),
  userIds: z.array(numericId).default([]),
  problemIds: z.array(numericId).min(1)
})
const updateAssignmentSchema = createAssignmentSchema.partial().refine((value) => Object.keys(value).length > 0, {
  message: 'At least one field must be updated'
})

export function registerAssignmentRoutes(app: Hono) {
  app.get('/api/assignments', authMiddleware, async (c) => {
    const user = await requireAuthUser(c)
    const { page, pageSize, status } = assignmentListQuerySchema.parse(c.req.query())
    const result = await getUserAssignments(user.id, pageSize, pageOffset(page, pageSize), status)
    return c.json({ items: result.items, page, pageSize, total: result.total })
  })

  app.get('/api/admin/assignments', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const { page, pageSize, status } = assignmentListQuerySchema.parse(c.req.query())
    const result = await getAdminAssignments(pageSize, pageOffset(page, pageSize), status)
    return c.json({ items: result.items, page, pageSize, total: result.total })
  })

  app.post('/api/admin/assignments', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const body = createAssignmentSchema.parse(await c.req.json())
    const groupIds = [...new Set(body.groupIds)]
    const userIds = [...new Set(body.userIds)]
    const problemIds = [...new Set(body.problemIds)]

    const [groups, users, problems] = await Promise.all([
      groupIds.length
        ? db
            .select({ id: schema.groups.id })
            .from(schema.groups)
            .where(inArray(schema.groups.id, groupIds))
        : Promise.resolve([]),
      userIds.length
        ? db
            .select({ id: schema.users.id })
            .from(schema.users)
            .where(inArray(schema.users.id, userIds))
        : Promise.resolve([]),
      visibleProblemRows(problemIds)
    ])

    if (groups.length !== groupIds.length) {
      return apiError(c, 400, 'GROUP_NOT_FOUND', 'One or more groups do not exist')
    }
    if (users.length !== userIds.length) {
      return apiError(c, 400, 'USER_NOT_FOUND', 'One or more users do not exist')
    }
    if (problems.length !== problemIds.length) {
      return apiError(c, 400, 'PROBLEM_NOT_FOUND', 'One or more problems do not exist')
    }

    const result = await db.transaction(async (tx) => {
      const [assignment] = await tx
        .insert(schema.assignments)
        .values({
          title: body.title,
          description: body.description,
          endAt: new Date(body.endAt)
        })
        .returning()

      if (groupIds.length) {
        await tx.insert(schema.assignmentGroups).values(
          groupIds.map((groupId) => ({
            assignmentId: assignment.id,
            groupId
          }))
        )
      }

      if (userIds.length) {
        await tx.insert(schema.assignmentUsers).values(
          userIds.map((userId) => ({
            assignmentId: assignment.id,
            userId
          }))
        )
      }

      await tx.insert(schema.assignmentProblems).values(
        problemIds.map((problemId, index) => ({
          assignmentId: assignment.id,
          problemId,
          sort: index
        }))
      )

      return assignment
    })

    return c.json(await getAssignmentDetail(result.id), 201)
  })

  app.get('/api/admin/assignments/:id', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const assignment = await getAssignmentDetail(numericId.parse(c.req.param('id')))
    if (!assignment) return notFound(c)
    return c.json(assignment)
  })

  app.patch('/api/admin/assignments/:id', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const body = updateAssignmentSchema.parse(await c.req.json())
    const current = await getAssignmentDetail(id)
    if (!current) return notFound(c)
    const ended = new Date() >= current.assignment.endAt
    if (ended && (body.endAt !== undefined || body.groupIds !== undefined || body.userIds !== undefined || body.problemIds !== undefined)) {
      return apiError(c, 409, 'ASSIGNMENT_ENDED', 'Ended assignments can only update title and description')
    }

    const relationError = await validateAssignmentRelations(body)
    if (relationError) {
      return apiError(c, 400, relationError.code, relationError.message)
    }

    await db.transaction(async (tx) => {
      const patch: Partial<typeof schema.assignments.$inferInsert> = { updatedAt: new Date() }
      if (body.title !== undefined) patch.title = body.title
      if (body.description !== undefined) patch.description = body.description
      if (body.endAt !== undefined) patch.endAt = new Date(body.endAt)
      await tx.update(schema.assignments).set(patch).where(eq(schema.assignments.id, id))

      if (body.groupIds) {
        await tx.delete(schema.assignmentGroups).where(eq(schema.assignmentGroups.assignmentId, id))
        const groupIds = [...new Set(body.groupIds)]
        if (groupIds.length) await tx.insert(schema.assignmentGroups).values(groupIds.map((groupId) => ({ assignmentId: id, groupId })))
      }
      if (body.userIds) {
        await tx.delete(schema.assignmentUsers).where(eq(schema.assignmentUsers.assignmentId, id))
        const userIds = [...new Set(body.userIds)]
        if (userIds.length) await tx.insert(schema.assignmentUsers).values(userIds.map((userId) => ({ assignmentId: id, userId })))
      }
      if (body.problemIds) {
        await tx.delete(schema.assignmentProblems).where(eq(schema.assignmentProblems.assignmentId, id))
        const problemIds = [...new Set(body.problemIds)]
        await tx.insert(schema.assignmentProblems).values(problemIds.map((problemId, sort) => ({ assignmentId: id, problemId, sort })))
      }
    })

    return c.json(await getAssignmentDetail(id))
  })

  app.delete('/api/admin/assignments/:id', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const [updated] = await db
      .update(schema.assignments)
      .set({ deletedAt: new Date(), updatedAt: new Date() })
      .where(eq(schema.assignments.id, id))
      .returning()
    if (!updated) return notFound(c)
    return c.json({ ok: true })
  })

  app.post('/api/admin/assignments/:id/restore', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const [updated] = await db
      .update(schema.assignments)
      .set({ deletedAt: null, updatedAt: new Date() })
      .where(eq(schema.assignments.id, id))
      .returning()
    if (!updated) return notFound(c)
    return c.json(await getAssignmentDetail(id))
  })

  app.get('/api/admin/assignments/:id/report', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const report = await getAssignmentReport(numericId.parse(c.req.param('id')))
    if (!report) return notFound(c)
    return c.json(report)
  })

  app.get('/api/my/assignments', authMiddleware, async (c) => {
    const user = await requireAuthUser(c)
    const { page, pageSize, status } = assignmentListQuerySchema.parse(c.req.query())
    const result = await getUserAssignments(user.id, pageSize, pageOffset(page, pageSize), status)

    return c.json({ items: result.items, page, pageSize, total: result.total })
  })

  app.get('/api/my/assignments/:id', authMiddleware, async (c) => {
    const user = await requireAuthUser(c)
    const assignment = await getUserAssignmentDetail(user.id, numericId.parse(c.req.param('id')))
    if (!assignment) return notFound(c)
    return c.json(assignment)
  })
}

const assignmentListQuerySchema = listQuerySchema.extend({
  status: z.enum(['current', 'past']).optional()
})

function visibleProblemRows(problemIds: number[]) {
  return db
    .select({ id: schema.problems.id })
    .from(schema.problems)
    .where(and(inArray(schema.problems.id, problemIds), eq(schema.problems.visible, true), isNull(schema.problems.deletedAt)))
}

async function validateAssignmentRelations(body: z.infer<typeof updateAssignmentSchema>) {
  if (body.groupIds) {
    const groupIds = [...new Set(body.groupIds)]
    const groups = groupIds.length
      ? await db.select({ id: schema.groups.id }).from(schema.groups).where(inArray(schema.groups.id, groupIds))
      : []
    if (groups.length !== groupIds.length) {
      return { code: 'GROUP_NOT_FOUND', message: 'One or more groups do not exist' }
    }
  }

  if (body.userIds) {
    const userIds = [...new Set(body.userIds)]
    const users = userIds.length
      ? await db.select({ id: schema.users.id }).from(schema.users).where(inArray(schema.users.id, userIds))
      : []
    if (users.length !== userIds.length) {
      return { code: 'USER_NOT_FOUND', message: 'One or more users do not exist' }
    }
  }

  if (body.problemIds) {
    const problemIds = [...new Set(body.problemIds)]
    const problems = await visibleProblemRows(problemIds)
    if (problems.length !== problemIds.length) {
      return { code: 'PROBLEM_NOT_FOUND', message: 'One or more problems do not exist' }
    }
  }

  return null
}
