import { Hono } from 'hono'
import { and, eq } from 'drizzle-orm'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { authMiddleware, requireGroup } from '../auth'
import { apiError, notFound } from '../errors'
import { countRows } from '../services/stats'
import { listQuerySchema, numericId, pageOffset } from '../validation'

export function registerAdminGroupRoutes(app: Hono) {
  app.get('/api/admin/groups', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const { page, pageSize } = listQuerySchema.parse(c.req.query())
    const total = await countRows(schema.groups)
    const items = await db
      .select()
      .from(schema.groups)
      .orderBy(schema.groups.name)
      .limit(pageSize)
      .offset(pageOffset(page, pageSize))
    return c.json({ items, page, pageSize, total })
  })

  app.post('/api/admin/groups', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const body = createGroupSchema.parse(await c.req.json())
    const [group] = await db
      .insert(schema.groups)
      .values({
        name: body.name
      })
      .returning()

    return c.json(group, 201)
  })

  app.patch('/api/admin/groups/:id', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const body = createGroupSchema.parse(await c.req.json())
    const [group] = await db
      .update(schema.groups)
      .set({ name: body.name, updatedAt: new Date() })
      .where(eq(schema.groups.id, id))
      .returning()
    if (!group) return notFound(c)
    return c.json(group)
  })

  app.delete('/api/admin/groups/:id', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const [group] = await db.delete(schema.groups).where(eq(schema.groups.id, id)).returning()
    if (!group) return notFound(c)
    return c.json({ ok: true })
  })

  app.get('/api/admin/groups/:id/users', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const groupId = numericId.parse(c.req.param('id'))
    const list = await db
      .select({
        id: schema.users.id,
        name: schema.users.name,
        email: schema.users.email,
        createdAt: schema.userGroups.createdAt
      })
      .from(schema.userGroups)
      .innerJoin(schema.users, eq(schema.userGroups.userId, schema.users.id))
      .where(eq(schema.userGroups.groupId, groupId))
      .orderBy(schema.users.name)

    return c.json({ items: list, page: 1, pageSize: list.length || 50, total: list.length })
  })

  app.post('/api/admin/groups/:id/users', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const groupId = numericId.parse(c.req.param('id'))
    const body = addGroupUserSchema.parse(await c.req.json())
    const [group, user] = await Promise.all([
      db
        .select({ id: schema.groups.id })
        .from(schema.groups)
        .where(eq(schema.groups.id, groupId))
        .limit(1),
      db
        .select({ id: schema.users.id })
        .from(schema.users)
        .where(eq(schema.users.id, body.userId))
        .limit(1)
    ])

    if (!group.length)
      return apiError(c, 404, 'GROUP_NOT_FOUND', 'Group does not exist')
    if (!user.length) return apiError(c, 404, 'USER_NOT_FOUND', 'User does not exist')

    await db
      .insert(schema.userGroups)
      .values({ groupId, userId: body.userId })
      .onConflictDoNothing()

    return c.json({ ok: true }, 201)
  })

  app.delete('/api/admin/groups/:id/users/:userId', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const groupId = numericId.parse(c.req.param('id'))
    const userId = numericId.parse(c.req.param('userId'))
    await db
      .delete(schema.userGroups)
      .where(and(eq(schema.userGroups.groupId, groupId), eq(schema.userGroups.userId, userId)))
    return c.json({ ok: true })
  })
}

const createGroupSchema = z.object({
  name: z.string().min(1).max(100)
})

const addGroupUserSchema = z.object({
  userId: numericId
})
