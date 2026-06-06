export const config = {
  port: Number(process.env.DOJ_API_PORT ?? 7974),
  redisUrl: process.env.REDIS_URL ?? 'redis://localhost:6379',
  sessionTtlSeconds: Number(process.env.SESSION_TTL_SECONDS ?? 60 * 60 * 24 * 30)
}
