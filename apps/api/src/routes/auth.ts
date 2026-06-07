import { timingSafeEqual } from 'node:crypto'
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
  verifyPasswordOrDummy
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
    if (
      settings.registrationInviteCode &&
      !inviteCodeMatches(settings.registrationInviteCode, body.inviteCode)
    ) {
      return c.json({ code: 'INVALID_INVITE_CODE', message: 'Invalid invite code' }, 403)
    }

    const existing = await findUserByNameOrEmail(body.name)
    if (existing || (await findUserByNameOrEmail(body.email))) {
      return c.json({ code: 'USER_EXISTS', message: 'User name or email already exists' }, 409)
    }

    const group = await getGroupByKey('user')
    if (!group) throw new Error('builtin group missing: user')

    const result = await db
      .transaction(async (tx) => {
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
      .catch((error: unknown) => {
        if (isUniqueViolation(error)) return null
        throw error
      })
    if (!result) {
      return c.json({ code: 'USER_EXISTS', message: 'User name or email already exists' }, 409)
    }

    const user = await getAuthUser(result.id)
    if (!user) throw new Error('registered user missing')

    return c.json({ token: await createToken(user.id), user }, 201)
  })

  app.post('/api/auth/login', async (c) => {
    const body = loginSchema.parse(await c.req.json())
    const ip = clientIp(c)
    const ipLimited = await checkRateLimit(c, 'login-ip', ip, 120, 10 * 60 * 1000)
    if (ipLimited) return ipLimited

    const rateLimited = await checkRateLimit(
      c,
      'login',
      `${ip}:${body.user.toLowerCase()}`,
      40,
      10 * 60 * 1000
    )
    if (rateLimited) return rateLimited

    const record = await findUserByNameOrEmail(body.user)
    const passwordOk = await verifyPasswordOrDummy(body.password, record?.passwordHash)
    if (!record || !passwordOk) {
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
    // Never return raw secrets; expose only whether each secret is configured.
    const { aiApiKey, registrationInviteCode, ...rest } = settings
    return c.json({
      ...rest,
      aiApiKeySet: aiApiKey.length > 0,
      registrationInviteRequired: registrationInviteCode.length > 0
    })
  })

  app.patch('/api/admin/settings', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const body = updateSettingsSchema.parse(await c.req.json())
    // An empty/omitted aiApiKey means "keep existing"; only overwrite when provided.
    if (body.aiApiKey === undefined || body.aiApiKey === '') delete body.aiApiKey
    // An omitted invite code means "keep existing"; an empty string explicitly disables it.
    if (body.registrationInviteCode === undefined) delete body.registrationInviteCode
    const settings = await updateRuntimeSettings(body)
    const { aiApiKey, registrationInviteCode, ...rest } = settings
    return c.json({
      ...rest,
      aiApiKeySet: aiApiKey.length > 0,
      registrationInviteRequired: registrationInviteCode.length > 0
    })
  })
}

const registerSchema = z.object({
  name: z.string().regex(/^[a-zA-Z0-9][a-zA-Z0-9_]{2,31}$/),
  email: z.email(),
  password: z.string().min(8).max(128),
  inviteCode: z.string().max(128).optional()
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

function isUniqueViolation(error: unknown) {
  return (
    typeof error === 'object' &&
    error !== null &&
    'code' in error &&
    (error as { code?: unknown }).code === '23505'
  )
}

function inviteCodeMatches(expected: string, actual: string | undefined) {
  if (!actual) return false
  const expectedBytes = Buffer.from(expected)
  const actualBytes = Buffer.from(actual)
  return expectedBytes.length === actualBytes.length && timingSafeEqual(expectedBytes, actualBytes)
}
