import type { Context } from 'hono'
import { config } from './config'
import { redisIncrWithTtl } from './redis'

const memoryBuckets = new Map<string, number[]>()
let checks = 0

export function clientIp(c: Context) {
  const forwardedFor = c.req.header('x-forwarded-for')?.split(',')[0]?.trim()
  return forwardedFor || c.req.header('x-real-ip') || 'local'
}

export async function checkRateLimit(
  c: Context,
  scope: string,
  key: string,
  limit: number,
  windowMs: number
) {
  const bucketKey = `doj:rate:${scope}:${key}`
  const windowSeconds = Math.max(1, Math.ceil(windowMs / 1000))
  const redisCount = await redisIncrWithTtl(bucketKey, windowSeconds)

  if (typeof redisCount === 'number') {
    if (redisCount > limit) {
      c.header('retry-after', String(windowSeconds))
      return c.json(
        { code: 'RATE_LIMITED', message: 'Too many attempts. Please try again later.' },
        429
      )
    }
    return null
  }

  if (config.redisUrl) {
    return c.json(
      {
        code: 'RATE_LIMIT_UNAVAILABLE',
        message: 'Rate limit backend is unavailable. Please try again later.'
      },
      503
    )
  }

  return checkMemoryRateLimit(c, bucketKey, limit, windowMs)
}

function checkMemoryRateLimit(c: Context, bucketKey: string, limit: number, windowMs: number) {
  const now = Date.now()
  checks += 1
  if (checks % 1000 === 0) pruneMemoryBuckets(now)

  const recent = (memoryBuckets.get(bucketKey) ?? []).filter((time) => now - time < windowMs)
  if (recent.length >= limit) {
    const retryAfterSeconds = Math.max(1, Math.ceil((windowMs - (now - recent[0])) / 1000))
    c.header('retry-after', String(retryAfterSeconds))
    return c.json(
      { code: 'RATE_LIMITED', message: 'Too many attempts. Please try again later.' },
      429
    )
  }

  recent.push(now)
  memoryBuckets.set(bucketKey, recent)
  return null
}

function pruneMemoryBuckets(now: number) {
  for (const [key, timestamps] of memoryBuckets.entries()) {
    const recent = timestamps.filter((time) => now - time < 60 * 60 * 1000)
    if (recent.length) {
      memoryBuckets.set(key, recent)
    } else {
      memoryBuckets.delete(key)
    }
  }
}
