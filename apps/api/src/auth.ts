import { eq, sql } from 'drizzle-orm'
import { sign, verify } from 'hono/jwt'
import type { Context, MiddlewareHandler } from 'hono'
import { db, schema } from '@doj/db/client'
import { config } from './config'

export interface AuthUser {
  id: number
  name: string
  email: string
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

export async function createToken(userId: number) {
  return sign(
    {
      sub: String(userId),
      exp: Math.floor(Date.now() / 1000) + 60 * 60 * 24 * 30
    },
    config.jwtSecret
  )
}

export async function getAuthUser(userId: number): Promise<AuthUser | null> {
  const [user] = await db.select().from(schema.users).where(eq(schema.users.id, userId)).limit(1)
  if (!user || user.disabledAt) return null

  const groups = await db
    .select({ key: schema.groups.key })
    .from(schema.userGroups)
    .innerJoin(schema.groups, eq(schema.userGroups.groupId, schema.groups.id))
    .where(eq(schema.userGroups.userId, user.id))

  return {
    id: user.id,
    name: user.name,
    email: user.email,
    groups: groups.map((item) => item.key)
  }
}

export async function findUserByNameOrEmail(value: string) {
  const [user] = await db
    .select()
    .from(schema.users)
    .where(sql`lower(${schema.users.name}) = lower(${value}) or lower(${schema.users.email}) = lower(${value})`)
    .limit(1)

  return user ?? null
}

export async function getGroupByKey(key: string) {
  const [group] = await db.select().from(schema.groups).where(eq(schema.groups.key, key)).limit(1)
  return group ?? null
}

export async function requireAuthUser(c: Context) {
  const user = c.get('authUser') as AuthUser | undefined
  if (!user) throw new Error('auth user missing')
  return user
}

export async function requireGroup(c: Context, group: string) {
  const user = await requireAuthUser(c)
  if (!user.groups.includes(group)) {
    return c.json({ code: 'FORBIDDEN', message: `Requires ${group} group` }, 403)
  }
  return null
}

export const authMiddleware: MiddlewareHandler = async (c, next) => {
  const header = c.req.header('authorization')
  const token = header?.startsWith('Bearer ') ? header.slice('Bearer '.length) : ''
  if (!token) {
    return c.json({ code: 'UNAUTHORIZED', message: 'Missing bearer token' }, 401)
  }

  try {
    const payload = await verify(token, config.jwtSecret, 'HS256')
    const userId = typeof payload.sub === 'string' ? Number(payload.sub) : NaN
    const user = Number.isInteger(userId) ? await getAuthUser(userId) : null
    if (!user) return c.json({ code: 'UNAUTHORIZED', message: 'Invalid bearer token' }, 401)
    c.set('authUser', user)
    await next()
  } catch {
    return c.json({ code: 'UNAUTHORIZED', message: 'Invalid bearer token' }, 401)
  }
}
