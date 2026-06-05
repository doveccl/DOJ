import { inArray, sql } from 'drizzle-orm'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { runtimeSettingsDefaults, type RuntimeSettings } from '@doj/shared/settings'

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

const settingKeys = Object.keys(runtimeSettingsDefaults) as (keyof RuntimeSettings)[]

export async function getRuntimeSettings(): Promise<RuntimeSettings> {
  const rows = await db
    .select()
    .from(schema.systemSettings)
    .where(inArray(schema.systemSettings.key, settingKeys))

  return runtimeSettingsSchema.parse(Object.fromEntries(rows.map((row) => [row.key, row.value])))
}

export async function updateRuntimeSettings(input: Partial<RuntimeSettings>) {
  const current = await getRuntimeSettings()
  const next = runtimeSettingsSchema.parse({ ...current, ...input })
  const changedKeys = settingKeys.filter((key) => input[key] !== undefined)

  if (changedKeys.length) {
    await db
      .insert(schema.systemSettings)
      .values(
        changedKeys.map((key) => ({
          key,
          value: next[key]
        }))
      )
      .onConflictDoUpdate({
        target: schema.systemSettings.key,
        set: {
          value: sql`excluded.value`,
          updatedAt: new Date()
        }
      })
  }

  return next
}

export async function ensureRuntimeSettings() {
  await updateRuntimeSettings(runtimeSettingsDefaults)
}
