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
    throw new Error(`Redis unavailable while REDIS is configured: ${error instanceof Error ? error.message : String(error)}`, {
      cause: error
    })
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

export async function redisSetJson<T>(key: string, value: T, ttlSeconds: number) {
  return redisSet(key, JSON.stringify(value), ttlSeconds)
}

export async function redisGetJson<T>(key: string) {
  const value = await redisGet(key)
  if (!value) return null
  try {
    return JSON.parse(value) as T
  } catch {
    return null
  }
}

export async function redisIncrWithTtl(key: string, ttlSeconds: number) {
  return withRedis(async (redis) => {
    const result = await redis.send('EVAL', [
      "local value = redis.call('INCR', KEYS[1]); if value == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]); end; return value",
      '1',
      key,
      String(ttlSeconds)
    ])
    return typeof result === 'number' ? result : Number(result)
  })
}

export async function redisCommand(command: string, args: string[] = []) {
  return withRedis((redis) => redis.send(command, args))
}
