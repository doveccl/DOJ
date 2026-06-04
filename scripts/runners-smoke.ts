const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const adminUser = process.env.DOJ_ADMIN_NAME ?? 'admin'
const adminPassword = process.env.DOJ_ADMIN_PASSWORD ?? 'admin12345'

async function api(path: string, init: RequestInit = {}) {
  const response = await fetch(`${apiBase}${path}`, init)
  if (!response.ok) {
    throw new Error(`${path} failed: ${response.status} ${await response.text()}`)
  }
  return response.json()
}

const admin = (await api('/api/auth/login', {
  method: 'POST',
  headers: {
    'content-type': 'application/json'
  },
  body: JSON.stringify({ user: adminUser, password: adminPassword })
})) as { token: string }

const headers = {
  'content-type': 'application/json',
  authorization: `Bearer ${admin.token}`
}

const runner = (await api('/api/admin/runners', {
  method: 'POST',
  headers,
  body: JSON.stringify({
    key: 'remote-smoke',
    name: 'Remote Smoke',
    enabled: false,
    kind: 'docker',
    endpoint: 'https://docker.example.test',
    authHeader: 'Bearer smoke-token',
    concurrency: 1,
    sortOrder: 90
  })
})) as { key: string; endpoint: string | null; authHeader: string | null; enabled: boolean }

if (runner.key !== 'remote-smoke' || runner.enabled || runner.endpoint !== 'https://docker.example.test') {
  throw new Error(`runner upsert mismatch: ${JSON.stringify(runner)}`)
}

const list = (await api('/api/admin/runners', {
  headers
})) as { list: Array<{ key: string }> }

if (!list.list.some((item) => item.key === 'local-docker')) {
  throw new Error(`local-docker runner missing: ${JSON.stringify(list.list)}`)
}

console.log({
  runnerKey: runner.key,
  total: list.list.length
})
