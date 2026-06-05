import { Hono } from 'hono'
import { desc, eq } from 'drizzle-orm'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { authMiddleware, requireAuthUser, requireGroup } from '../auth'
import { numericId } from '../validation'

export function registerAdminUserRoutes(app: Hono) {
  app.get('/api/users', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const list = await db
      .select({
        id: schema.users.id,
        name: schema.users.name,
        email: schema.users.email,
        solvedCount: schema.users.solvedCount,
        submissionCount: schema.users.submissionCount,
        disabledAt: schema.users.disabledAt,
        createdAt: schema.users.createdAt
      })
      .from(schema.users)
      .orderBy(desc(schema.users.createdAt))
      .limit(50)

    return c.json({ total: list.length, list })
  })

  app.patch('/api/users/:id', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const authUser = await requireAuthUser(c)
    const id = numericId.parse(c.req.param('id'))
    const body = updateUserSchema.parse(await c.req.json())
    if (id === authUser.id && body.disabled) {
      return c.json(
        { code: 'CANNOT_DISABLE_SELF', message: 'Cannot disable your own account' },
        400
      )
    }

    const [user] = await db
      .update(schema.users)
      .set({
        disabledAt: body.disabled ? new Date() : null,
        updatedAt: new Date()
      })
      .where(eq(schema.users.id, id))
      .returning({
        id: schema.users.id,
        name: schema.users.name,
        email: schema.users.email,
        solvedCount: schema.users.solvedCount,
        submissionCount: schema.users.submissionCount,
        disabledAt: schema.users.disabledAt,
        createdAt: schema.users.createdAt
      })

    if (!user) return c.notFound()
    return c.json(user)
  })
}

const updateUserSchema = z.object({
  disabled: z.boolean()
})
