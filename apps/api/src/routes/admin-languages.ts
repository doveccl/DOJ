import { Hono } from 'hono'
import { asc, eq, sql } from 'drizzle-orm'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { authMiddleware, requireGroup } from '../auth'
import { apiError, notFound } from '../errors'

export function registerAdminLanguageRoutes(app: Hono) {
  app.get('/api/admin/languages', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const list = await db
      .select()
      .from(schema.languages)
      .orderBy(asc(schema.languages.sort), asc(schema.languages.id))

    return c.json(list)
  })

  app.post('/api/admin/languages', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const body = languageConfigSchema.parse(await c.req.json())
    const [language] = await db
      .insert(schema.languages)
      .values(body)
      .returning()
      .catch((error: unknown) => {
        if (isUniqueViolation(error)) return []
        throw error
      })
    if (!language) return apiError(c, 409, 'LANGUAGE_EXISTS', 'Language id already exists')

    return c.json(language, 201)
  })

  app.patch('/api/admin/languages/:id', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const body = updateLanguageConfigSchema.parse(await c.req.json())
    const [language] = await db
      .update(schema.languages)
      .set({
        ...body,
        updatedAt: new Date()
      })
      .where(eq(schema.languages.id, c.req.param('id')))
      .returning()

    if (!language) return notFound(c)
    return c.json(language)
  })

  app.delete('/api/admin/languages/:id', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const id = c.req.param('id')
    const [count] = await db.select({ total: sql<number>`count(*)::int` }).from(schema.languages)
    if ((count?.total ?? 0) <= 1) return apiError(c, 409, 'LAST_LANGUAGE', 'Cannot delete the last language')
    const [used] = await db
      .select({ id: schema.submissions.id })
      .from(schema.submissions)
      .where(eq(schema.submissions.languageId, id))
      .limit(1)
    if (used) return apiError(c, 409, 'LANGUAGE_IN_USE', 'Language has historical submissions')
    const [deleted] = await db.delete(schema.languages).where(eq(schema.languages.id, id)).returning()
    if (!deleted) return notFound(c)
    return c.json({ ok: true })
  })
}

const languageConfigSchema = z.object({
  id: z.string().regex(/^[a-z][a-z0-9-]{0,31}$/),
  name: z.string().min(1).max(100),
  source: z.string().regex(/^(?!.*\.\.)[A-Za-z0-9_.-]{1,64}$/),
  dockerfile: z.string().min(1).max(20_000),
  sort: z.number().int().default(0)
})

const updateLanguageConfigSchema = languageConfigSchema.omit({ id: true }).partial()

function isUniqueViolation(error: unknown) {
  return (
    typeof error === 'object' &&
    error !== null &&
    'code' in error &&
    (error as { code?: unknown }).code === '23505'
  )
}
