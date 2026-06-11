import { config } from './config'
import { redisDel, redisGet, redisSet } from './redis'

const memorySessions = new Map<string, { userId: number; expiresAt: number }>()

export async function createSession(userId: number) {
  const token = randomToken()
  const key = sessionKey(token)
  const stored = await redisSet(key, String(userId), config.sessionTtlSeconds)

  if (!stored) {
    if (config.redisUrl) {
      throw new Error('session backend unavailable')
    }
    memorySessions.set(token, {
      userId,
      expiresAt: Date.now() + config.sessionTtlSeconds * 1000
    })
  }

  return token
}

export async function getSessionUserId(token: string) {
  const fromRedis = await redisGet(sessionKey(token))
  if (fromRedis) {
    await redisSet(sessionKey(token), fromRedis, config.sessionTtlSeconds)
    const userId = Number(fromRedis)
    return Number.isInteger(userId) ? userId : null
  }

  if (config.redisUrl) return null

  const session = memorySessions.get(token)
  if (!session) return null
  if (session.expiresAt <= Date.now()) {
    memorySessions.delete(token)
    return null
  }

  session.expiresAt = Date.now() + config.sessionTtlSeconds * 1000
  return session.userId
}

export async function destroySession(token: string) {
  memorySessions.delete(token)
  await redisDel(sessionKey(token))
}

function sessionKey(token: string) {
  return `session:${token}`
}

function randomToken() {
  const bytes = crypto.getRandomValues(new Uint8Array(32))
  return Buffer.from(bytes).toString('base64url')
}
