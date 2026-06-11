import { eq, sql } from 'drizzle-orm'
import { createHash } from 'node:crypto'
import type { Context, MiddlewareHandler } from 'hono'
import { db, schema } from '@doj/db/client'
import { apiError } from './errors'
import { createSession, destroySession, getSessionUserId } from './session'
import { getRuntimeSettings } from './settings'

export interface AuthUser {
  id: number
  name: string
  email: string
  introduction: string
  admin: boolean
  disabled: boolean
  mustChangePassword: boolean
  avatarUrl: string
  groups: string[]
}

export async function hashPassword(password: string) {
  return Bun.password.hash(password, {
    algorithm: 'argon2id',
    memoryCost: 19456,
    timeCost: 2
  })
}

export async function verifyPassword(password: string, hash: string) {
  return Bun.password.verify(password, hash)
}

const dummyPasswordHash = hashPassword('doj-dummy-password')

export async function verifyPasswordOrDummy(password: string, hash: string | null | undefined) {
  return verifyPassword(password, hash ?? (await dummyPasswordHash))
}

export async function createToken(userId: number) {
  return createSession(userId)
}

export async function getAuthUser(userId: number): Promise<AuthUser | null> {
  const [user] = await db.select().from(schema.users).where(eq(schema.users.id, userId)).limit(1)
  if (!user || user.disabledAt) return null

  const groups = await db
    .select({ name: schema.groups.name })
    .from(schema.userGroups)
    .innerJoin(schema.groups, eq(schema.userGroups.groupId, schema.groups.id))
    .where(eq(schema.userGroups.userId, user.id))

  return {
    id: user.id,
    name: user.name,
    email: user.email,
    introduction: user.introduction,
    admin: user.admin,
    disabled: user.disabledAt !== null,
    mustChangePassword: user.mustChangePassword,
    avatarUrl: gravatarUrl(user.email),
    groups: groups.map((item) => item.name)
  }
}

export async function findUserByNameOrEmail(value: string) {
  const [user] = await db
    .select()
    .from(schema.users)
    .where(
      sql`lower(${schema.users.name}) = lower(${value}) or lower(${schema.users.email}) = lower(${value})`
    )
    .limit(1)

  return user ?? null
}

export async function requireAuthUser(c: Context) {
  const user = c.get('authUser') as AuthUser | undefined
  if (!user) throw new Error('auth user missing')
  return user
}

export async function requireGroup(c: Context, group: string) {
  const user = await requireAuthUser(c)
  if (group === 'admin' ? !user.admin : !user.groups.includes(group)) {
    return apiError(c, 403, 'FORBIDDEN', `Requires ${group} permission`)
  }
  return null
}

export async function requireAdmin(c: Context) {
  const user = await requireAuthUser(c)
  if (!user.admin) return apiError(c, 403, 'FORBIDDEN', 'Requires admin permission')
  return null
}

export async function getOptionalAuthUser(c: Context) {
  const header = c.req.header('authorization')
  const token = header?.startsWith('Bearer ') ? header.slice('Bearer '.length) : ''
  if (!token) return null

  const userId = await getSessionUserId(token)
  if (userId === null) return null
  const user = await getAuthUser(userId)
  if (!user) await destroySession(token)
  return user
}

export async function denyGuestAccess(c: Context, message = 'Sign in to view this content') {
  const settings = await getRuntimeSettings()
  if (settings.general.guestAccess) return null
  const user = await getOptionalAuthUser(c)
  if (user) return null
  return apiError(c, 401, 'UNAUTHORIZED', message)
}

export const authMiddleware: MiddlewareHandler = async (c, next) => {
  const header = c.req.header('authorization')
  const token = header?.startsWith('Bearer ') ? header.slice('Bearer '.length) : ''
  if (!token) {
    return apiError(c, 401, 'UNAUTHORIZED', 'Missing bearer token')
  }

  const userId = await getSessionUserId(token)
  const user = userId === null ? null : await getAuthUser(userId)
  if (!user) {
    if (userId !== null) await destroySession(token)
    return apiError(c, 401, 'UNAUTHORIZED', 'Invalid bearer token')
  }
  if (user.mustChangePassword && !isPasswordChangeRoute(c)) {
    return apiError(c, 403, 'MUST_CHANGE_PASSWORD', 'Password must be changed before continuing')
  }
  c.set('authUser', user)
  await next()
}

function isPasswordChangeRoute(c: Context) {
  const path = new URL(c.req.url).pathname
  if (c.req.method === 'GET' && path === '/api/auth/self') return true
  if (c.req.method === 'PATCH' && path === '/api/auth/self') return true
  if (c.req.method === 'POST' && path === '/api/auth/logout') return true
  return false
}

function gravatarUrl(email: string) {
  const hash = createHash('md5').update(email.trim().toLowerCase()).digest('hex')
  return `https://www.gravatar.com/avatar/${hash}?d=identicon&s=80`
}
