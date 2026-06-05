export const config = {
  port: Number(process.env.DOJ_API_PORT ?? 7974),
  redisUrl: process.env.REDIS_URL ?? 'redis://localhost:6379',
  sessionTtlSeconds: Number(process.env.SESSION_TTL_SECONDS ?? 60 * 60 * 24 * 30),
  registrationEnabled: process.env.REGISTRATION_ENABLED !== '0',
  aiCoachingEnabled: process.env.AI_COACHING_ENABLED !== '0',
  aiProvider: process.env.AI_PROVIDER ?? 'local-rules',
  openaiApiKey: process.env.OPENAI_API_KEY ?? '',
  openaiModel: process.env.OPENAI_MODEL ?? 'gpt-5-mini'
}
