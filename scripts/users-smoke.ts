const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const adminUser = process.env.DOJ_ADMIN_NAME ?? 'admin'
const adminPassword = process.env.DOJ_ADMIN_PASSWORD ?? 'admin12345'
const runId = crypto.randomUUID()
const userName = `userctl_${runId.slice(0, 8)}`
const email = `userctl_${runId}@example.test`
const password = 'password123'

const registered = await api<{ user: { id: number }; token: string }>('/api/auth/register', {
  method: 'POST',
  body: JSON.stringify({ name: userName, email, password })
})

const admin = await api<{ token: string }>('/api/auth/login', {
  method: 'POST',
  body: JSON.stringify({ user: adminUser, password: adminPassword })
})
const headers = authHeaders(admin.token)

const disabled = await api<{ id: number; disabledAt: string | null }>(
  `/api/users/${registered.user.id}`,
  {
    method: 'PATCH',
    headers,
    body: JSON.stringify({ disabled: true })
  }
)
if (!disabled.disabledAt) throw new Error(`expected disabled user: ${JSON.stringify(disabled)}`)

const disabledLogin = await fetch(`${apiBase}/api/auth/login`, {
  method: 'POST',
  headers: { 'content-type': 'application/json' },
  body: JSON.stringify({ user: userName, password })
})
if (disabledLogin.status !== 403) {
  throw new Error(
    `expected disabled login 403, got ${disabledLogin.status}: ${await disabledLogin.text()}`
  )
}

const enabled = await api<{ id: number; disabledAt: string | null }>(
  `/api/users/${registered.user.id}`,
  {
    method: 'PATCH',
    headers,
    body: JSON.stringify({ disabled: false })
  }
)
if (enabled.disabledAt) throw new Error(`expected enabled user: ${JSON.stringify(enabled)}`)

await api('/api/auth/login', {
  method: 'POST',
  body: JSON.stringify({ user: email, password })
})

console.log({
  userId: registered.user.id,
  disabledLoginStatus: disabledLogin.status,
  reenabled: enabled.disabledAt === null
})

async function api<T>(path: string, init: RequestInit = {}) {
  const requestHeaders = new Headers(init.headers)
  if (!requestHeaders.has('content-type') && init.body) {
    requestHeaders.set('content-type', 'application/json')
  }
  const response = await fetch(`${apiBase}${path}`, { ...init, headers: requestHeaders })
  if (!response.ok) throw new Error(`${path} failed: ${response.status} ${await response.text()}`)
  return (await response.json()) as T
}

function authHeaders(token: string) {
  return {
    'content-type': 'application/json',
    authorization: `Bearer ${token}`
  }
}
