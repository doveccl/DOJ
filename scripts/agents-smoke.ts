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

const agent = (await api('/api/admin/agents', {
  method: 'POST',
  headers,
  body: JSON.stringify({
    key: 'remote-smoke',
    name: 'Remote Smoke',
    enabled: false,
    labels: ['remote', 'linux'],
    token: 'smoke-agent-token-1234',
    concurrency: 1,
    sortOrder: 90
  })
})) as {
  key: string
  labels: string[]
  enabled: boolean
  token?: string
  tokenHash?: string
  id: number
}

if (
  agent.key !== 'remote-smoke' ||
  agent.enabled ||
  agent.token !== 'smoke-agent-token-1234' ||
  agent.tokenHash !== undefined ||
  agent.labels.join(',') !== 'remote,linux'
) {
  throw new Error(`agent upsert mismatch: ${JSON.stringify(agent)}`)
}

const list = (await api('/api/admin/agents', {
  headers
})) as { list: Array<{ id: number; key: string; tokenHash?: string }> }

const local = list.list.find((item) => item.key === 'local-agent')
if (!local) {
  throw new Error(`local-agent missing: ${JSON.stringify(list.list)}`)
}
if (list.list.some((item) => item.tokenHash !== undefined)) {
  throw new Error(`agent list leaked token hash: ${JSON.stringify(list.list)}`)
}

const rotated = (await api(`/api/admin/agents/${agent.id}/rotate-token`, {
  method: 'POST',
  headers
})) as { key: string; token: string }

if (rotated.key !== 'remote-smoke' || rotated.token.length < 24) {
  throw new Error(`agent token rotation failed: ${JSON.stringify(rotated)}`)
}

console.log({
  agentKey: agent.key,
  total: list.list.length,
  rotated: rotated.key
})
