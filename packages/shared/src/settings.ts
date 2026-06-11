export interface RuntimeSettings {
  general: {
    notice: string
    signup: boolean
    publicCode: boolean
    guestAccess: boolean
  }
  smtp: {
    enabled: boolean
    _host: string
    _port: number
    _user: string
    _password: string
    from: string
  }
  ai: {
    enabled: boolean
    _baseUrl: string
    _model: string
    _apiKey: string
  }
}

export const runtimeSettingsDefaults: RuntimeSettings = {
  general: {
    notice: '',
    signup: false,
    publicCode: false,
    guestAccess: true
  },
  smtp: {
    enabled: false,
    _host: '',
    _port: 587,
    _user: '',
    _password: '',
    from: ''
  },
  ai: {
    enabled: false,
    _baseUrl: '',
    _model: '',
    _apiKey: ''
  }
}

export const privateSettingFields = new Set([
  '_host',
  '_port',
  '_user',
  '_password',
  '_baseUrl',
  '_model',
  '_apiKey'
])
