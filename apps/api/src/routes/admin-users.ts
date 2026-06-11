import { Hono } from 'hono'
import { and, desc, eq, ilike, isNull, or, sql } from 'drizzle-orm'
import { createHash, randomBytes } from 'node:crypto'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { authMiddleware, hashPassword, requireGroup } from '../auth'
import { apiError, notFound } from '../errors'
import { countRows } from '../services/stats'
import { listQuerySchema, numericId, pageOffset } from '../validation'

export function registerAdminUserRoutes(app: Hono) {
  app.get('/api/users/:id', async (c) => {
    const id = numericId.parse(c.req.param('id'))
    const [user] = await db
      .select({
        id: schema.users.id,
        name: schema.users.name,
        introduction: schema.users.introduction,
        email: schema.users.email,
        createdAt: schema.users.createdAt
      })
      .from(schema.users)
      .where(and(eq(schema.users.id, id), isNull(schema.users.disabledAt)))
      .limit(1)
    if (!user) return notFound(c)
    const [stats] = await db
      .select({
        solved: sql<number>`count(distinct ${schema.submissions.problemId}) filter (where ${schema.submissions.status} = 'AC')::int`,
        submissions: sql<number>`count(${schema.submissions.id})::int`
      })
      .from(schema.submissions)
      .where(eq(schema.submissions.userId, id))
    return c.json({
      id: user.id,
      name: user.name,
      introduction: user.introduction,
      avatarUrl: gravatarUrl(user.email),
      solved: stats?.solved ?? 0,
      submissions: stats?.submissions ?? 0,
      createdAt: user.createdAt.toISOString()
    })
  })

  app.get('/api/admin/users', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const { page, pageSize, q } = adminUserListQuerySchema.parse(c.req.query())
    const keyword = q.trim()
    const where = keyword
      ? or(ilike(schema.users.name, `%${keyword}%`), ilike(schema.users.email, `%${keyword}%`))
      : sql`true`
    const total = await countRows(schema.users, where)
    const items = await db
      .select({
        id: schema.users.id,
        name: schema.users.name,
        email: schema.users.email,
        introduction: schema.users.introduction,
        admin: schema.users.admin,
        disabled: sql<boolean>`${schema.users.disabledAt} is not null`,
        mustChangePassword: schema.users.mustChangePassword,
        createdAt: schema.users.createdAt,
        updatedAt: schema.users.updatedAt
      })
      .from(schema.users)
      .where(where)
      .orderBy(desc(schema.users.createdAt))
      .limit(pageSize)
      .offset(pageOffset(page, pageSize))

    return c.json({ items, page, pageSize, total })
  })

  app.post('/api/admin/users', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const body = createUserSchema.parse(await c.req.json())
    const [created] = await db
      .insert(schema.users)
      .values({
        name: body.name,
        email: body.email.trim().toLowerCase(),
        passwordHash: await hashPassword(body.password),
        admin: body.admin ?? false,
        disabledAt: body.disabled ? new Date() : null
      })
      .returning()
      .catch((error: unknown) => {
        if (isUniqueViolation(error)) return []
        throw error
      })
    if (!created) return apiError(c, 409, 'EMAIL_EXISTS', 'Email already exists')
    return c.json(formatAdminUser(created), 201)
  })

  app.get('/api/admin/users/:id', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const [user] = await db.select().from(schema.users).where(eq(schema.users.id, numericId.parse(c.req.param('id')))).limit(1)
    if (!user) return notFound(c)
    return c.json(formatAdminUser(user))
  })

  app.patch('/api/admin/users/:id', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const body = updateUserSchema.parse(await c.req.json())
    if ((body.admin === false || body.disabled === true) && await wouldRemoveLastAdmin(id)) {
      return apiError(c, 409, 'LAST_ADMIN', 'Cannot remove the last active admin')
    }

    const patch: Partial<typeof schema.users.$inferInsert> = { updatedAt: new Date() }
    if (body.name !== undefined) patch.name = body.name
    if (body.email !== undefined) patch.email = body.email.trim().toLowerCase()
    if (body.introduction !== undefined) patch.introduction = body.introduction
    if (body.admin !== undefined) patch.admin = body.admin
    if (body.disabled !== undefined) patch.disabledAt = body.disabled ? new Date() : null
    if (body.mustChangePassword !== undefined) patch.mustChangePassword = body.mustChangePassword

    const [user] = await db
      .update(schema.users)
      .set(patch)
      .where(eq(schema.users.id, id))
      .returning()
      .catch((error: unknown) => {
        if (isUniqueViolation(error)) return []
        throw error
      })

    if (!user) return notFound(c)
    return c.json(formatAdminUser(user))
  })

  app.post('/api/admin/users/:id/reset-password', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const body = resetPasswordSchema.parse(await c.req.json().catch(() => ({})))
    const password = body.password ?? randomBytes(12).toString('base64url')
    const [user] = await db
      .update(schema.users)
      .set({ passwordHash: await hashPassword(password), mustChangePassword: true, updatedAt: new Date() })
      .where(eq(schema.users.id, id))
      .returning({ id: schema.users.id })
    if (!user) return notFound(c)
    return c.json({ password })
  })
}

const createUserSchema = z.object({
  name: z.string().regex(/^[a-zA-Z0-9][a-zA-Z0-9_]{2,31}$/),
  email: z.email().transform((email) => email.trim().toLowerCase()),
  password: z.string().min(8).max(128),
  admin: z.boolean().optional(),
  disabled: z.boolean().optional()
})

const adminUserListQuerySchema = listQuerySchema.extend({
  q: z.string().default('')
})

const updateUserSchema = z.object({
  name: z.string().regex(/^[a-zA-Z0-9][a-zA-Z0-9_]{2,31}$/).optional(),
  email: z.email().transform((email) => email.trim().toLowerCase()).optional(),
  introduction: z.string().max(500).optional(),
  admin: z.boolean().optional(),
  disabled: z.boolean().optional(),
  mustChangePassword: z.boolean().optional()
}).refine((value) => Object.keys(value).length > 0, { message: 'At least one field must be updated' })

const resetPasswordSchema = z.object({
  password: z.string().min(8).max(128).optional()
})

function formatAdminUser(user: typeof schema.users.$inferSelect) {
  return {
    id: user.id,
    name: user.name,
    email: user.email,
    introduction: user.introduction,
    admin: user.admin,
    disabled: user.disabledAt !== null,
    mustChangePassword: user.mustChangePassword,
    createdAt: user.createdAt.toISOString(),
    updatedAt: user.updatedAt.toISOString()
  }
}

async function wouldRemoveLastAdmin(id: number) {
  const [target] = await db.select().from(schema.users).where(eq(schema.users.id, id)).limit(1)
  if (!target?.admin || target.disabledAt) return false
  const [row] = await db
    .select({ total: sql<number>`count(*)::int` })
    .from(schema.users)
    .where(and(eq(schema.users.admin, true), isNull(schema.users.disabledAt)))
  return (row?.total ?? 0) <= 1
}

function gravatarUrl(email: string) {
  const hash = createHash('md5').update(email.trim().toLowerCase()).digest('hex')
  return `https://www.gravatar.com/avatar/${hash}?d=identicon&s=80`
}

function isUniqueViolation(error: unknown) {
  return (
    typeof error === 'object' &&
    error !== null &&
    'code' in error &&
    (error as { code?: unknown }).code === '23505'
  )
}
