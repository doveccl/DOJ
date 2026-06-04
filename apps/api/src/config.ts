export const config = {
  port: Number(process.env.DOJ_API_PORT ?? 7974),
  jwtSecret: process.env.JWT_SECRET ?? 'dev-secret',
  aiCoachingEnabled: process.env.AI_COACHING_ENABLED !== '0'
}
