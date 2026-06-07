export interface RuntimeSettings {
  registrationEnabled: boolean
  registrationInviteCode: string
  aiCoachingEnabled: boolean
  guestProblemsetVisible: boolean
  sourceOpenDefault: boolean
  outputLimitBytes: number
  aiProvider: 'local-rules' | 'openai'
  aiApiKey: string
  aiBaseUrl: string
  aiModel: string
}

export const runtimeSettingsDefaults: RuntimeSettings = {
  registrationEnabled: true,
  registrationInviteCode: '',
  aiCoachingEnabled: true,
  guestProblemsetVisible: true,
  sourceOpenDefault: false,
  outputLimitBytes: 64 * 1024 * 1024,
  aiProvider: 'local-rules',
  aiApiKey: '',
  aiBaseUrl: 'https://api.openai.com/v1',
  aiModel: 'gpt-5-mini'
}

// Keys that must never be sent to non-admin clients or shown in plain text.
export const secretSettingKeys: (keyof RuntimeSettings)[] = ['aiApiKey', 'registrationInviteCode']
