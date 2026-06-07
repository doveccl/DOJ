import { z } from 'zod'
import { ensureRuntimeSettings, getRuntimeSettings, updateRuntimeSettings } from '@doj/db/settings'

export const runtimeSettingsSchema = z.object({
  registrationEnabled: z.boolean().default(true),
  registrationInviteCode: z.string().max(128).default(''),
  aiCoachingEnabled: z.boolean().default(true),
  guestProblemsetVisible: z.boolean().default(true),
  sourceOpenDefault: z.boolean().default(false),
  outputLimitBytes: z
    .number()
    .int()
    .positive()
    .default(64 * 1024 * 1024),
  aiProvider: z.enum(['local-rules', 'openai']).default('local-rules'),
  aiApiKey: z.string().max(400).default(''),
  aiBaseUrl: z.string().max(400).default('https://api.openai.com/v1'),
  aiModel: z.string().max(200).default('gpt-5-mini')
})

export { ensureRuntimeSettings, getRuntimeSettings, updateRuntimeSettings }
