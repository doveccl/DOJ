import { z } from 'zod'
import { ensureRuntimeSettings, getRuntimeSettings, updateRuntimeSettings } from '@doj/db/settings'

export const runtimeSettingsSchema = z.object({
  general: z.object({
    notice: z.string().max(20_000).default(''),
    signup: z.boolean().default(false),
    publicCode: z.boolean().default(false),
    guestAccess: z.boolean().default(true)
  }),
  smtp: z.object({
    enabled: z.boolean().default(false),
    _host: z.string().max(400).default(''),
    _port: z.number().int().positive().default(587),
    _user: z.string().max(400).default(''),
    _password: z.string().max(400).default(''),
    from: z.string().max(320).default('')
  }),
  ai: z.object({
    enabled: z.boolean().default(false),
    _baseUrl: z.string().max(400).default(''),
    _model: z.string().max(200).default(''),
    _apiKey: z.string().max(400).default('')
  })
})

export { ensureRuntimeSettings, getRuntimeSettings, updateRuntimeSettings }
