const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const adminUser = process.env.DOJ_ADMIN_NAME ?? 'admin'
const adminPassword = process.env.DOJ_ADMIN_PASSWORD ?? 'admin12345'
const runId = crypto.randomUUID()

const admin = await loginAdmin()
const original = await apiFetch<Record<string, unknown>>('/api/admin/settings', {
  token: admin.token
})

try {
  if (!original.registrationInviteRequired) {
    const inviteCode = `invite_${runId.slice(0, 8)}`
    const inviteSettings = await apiFetch<Record<string, unknown>>('/api/admin/settings', {
      token: admin.token,
      method: 'PATCH',
      body: { registrationInviteCode: inviteCode }
    })
    if (inviteSettings.registrationInviteRequired !== true) {
      throw new Error('registration invite setting did not update')
    }

    const inviteConfig = await apiFetch<Record<string, unknown>>('/api/config')
    if (inviteConfig.registrationInviteRequired !== true) {
      throw new Error(`public invite config mismatch: ${JSON.stringify(inviteConfig)}`)
    }

    const blockedInviteResponse = await fetch(`${apiBase}/api/auth/register`, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        name: `invite_blocked_${runId.slice(0, 8)}`,
        email: `invite_blocked_${runId}@example.test`,
        password: 'password123'
      })
    })
    if (blockedInviteResponse.status !== 403) {
      throw new Error(`expected invite 403, got ${blockedInviteResponse.status}`)
    }

    const invitedResponse = await fetch(`${apiBase}/api/auth/register`, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        name: `invite_ok_${runId.slice(0, 8)}`,
        email: `invite_ok_${runId}@example.test`,
        password: 'password123',
        inviteCode
      })
    })
    if (invitedResponse.status !== 201) {
      throw new Error(`expected invite register 201, got ${invitedResponse.status}`)
    }
  }

  const updated = await apiFetch<Record<string, unknown>>('/api/admin/settings', {
    token: admin.token,
    method: 'PATCH',
    body: {
      registrationEnabled: false,
      guestProblemsetVisible: false,
      sourceOpenDefault: true
    }
  })

  if (updated.registrationEnabled !== false) throw new Error('registration setting did not update')
  if (updated.guestProblemsetVisible !== false)
    throw new Error('guest problemset setting did not update')

  const config = await apiFetch<Record<string, unknown>>('/api/config')
  if (config.registration !== false || config.guestProblemsetVisible !== false) {
    throw new Error(`public config mismatch: ${JSON.stringify(config)}`)
  }

  const registerResponse = await fetch(`${apiBase}/api/auth/register`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({
      name: `blocked_${runId.slice(0, 8)}`,
      email: `blocked_${runId}@example.test`,
      password: 'password123'
    })
  })
  if (registerResponse.status !== 403) {
    throw new Error(`expected registration 403, got ${registerResponse.status}`)
  }

  const problemsetResponse = await fetch(`${apiBase}/api/problems`)
  if (problemsetResponse.status !== 401) {
    throw new Error(`expected anonymous problemset 401, got ${problemsetResponse.status}`)
  }

  console.log({
    registration: config.registration,
    registrationInviteRequired: config.registrationInviteRequired,
    guestProblemsetVisible: config.guestProblemsetVisible,
    registerStatus: registerResponse.status,
    problemsetStatus: problemsetResponse.status
  })
} finally {
  await apiFetch('/api/admin/settings', {
    token: admin.token,
    method: 'PATCH',
    body: {
      ...original,
      ...(original.registrationInviteRequired ? {} : { registrationInviteCode: '' })
    }
  })
}

async function loginAdmin() {
  const response = await fetch(`${apiBase}/api/auth/login`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ user: adminUser, password: adminPassword })
  })

  if (!response.ok)
    throw new Error(`admin auth API failed: ${response.status} ${await response.text()}`)
  return (await response.json()) as { token: string }
}

async function apiFetch<T>(
  path: string,
  options: { token?: string; method?: string; body?: unknown } = {}
) {
  const response = await fetch(`${apiBase}${path}`, {
    method: options.method ?? 'GET',
    headers: {
      ...(options.body === undefined ? {} : { 'content-type': 'application/json' }),
      ...(options.token ? { authorization: `Bearer ${options.token}` } : {})
    },
    body: options.body === undefined ? undefined : JSON.stringify(options.body)
  })

  if (!response.ok) throw new Error(`${path} failed: ${response.status} ${await response.text()}`)
  return (await response.json()) as T
}
