import { inArray, sql } from 'drizzle-orm'
import { runtimeSettingsDefaults, type RuntimeSettings } from '@doj/shared/settings'
import { db, schema } from './client'

const settingKeys = Object.keys(runtimeSettingsDefaults) as (keyof RuntimeSettings)[]

type RuntimeSettingsPatch = {
  general?: Partial<RuntimeSettings['general']>
  smtp?: Partial<RuntimeSettings['smtp']>
  ai?: Partial<RuntimeSettings['ai']>
}

export async function getRuntimeSettings(): Promise<RuntimeSettings> {
  const rows = await db
    .select()
    .from(schema.settings)
    .where(inArray(schema.settings.key, settingKeys))

  return normalizeRuntimeSettings(Object.fromEntries(rows.map((row) => [row.key, row.value])))
}

export async function updateRuntimeSettings(input: RuntimeSettingsPatch) {
  const current = await getRuntimeSettings()
  const next = normalizeRuntimeSettings({
    general: { ...current.general, ...input.general },
    smtp: { ...current.smtp, ...input.smtp },
    ai: { ...current.ai, ...input.ai }
  })
  const changedKeys = settingKeys.filter((key) => input[key] !== undefined)

  if (changedKeys.length) {
    await db
      .insert(schema.settings)
      .values(
        changedKeys.map((key) => ({
          key,
          value: next[key]
        }))
      )
      .onConflictDoUpdate({
        target: schema.settings.key,
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
    general: normalizeGeneral(input.general),
    smtp: normalizeSmtp(input.smtp),
    ai: normalizeAi(input.ai)
  }
}

function normalizeGeneral(value: unknown): RuntimeSettings['general'] {
  const input = isRecord(value) ? value : {}
  return {
    notice: readString(input.notice, runtimeSettingsDefaults.general.notice),
    signup: readBoolean(input.signup, runtimeSettingsDefaults.general.signup),
    publicCode: readBoolean(input.publicCode, runtimeSettingsDefaults.general.publicCode),
    guestAccess: readBoolean(input.guestAccess, runtimeSettingsDefaults.general.guestAccess)
  }
}

function normalizeSmtp(value: unknown): RuntimeSettings['smtp'] {
  const input = isRecord(value) ? value : {}
  return {
    enabled: readBoolean(input.enabled, runtimeSettingsDefaults.smtp.enabled),
    _host: readString(input._host, runtimeSettingsDefaults.smtp._host),
    _port: readPositiveInteger(input._port, runtimeSettingsDefaults.smtp._port),
    _user: readString(input._user, runtimeSettingsDefaults.smtp._user),
    _password: readString(input._password, runtimeSettingsDefaults.smtp._password),
    from: readString(input.from, runtimeSettingsDefaults.smtp.from)
  }
}

function normalizeAi(value: unknown): RuntimeSettings['ai'] {
  const input = isRecord(value) ? value : {}
  return {
    enabled: readBoolean(input.enabled, runtimeSettingsDefaults.ai.enabled),
    _baseUrl: readString(input._baseUrl, runtimeSettingsDefaults.ai._baseUrl),
    _model: readString(input._model, runtimeSettingsDefaults.ai._model),
    _apiKey: readString(input._apiKey, runtimeSettingsDefaults.ai._apiKey)
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
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
