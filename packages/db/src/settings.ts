import { inArray, sql } from 'drizzle-orm'
import { runtimeSettingsDefaults, type RuntimeSettings } from '@doj/shared/settings'
import { db, schema } from './client'

const settingKeys = Object.keys(runtimeSettingsDefaults) as (keyof RuntimeSettings)[]

export async function getRuntimeSettings(): Promise<RuntimeSettings> {
  const rows = await db
    .select()
    .from(schema.systemSettings)
    .where(inArray(schema.systemSettings.key, settingKeys))

  return normalizeRuntimeSettings(Object.fromEntries(rows.map((row) => [row.key, row.value])))
}

export async function updateRuntimeSettings(input: Partial<RuntimeSettings>) {
  const current = await getRuntimeSettings()
  const next = normalizeRuntimeSettings({ ...current, ...input })
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

function normalizeRuntimeSettings(input: Record<string, unknown>): RuntimeSettings {
  return {
    registrationEnabled: readBoolean(input.registrationEnabled, true),
    registrationInviteCode: readString(
      input.registrationInviteCode,
      runtimeSettingsDefaults.registrationInviteCode
    ),
    aiCoachingEnabled: readBoolean(input.aiCoachingEnabled, true),
    guestProblemsetVisible: readBoolean(input.guestProblemsetVisible, true),
    sourceOpenDefault: readBoolean(input.sourceOpenDefault, false),
    outputLimitBytes: readPositiveInteger(
      input.outputLimitBytes,
      runtimeSettingsDefaults.outputLimitBytes
    ),
    aiProvider: input.aiProvider === 'openai' ? 'openai' : 'local-rules',
    aiApiKey: readString(input.aiApiKey, runtimeSettingsDefaults.aiApiKey),
    aiBaseUrl: readString(input.aiBaseUrl, runtimeSettingsDefaults.aiBaseUrl),
    aiModel: readString(input.aiModel, runtimeSettingsDefaults.aiModel)
  }
}

function readBoolean(value: unknown, fallback: boolean) {
  return typeof value === 'boolean' ? value : fallback
}

function readString(value: unknown, fallback: string) {
  return typeof value === 'string' ? value : fallback
}

function readPositiveInteger(value: unknown, fallback: number) {
  return typeof value === 'number' && Number.isInteger(value) && value > 0 ? value : fallback
}
