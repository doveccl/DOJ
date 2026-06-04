const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const adminUser = process.env.DOJ_ADMIN_NAME ?? 'admin'
const adminPassword = process.env.DOJ_ADMIN_PASSWORD ?? 'admin12345'
const runId = crypto.randomUUID()

async function api(path: string, init: RequestInit = {}) {
  const response = await fetch(`${apiBase}${path}`, init)
  if (!response.ok) {
    throw new Error(`${path} failed: ${response.status} ${await response.text()}`)
  }
  return response.json()
}

const student = (await api('/api/auth/register', {
  method: 'POST',
  headers: {
    'content-type': 'application/json'
  },
  body: JSON.stringify({
    name: `member_${runId.slice(0, 8)}`,
    email: `member_${runId}@example.test`,
    password: 'password123'
  })
})) as { token: string; user: { id: number } }

const admin = (await api('/api/auth/login', {
  method: 'POST',
  headers: {
    'content-type': 'application/json'
  },
  body: JSON.stringify({ user: adminUser, password: adminPassword })
})) as { token: string }

const adminHeaders = {
  'content-type': 'application/json',
  authorization: `Bearer ${admin.token}`
}

const group = (await api('/api/groups', {
  method: 'POST',
  headers: adminHeaders,
  body: JSON.stringify({
    key: `member_${runId.slice(0, 8)}`,
    name: 'Membership Smoke Group'
  })
})) as { id: number; key: string }

await api(`/api/groups/${group.id}/users`, {
  method: 'POST',
  headers: adminHeaders,
  body: JSON.stringify({
    userId: student.user.id,
    manager: true
  })
})

const members = (await api(`/api/groups/${group.id}/users`, {
  headers: adminHeaders
})) as { list: Array<{ id: number; manager: boolean }> }

const member = members.list.find((item) => item.id === student.user.id)
if (!member?.manager) {
  throw new Error(`group member missing or not manager: ${JSON.stringify(members)}`)
}

const self = (await api('/api/auth/self', {
  headers: {
    authorization: `Bearer ${student.token}`
  }
})) as { groups: string[] }

if (!self.groups.includes(group.key)) {
  throw new Error(`auth self missing new group: ${JSON.stringify(self.groups)}`)
}

console.log({
  groupKey: group.key,
  userId: student.user.id,
  manager: member.manager
})
