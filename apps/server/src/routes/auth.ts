import { Hono } from 'hono'
import { eq } from 'drizzle-orm'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { apiError } from '../errors'
import { checkRateLimit, clientIp } from '../rate-limit'
import { redisGetJson, redisSetJson } from '../redis'
import {
  authMiddleware,
  createToken,
  findUserByNameOrEmail,
  getAuthUser,
  hashPassword,
  requireAuthUser,
  requireGroup,
  verifyPasswordOrDummy
} from '../auth'
import { getRuntimeSettings, updateRuntimeSettings } from '../settings'
import { destroySession } from '../session'

const emailCodeTtlSeconds = 10 * 60
const memoryEmailCodes = new Map<string, { code: string; userId: number | null; expiresAt: number }>()

export function registerAuthRoutes(app: Hono) {
  app.post('/api/auth/register', async (c) => {
    const settings = await getRuntimeSettings()
    if (!settings.general.signup) {
      return apiError(c, 403, 'REGISTRATION_DISABLED', 'Registration is disabled')
    }
    if (!smtpConfigured(settings)) {
      return apiError(c, 400, 'SMTP_REQUIRED', 'SMTP must be configured before registration')
    }

    const body = registerSchema.parse(await c.req.json())
    const rateLimited = await checkRateLimit(c, 'register', clientIp(c), 200, 60 * 60 * 1000)
    if (rateLimited) return rateLimited
    const codeOk = await verifyEmailCode('register', body.email, body.code, null)
    if (!codeOk) return apiError(c, 400, 'INVALID_EMAIL_CODE', 'Email verification code is invalid')
    const existing = await findUserByNameOrEmail(body.name)
    if (existing || (await findUserByNameOrEmail(body.email))) {
      return apiError(c, 409, 'EMAIL_EXISTS', 'Email already exists')
    }

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

        return user
      })
      .catch((error: unknown) => {
        if (isUniqueViolation(error)) return null
        throw error
      })
    if (!result) {
      return apiError(c, 409, 'EMAIL_EXISTS', 'Email already exists')
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
      return apiError(c, 401, 'INVALID_CREDENTIALS', 'Invalid user or password')
    }

    const user = await getAuthUser(record.id)
    if (!user) return apiError(c, 403, 'USER_DISABLED', 'User is disabled')

    return c.json({ token: await createToken(user.id), user })
  })

  app.get('/api/auth/self', authMiddleware, async (c) => c.json(await requireAuthUser(c)))

  app.patch('/api/auth/self', authMiddleware, async (c) => {
    const current = await requireAuthUser(c)
    const body = updateProfileSchema.parse(await c.req.json())

    const patch: { introduction?: string; passwordHash?: string; mustChangePassword?: boolean } = {}
    if (body.introduction !== undefined) patch.introduction = body.introduction
    if (body.password) {
      if (!body.currentPassword) {
        return apiError(c, 400, 'CURRENT_PASSWORD_REQUIRED', 'Current password is required')
      }
      const record = await findUserByNameOrEmail(current.email)
      const ok = await verifyPasswordOrDummy(body.currentPassword, record?.passwordHash)
      if (!ok) {
        return apiError(c, 403, 'INVALID_CURRENT_PASSWORD', 'Current password is incorrect')
      }
      patch.passwordHash = await hashPassword(body.password)
      patch.mustChangePassword = false
    }

    if (Object.keys(patch).length) {
      await db
        .update(schema.users)
        .set({ ...patch, updatedAt: new Date() })
        .where(eq(schema.users.id, current.id))
    }

    return c.json(await getAuthUser(current.id))
  })

  app.post('/api/auth/email-code', async (c) => {
    const body = emailCodeSchema.parse(await c.req.json())
    const settings = await getRuntimeSettings()
    if (!smtpConfigured(settings)) {
      return apiError(c, 400, 'SMTP_REQUIRED', 'SMTP must be configured before sending email codes')
    }
    if (body.purpose === 'register' && (await findUserByNameOrEmail(body.email))) {
      return apiError(c, 409, 'EMAIL_EXISTS', 'Email already exists')
    }

    const code = createEmailCode()
    const stored = await storeEmailCode(body.purpose, body.email, code, null)
    if (!stored) return apiError(c, 500, 'SMTP_SEND_FAILED', 'Failed to send email verification code')
    console.info(`Email code for ${body.purpose}:${body.email}: ${code}`)
    return c.json({ ok: true })
  })

  app.patch('/api/auth/email', authMiddleware, async (c) => {
    const current = await requireAuthUser(c)
    const body = changeEmailSchema.parse(await c.req.json())
    const existing = await findUserByNameOrEmail(body.email)
    if (existing && existing.id !== current.id) {
      return apiError(c, 409, 'EMAIL_EXISTS', 'Email already exists')
    }

    await db
      .update(schema.users)
      .set({ email: body.email, updatedAt: new Date() })
      .where(eq(schema.users.id, current.id))
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
    return c.json(redactSettings(settings))
  })

  app.patch('/api/admin/settings', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const body = updateSettingsSchema.parse(await c.req.json())
    if (body.general?.signup) {
      const current = await getRuntimeSettings()
      const nextSmtp = { ...current.smtp, ...body.smtp }
      if (!smtpConfigured({ ...current, smtp: nextSmtp })) {
        return apiError(c, 400, 'SMTP_REQUIRED', 'SMTP must be configured before enabling signup')
      }
    }
    const settings = await updateRuntimeSettings(body)
    return c.json(redactSettings(settings))
  })
}

