const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const runId = crypto.randomUUID()
const name = `auth_${runId.slice(0, 12).replaceAll('-', '_')}`
const email = `${name}@example.test`
const password = 'password123'

const configResponse = await fetch(`${apiBase}/api/config`)
if (!configResponse.ok) {
  throw new Error(`config failed: ${configResponse.status} ${await configResponse.text()}`)
}
const config = (await configResponse.json()) as { registration: boolean }
if (!config.registration) {
  throw new Error('registration should be enabled for auth smoke')
}

const registerResponse = await fetch(`${apiBase}/api/auth/register`, {
  method: 'POST',
  headers: {
    'content-type': 'application/json'
  },
  body: JSON.stringify({ name, email, password })
})

if (!registerResponse.ok) {
  throw new Error(`register failed: ${registerResponse.status} ${await registerResponse.text()}`)
}

const registered = (await registerResponse.json()) as {
  token: string
  user: { id: number; name: string; email: string; groups: string[] }
}

if (!registered.user.groups.includes('user')) {
  throw new Error(`registered user is missing user group: ${registered.user.groups.join(', ')}`)
}

const selfResponse = await fetch(`${apiBase}/api/auth/self`, {
  headers: {
    authorization: `Bearer ${registered.token}`
  }
})

if (!selfResponse.ok) {
  throw new Error(`self failed: ${selfResponse.status} ${await selfResponse.text()}`)
}

const self = (await selfResponse.json()) as { id: number }
if (self.id !== registered.user.id) {
  throw new Error(`self returned wrong user: ${self.id}`)
}

const loginResponse = await fetch(`${apiBase}/api/auth/login`, {
  method: 'POST',
  headers: {
    'content-type': 'application/json'
  },
  body: JSON.stringify({ user: name, password })
})

if (!loginResponse.ok) {
  throw new Error(`login failed: ${loginResponse.status} ${await loginResponse.text()}`)
}

const login = (await loginResponse.json()) as { user: { id: number } }
if (login.user.id !== registered.user.id) {
  throw new Error(`login returned wrong user: ${login.user.id}`)
}

console.log({
  id: registered.user.id,
  name: registered.user.name,
  groups: registered.user.groups
})
