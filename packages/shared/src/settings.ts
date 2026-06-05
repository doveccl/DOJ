export interface RuntimeSettings {
  registrationEnabled: boolean
  aiCoachingEnabled: boolean
  guestProblemsetVisible: boolean
  sourceOpenDefault: boolean
  outputLimitBytes: number
}

export const runtimeSettingsDefaults: RuntimeSettings = {
  registrationEnabled: true,
  aiCoachingEnabled: true,
  guestProblemsetVisible: true,
  sourceOpenDefault: false,
  outputLimitBytes: 64 * 1024 * 1024
}
