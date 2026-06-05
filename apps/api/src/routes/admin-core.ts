import { Hono } from 'hono'
import { asc, desc, eq } from 'drizzle-orm'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { DockerRunner } from '@doj/runner/docker-runner'
import { authMiddleware, requireAuthUser, requireGroup } from '../auth'
import { numericId } from '../validation'

export function registerAdminCoreRoutes(app: Hono) {
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

  app.get('/api/admin/runners', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const list = await db
      .select()
      .from(schema.judgeRunners)
      .orderBy(asc(schema.judgeRunners.sortOrder), asc(schema.judgeRunners.id))

    return c.json({ total: list.length, list })
  })

  app.post('/api/admin/runners', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const body = runnerConfigSchema.parse(await c.req.json())
    const [runner] = await db
      .insert(schema.judgeRunners)
      .values({
        ...body,
        endpoint: body.endpoint || null,
        authHeader: null
      })
      .onConflictDoUpdate({
        target: schema.judgeRunners.key,
        set: {
          name: body.name,
          enabled: body.enabled,
          kind: body.kind,
          endpoint: body.endpoint || null,
          authHeader: null,
          concurrency: body.concurrency,
          sortOrder: body.sortOrder,
          updatedAt: new Date()
        }
      })
      .returning()

    return c.json(runner, 201)
  })

  app.post('/api/admin/runners/:id/check', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const [runner] = await db
      .select()
      .from(schema.judgeRunners)
      .where(eq(schema.judgeRunners.id, id))
      .limit(1)
    if (!runner) return c.notFound()
    if (runner.kind !== 'docker') {
      return c.json(
        { code: 'UNSUPPORTED_RUNNER', message: `Unsupported runner kind: ${runner.kind}` },
        400
      )
    }

    const result = await new DockerRunner({
      endpoint: runner.endpoint,
      authHeader: runner.authHeader
    }).check()

    return c.json({ ok: true, runnerId: runner.id, ...result })
  })
}

const createGroupSchema = z.object({
  key: z.string().regex(/^[a-z][a-z0-9_-]{1,63}$/),
  name: z.string().min(1).max(128),
  description: z.string().max(500).default('')
})

const updateUserSchema = z.object({
  disabled: z.boolean()
})

const addGroupUserSchema = z.object({
  userId: numericId,
  manager: z.boolean().default(false)
})

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

const runnerConfigSchema = z.object({
  key: z.string().regex(/^[a-z][a-z0-9_-]{1,63}$/),
  name: z.string().min(1).max(128),
  enabled: z.boolean().default(true),
  kind: z.enum(['docker']).default('docker'),
  endpoint: z.string().max(1000).optional(),
  concurrency: z.number().int().positive().default(2),
  sortOrder: z.number().int().min(0).default(100)
})
