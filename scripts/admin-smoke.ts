const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const user = process.env.DOJ_ADMIN_NAME ?? 'admin'
const password = process.env.DOJ_ADMIN_PASSWORD ?? 'admin12345'
const groupKey = `group_${crypto.randomUUID().slice(0, 8)}`

const loginResponse = await fetch(`${apiBase}/api/auth/login`, {
  method: 'POST',
  headers: {
    'content-type': 'application/json'
  },
  body: JSON.stringify({ user, password })
})

if (!loginResponse.ok) {
  throw new Error(`admin login failed: ${loginResponse.status} ${await loginResponse.text()}`)
}

const login = (await loginResponse.json()) as { token: string }

const createResponse = await fetch(`${apiBase}/api/groups`, {
  method: 'POST',
  headers: {
    'content-type': 'application/json',
    authorization: `Bearer ${login.token}`
  },
  body: JSON.stringify({
    key: groupKey,
    name: `Smoke ${groupKey}`,
    description: 'Created by admin smoke test.'
  })
})

if (!createResponse.ok) {
  throw new Error(`group create failed: ${createResponse.status} ${await createResponse.text()}`)
}

const listResponse = await fetch(`${apiBase}/api/groups`, {
  headers: {
    authorization: `Bearer ${login.token}`
  }
})

if (!listResponse.ok) {
  throw new Error(`group list failed: ${listResponse.status} ${await listResponse.text()}`)
}

const list = (await listResponse.json()) as { list: Array<{ key: string }> }
if (!list.list.some((group) => group.key === groupKey)) {
  throw new Error(`created group missing from list: ${groupKey}`)
}

console.log({ groupKey, total: list.list.length })
