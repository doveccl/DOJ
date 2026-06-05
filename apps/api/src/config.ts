export const config = {
  port: Number(process.env.DOJ_API_PORT ?? 7974),
  jwtSecret: process.env.JWT_SECRET ?? 'dev-secret',
  aiCoachingEnabled: process.env.AI_COACHING_ENABLED !== '0',
  aiProvider: process.env.AI_PROVIDER ?? 'local-stub',
  openaiApiKey: process.env.OPENAI_API_KEY ?? '',
  openaiModel: process.env.OPENAI_MODEL ?? 'gpt-5-mini'
}
