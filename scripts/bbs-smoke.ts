const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const runId = crypto.randomUUID()

const user = await api<{ token: string }>('/api/auth/register', {
  method: 'POST',
  body: JSON.stringify({
    name: `bbs_${runId.slice(0, 8)}`,
    email: `bbs_${runId}@example.test`,
    password: 'password123'
  })
})

const topic = await api<{ topic: { id: number }; replies: Array<{ id: number }> }>(
  '/api/bbs/topics',
  {
    method: 'POST',
    headers: authHeaders(user.token),
    body: JSON.stringify({
      title: `BBS Smoke ${runId.slice(0, 8)}`,
      contentMarkdown: 'First post',
      tags: ['smoke']
    })
  }
)

await api(`/api/bbs/topics/${topic.topic.id}/replies`, {
  method: 'POST',
  headers: authHeaders(user.token),
  body: JSON.stringify({
    contentMarkdown: 'Second post'
  })
})

const detail = await api<{ replies: Array<{ contentMarkdown: string }> }>(
  `/api/bbs/topics/${topic.topic.id}`
)

if (detail.replies.length !== 2 || detail.replies[1]?.contentMarkdown !== 'Second post') {
  throw new Error(`BBS detail mismatch: ${JSON.stringify(detail)}`)
}

const list = await api<{ list: Array<{ id: number }> }>('/api/bbs/topics')
if (!list.list.some((item) => item.id === topic.topic.id)) {
  throw new Error(`BBS topic missing from list: ${topic.topic.id}`)
}

console.log({
  topicId: topic.topic.id,
  replies: detail.replies.length
})

async function api<T>(path: string, init: RequestInit = {}) {
  const headers = new Headers(init.headers)
  if (!headers.has('content-type') && init.body) headers.set('content-type', 'application/json')
  const response = await fetch(`${apiBase}${path}`, { ...init, headers })
  if (!response.ok) throw new Error(`${path} failed: ${response.status} ${await response.text()}`)
  return (await response.json()) as T
}

function authHeaders(token: string) {
  return {
    'content-type': 'application/json',
    authorization: `Bearer ${token}`
  }
}
