import { Hono } from 'hono'
import { eq } from 'drizzle-orm'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { authMiddleware, requireGroup } from '../auth'
import { numericId } from '../validation'

export function registerAdminGroupRoutes(app: Hono) {
  app.get('/api/groups', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const list = await db.select().from(schema.groups).orderBy(schema.groups.key)
    return c.json({ total: list.length, list })
  })

  app.post('/api/groups', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const body = createGroupSchema.parse(await c.req.json())
    const [group] = await db
      .insert(schema.groups)
      .values({
        key: body.key,
        name: body.name,
        description: body.description
      })
      .returning()

    return c.json(group, 201)
  })

  app.get('/api/groups/:id/users', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const groupId = numericId.parse(c.req.param('id'))
    const list = await db
      .select({
        id: schema.users.id,
        name: schema.users.name,
        email: schema.users.email,
        manager: schema.userGroups.manager,
        createdAt: schema.userGroups.createdAt
      })
      .from(schema.userGroups)
      .innerJoin(schema.users, eq(schema.userGroups.userId, schema.users.id))
      .where(eq(schema.userGroups.groupId, groupId))
      .orderBy(schema.users.name)

    return c.json({ total: list.length, list })
  })

  app.post('/api/groups/:id/users', authMiddleware, async (c) => {
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
      return c.json({ code: 'GROUP_NOT_FOUND', message: 'Group does not exist' }, 404)
    if (!user.length) return c.json({ code: 'USER_NOT_FOUND', message: 'User does not exist' }, 404)

    await db
      .insert(schema.userGroups)
      .values({ groupId, userId: body.userId, manager: body.manager })
      .onConflictDoUpdate({
        target: [schema.userGroups.userId, schema.userGroups.groupId],
        set: {
          manager: body.manager
        }
      })

    return c.json({ ok: true }, 201)
  })
}

const createGroupSchema = z.object({
  key: z.string().regex(/^[a-z][a-z0-9_-]{1,63}$/),
  name: z.string().min(1).max(128),
  description: z.string().max(500).default('')
})

const addGroupUserSchema = z.object({
  userId: numericId,
  manager: z.boolean().default(false)
})
