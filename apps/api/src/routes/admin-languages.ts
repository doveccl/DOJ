import { Hono } from 'hono'
import { asc, eq } from 'drizzle-orm'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { authMiddleware, requireGroup } from '../auth'

export function registerAdminLanguageRoutes(app: Hono) {
  app.get('/api/admin/languages', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const list = await db
      .select()
      .from(schema.judgeLanguages)
      .orderBy(asc(schema.judgeLanguages.sortOrder), asc(schema.judgeLanguages.id))

    return c.json({ total: list.length, list })
  })

  app.post('/api/admin/languages', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const body = languageConfigSchema.parse(await c.req.json())
    const [language] = await db
      .insert(schema.judgeLanguages)
      .values(body)
      .onConflictDoUpdate({
        target: schema.judgeLanguages.id,
        set: {
          name: body.name,
          enabled: body.enabled,
          sourceFile: body.sourceFile,
          dockerfile: body.dockerfile,
          command: body.command,
          sortOrder: body.sortOrder,
          updatedAt: new Date()
        }
      })
      .returning()

    return c.json(language, 201)
  })

  app.patch('/api/admin/languages/:id', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const body = updateLanguageConfigSchema.parse(await c.req.json())
    const [language] = await db
      .update(schema.judgeLanguages)
      .set({
        ...body,
        updatedAt: new Date()
      })
      .where(eq(schema.judgeLanguages.id, c.req.param('id')))
      .returning()

    if (!language) return c.notFound()
    return c.json(language)
  })
}

const languageConfigSchema = z.object({
  id: z.string().regex(/^[a-z][a-z0-9_-]{1,63}$/),
  name: z.string().min(1).max(128),
  enabled: z.boolean().default(true),
  sourceFile: z.string().min(1).max(128),
  dockerfile: z.string().min(1).max(20_000),
  command: z.array(z.string().min(1).max(200)).default([]),
  sortOrder: z.number().int().min(0).default(100)
})

const updateLanguageConfigSchema = languageConfigSchema.omit({ id: true }).partial()
