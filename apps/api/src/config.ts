export const config = {
  port: Number(process.env.PORT ?? 7974),
  redisUrl: process.env.REDIS,
  sessionTtlSeconds: Number(process.env.SESSION_TTL_SECONDS ?? 60 * 60 * 24 * 30)
}
