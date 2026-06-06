import { Hono } from 'hono'
import { eq } from 'drizzle-orm'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { checkRateLimit, clientIp } from '../rate-limit'
import {
  authMiddleware,
  createToken,
  findUserByNameOrEmail,
  getAuthUser,
  getGroupByKey,
  hashPassword,
  requireAuthUser,
  requireGroup,
  verifyPassword
} from '../auth'
import { getRuntimeSettings, runtimeSettingsSchema, updateRuntimeSettings } from '../settings'
import { destroySession } from '../session'

export function registerAuthRoutes(app: Hono) {
  app.post('/api/auth/register', async (c) => {
    const settings = await getRuntimeSettings()
    if (!settings.registrationEnabled) {
      return c.json({ code: 'REGISTRATION_DISABLED', message: 'Registration is disabled' }, 403)
    }

    const body = registerSchema.parse(await c.req.json())
    const rateLimited = await checkRateLimit(c, 'register', clientIp(c), 200, 60 * 60 * 1000)
    if (rateLimited) return rateLimited

    const existing = await findUserByNameOrEmail(body.name)
    if (existing || (await findUserByNameOrEmail(body.email))) {
      return c.json({ code: 'USER_EXISTS', message: 'User name or email already exists' }, 409)
    }

    const group = await getGroupByKey('user')
    if (!group) throw new Error('builtin group missing: user')

    const result = await db.transaction(async (tx) => {
      const [user] = await tx
        .insert(schema.users)
        .values({
          name: body.name,
          email: body.email,
          passwordHash: await hashPassword(body.password)
        })
        .returning()

      await tx.insert(schema.userGroups).values({ userId: user.id, groupId: group.id })
      return user
    })

    const user = await getAuthUser(result.id)
    if (!user) throw new Error('registered user missing')

    return c.json({ token: await createToken(user.id), user }, 201)
  })

  app.post('/api/auth/login', async (c) => {
    const body = loginSchema.parse(await c.req.json())
    const rateLimited = await checkRateLimit(
      c,
      'login',
      `${clientIp(c)}:${body.user.toLowerCase()}`,
      40,
      10 * 60 * 1000
    )
    if (rateLimited) return rateLimited

    const record = await findUserByNameOrEmail(body.user)
    if (!record || !(await verifyPassword(body.password, record.passwordHash))) {
      return c.json({ code: 'INVALID_CREDENTIALS', message: 'Invalid user or password' }, 401)
    }

    const user = await getAuthUser(record.id)
    if (!user) return c.json({ code: 'USER_DISABLED', message: 'User is disabled' }, 403)

    return c.json({ token: await createToken(user.id), user })
  })

  app.get('/api/auth/self', authMiddleware, async (c) => c.json(await requireAuthUser(c)))

  app.patch('/api/auth/self', authMiddleware, async (c) => {
    const current = await requireAuthUser(c)
    const body = updateProfileSchema.parse(await c.req.json())

    const patch: { introduction?: string; passwordHash?: string } = {}
    if (body.introduction !== undefined) patch.introduction = body.introduction
    if (body.password) patch.passwordHash = await hashPassword(body.password)

    if (Object.keys(patch).length) {
      await db
        .update(schema.users)
        .set({ ...patch, updatedAt: new Date() })
        .where(eq(schema.users.id, current.id))
    }

    return c.json(await getAuthUser(current.id))
  })

  app.post('/api/auth/logout', authMiddleware, async (c) => {
    const token = c.req.header('authorization')?.slice('Bearer '.length) ?? ''
    if (token) await destroySession(token)
    return c.json({ ok: true })
  })

  app.get('/api/admin/settings', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const settings = await getRuntimeSettings()
    // Never return the raw secret; expose only whether it is configured.
    const { aiApiKey, ...rest } = settings
    return c.json({ ...rest, aiApiKeySet: aiApiKey.length > 0 })
  })

  app.patch('/api/admin/settings', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const body = updateSettingsSchema.parse(await c.req.json())
    // An empty/omitted aiApiKey means "keep existing"; only overwrite when provided.
    if (body.aiApiKey === undefined || body.aiApiKey === '') delete body.aiApiKey
    const settings = await updateRuntimeSettings(body)
    const { aiApiKey, ...rest } = settings
    return c.json({ ...rest, aiApiKeySet: aiApiKey.length > 0 })
  })
}

const registerSchema = z.object({
  name: z.string().regex(/^[a-zA-Z0-9][a-zA-Z0-9_]{2,31}$/),
  email: z.email(),
  password: z.string().min(8).max(128)
})

const loginSchema = z.object({
  user: z.string().min(1),
  password: z.string().min(1)
})

const updateProfileSchema = z.object({
  introduction: z.string().max(500).optional(),
  password: z.string().min(8).max(128).optional()
})

const updateSettingsSchema = runtimeSettingsSchema.partial()
