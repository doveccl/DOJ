import { RedisClient } from 'bun'
import { config } from './config'

let client: RedisClient | null = null
let unavailableUntil = 0

function getClient() {
  if (!config.redisUrl || Date.now() < unavailableUntil) return null
  if (client) return client

  client = new RedisClient(config.redisUrl, {
    connectionTimeout: 500
  })
  client.onclose = () => {
    unavailableUntil = Date.now() + 5000
  }
  return client
}

async function withRedis<T>(operation: (redis: RedisClient) => Promise<T>) {
  const redis = getClient()
  if (!redis) return null

  try {
    return await operation(redis)
  } catch (error) {
    unavailableUntil = Date.now() + 5000
    console.warn('Redis unavailable, falling back to process memory:', error)
    return null
  }
}

export async function redisGet(key: string) {
  return withRedis((redis) => redis.get(key))
}

export async function redisSet(key: string, value: string, ttlSeconds: number) {
  const result = await withRedis(async (redis) => {
    await redis.set(key, value)
    await redis.expire(key, ttlSeconds)
    return true
  })
  return result === true
}

export async function redisDel(key: string) {
  await withRedis((redis) => redis.del(key))
}

export async function redisIncrWithTtl(key: string, ttlSeconds: number) {
  return withRedis(async (redis) => {
    const count = await redis.incr(key)
    if (count === 1) await redis.expire(key, ttlSeconds)
    return count
  })
}