const registerSchema = z.object({
  name: z.string().regex(/^[a-zA-Z0-9][a-zA-Z0-9_]{2,31}$/),
  email: z.email().transform(normalizeEmail),
  password: z.string().min(8).max(128),
  code: z.string().min(4).max(12)
})

const loginSchema = z.object({
  user: z.string().min(1),
  password: z.string().min(1)
})

const updateProfileSchema = z.object({
  introduction: z.string().max(160).optional(),
  currentPassword: z.string().min(1).optional(),
  password: z.string().min(8).max(128).optional()
})

const emailCodeSchema = z.object({
  purpose: z.literal('register'),
  email: z.email().transform(normalizeEmail)
})

const changeEmailSchema = z.object({
  email: z.email().transform(normalizeEmail)
})

const updateSettingsSchema = z.object({
  general: z
    .object({
      notice: z.string().max(20_000),
      signup: z.boolean(),
      publicCode: z.boolean(),
      guestAccess: z.boolean()
    })
    .partial()
    .optional(),
  smtp: z
    .object({
      enabled: z.boolean(),
      _host: z.string().max(400),
      _port: z.number().int().positive(),
      _user: z.string().max(400),
      _password: z.string().max(400),
      from: z.string().max(320)
    })
    .partial()
    .optional(),
  ai: z
    .object({
      enabled: z.boolean(),
      _baseUrl: z.string().max(400),
      _model: z.string().max(200),
      _apiKey: z.string().max(400)
    })
    .partial()
    .optional()
})

function isUniqueViolation(error: unknown) {
  return (
    typeof error === 'object' &&
    error !== null &&
    'code' in error &&
    (error as { code?: unknown }).code === '23505'
  )
}

function redactSettings(settings: Awaited<ReturnType<typeof getRuntimeSettings>>) {
  return {
    general: settings.general,
    smtp: {
      enabled: settings.smtp.enabled,
      hostSet: Boolean(settings.smtp._host),
      portSet: Boolean(settings.smtp._port),
      userSet: Boolean(settings.smtp._user),
      passwordSet: Boolean(settings.smtp._password),
      from: settings.smtp.from
    },
    ai: {
      enabled: settings.ai.enabled,
      baseUrlSet: Boolean(settings.ai._baseUrl),
      modelSet: Boolean(settings.ai._model),
      apiKeySet: Boolean(settings.ai._apiKey)
    }
  }
}

function normalizeEmail(email: string) {
  return email.trim().toLowerCase()
}

function smtpConfigured(settings: Awaited<ReturnType<typeof getRuntimeSettings>>) {
  return settings.smtp.enabled && Boolean(settings.smtp._host && settings.smtp.from)
}

function createEmailCode() {
  const bytes = crypto.getRandomValues(new Uint8Array(4))
  const value = bytes.reduce((acc, byte) => (acc << 8) + byte, 0) % 1_000_000
  return value.toString().padStart(6, '0')
}

function emailCodeKey(purpose: string, email: string) {
  return `emailCode:${purpose}:${email}`
}

async function storeEmailCode(purpose: string, email: string, code: string, userId: number | null) {
  const value = { code, userId, newEmail: email, createdAt: new Date().toISOString() }
  const stored = await redisSetJson(emailCodeKey(purpose, email), value, emailCodeTtlSeconds)
  if (!stored) {
    memoryEmailCodes.set(emailCodeKey(purpose, email), {
      code,
      userId,
      expiresAt: Date.now() + emailCodeTtlSeconds * 1000
    })
  }
  return true
}

async function verifyEmailCode(
  purpose: 'register',
  email: string,
  code: string,
  userId: number | null
) {
  const key = emailCodeKey(purpose, email)
  const stored = await redisGetJson<{ code: string; userId: number | null }>(key)
  const fallback = memoryEmailCodes.get(key)
  const value =
    stored ??
    (fallback && fallback.expiresAt > Date.now()
      ? { code: fallback.code, userId: fallback.userId }
      : null)
  return Boolean(value && value.code === code && value.userId === userId)
}
