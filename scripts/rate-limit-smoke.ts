const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const runId = crypto.randomUUID()
const loginIp = `198.51.100.${Math.floor(Math.random() * 200) + 1}`

let lastStatus = 0
for (let attempt = 0; attempt < 41; attempt += 1) {
  const response = await fetch(`${apiBase}/api/auth/login`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      'x-forwarded-for': loginIp
    },
    body: JSON.stringify({
      user: `missing_${runId}`,
      password: 'wrong-password'
    })
  })
  lastStatus = response.status
}

if (lastStatus !== 429) {
  throw new Error(`expected login rate limit 429, got ${lastStatus}`)
}

const otherIpStatus = await fetch(`${apiBase}/api/auth/login`, {
  method: 'POST',
  headers: {
    'content-type': 'application/json',
    'x-forwarded-for': '203.0.113.42'
  },
  body: JSON.stringify({
    user: `missing_${runId}`,
    password: 'wrong-password'
  })
})

if (otherIpStatus.status !== 401) {
  throw new Error(`expected other IP to avoid rate limit, got ${otherIpStatus.status}`)
}

console.log({
  loginIp,
  limitedStatus: lastStatus,
  otherIpStatus: otherIpStatus.status
})
