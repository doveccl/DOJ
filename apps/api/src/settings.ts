import { z } from 'zod'
import { ensureRuntimeSettings, getRuntimeSettings, updateRuntimeSettings } from '@doj/db/settings'

export const runtimeSettingsSchema = z.object({
  registrationEnabled: z.boolean().default(true),
  aiCoachingEnabled: z.boolean().default(true),
  guestProblemsetVisible: z.boolean().default(true),
  sourceOpenDefault: z.boolean().default(false),
  outputLimitBytes: z
    .number()
    .int()
    .positive()
    .default(64 * 1024 * 1024)
})

export { ensureRuntimeSettings, getRuntimeSettings, updateRuntimeSettings }
