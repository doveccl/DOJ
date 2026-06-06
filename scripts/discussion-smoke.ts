const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const adminUser = process.env.DOJ_ADMIN_NAME ?? 'admin'
const adminPassword = process.env.DOJ_ADMIN_PASSWORD ?? 'admin12345'
const runId = crypto.randomUUID()

const admin = await api<{ token: string }>('/api/auth/login', {
  method: 'POST',
  body: JSON.stringify({ user: adminUser, password: adminPassword })
})

const user = await api<{ token: string }>('/api/auth/register', {
  method: 'POST',
  body: JSON.stringify({
    name: `disc_${runId.slice(0, 8)}`,
    email: `disc_${runId}@example.test`,
    password: 'password123'
  })
})

const topic = await api<{ topic: { id: number }; replies: Array<{ id: number }> }>(
  '/api/discussion/topics',
  {
    method: 'POST',
    headers: authHeaders(user.token),
    body: JSON.stringify({
      title: `Discussion Smoke ${runId.slice(0, 8)}`,
      contentMarkdown: 'First post',
      tags: ['smoke']
    })
  }
)

const hiddenProblem = await api<{ problem: { id: number } }>('/api/problems', {
  method: 'POST',
  headers: authHeaders(admin.token),
  body: JSON.stringify({
    title: `Discussion Hidden ${runId.slice(0, 8)}`,
    statementMarkdown: '# Hidden\n\nThis should not be linked.',
    testCases: []
  })
})
const legacyLinkedTopic = await api<{ topic: { id: number; linkedProblemId: number | null } }>(
  '/api/discussion/topics',
  {
    method: 'POST',
    headers: authHeaders(user.token),
    body: JSON.stringify({
      title: `Legacy Hidden Link ${runId.slice(0, 8)}`,
      contentMarkdown: 'This link should be sanitized after the problem is hidden.',
      linkedProblemId: hiddenProblem.problem.id
    })
  }
)
if (legacyLinkedTopic.topic.linkedProblemId !== hiddenProblem.problem.id) {
  throw new Error(
    `expected visible linked problem before hiding: ${JSON.stringify(legacyLinkedTopic)}`
  )
}
await api(`/api/problems/${hiddenProblem.problem.id}`, {
  method: 'PATCH',
  headers: authHeaders(admin.token),
  body: JSON.stringify({ visible: false })
})
const hiddenLinkStatus = await apiStatus('/api/discussion/topics', {
  method: 'POST',
  headers: authHeaders(user.token),
  body: JSON.stringify({
    title: `Hidden Link ${runId.slice(0, 8)}`,
    contentMarkdown: 'Should be rejected',
    linkedProblemId: hiddenProblem.problem.id
  })
})
if (hiddenLinkStatus !== 404) {
  throw new Error(`expected hidden linked problem 404, got ${hiddenLinkStatus}`)
}

await api(`/api/discussion/topics/${topic.topic.id}/replies`, {
  method: 'POST',
  headers: authHeaders(user.token),
  body: JSON.stringify({
    contentMarkdown: 'Second post'
  })
})

const detail = await api<{ replies: Array<{ contentMarkdown: string }> }>(
  `/api/discussion/topics/${topic.topic.id}`
)

if (detail.replies.length !== 2 || detail.replies[1]?.contentMarkdown !== 'Second post') {
  throw new Error(`Discussion detail mismatch: ${JSON.stringify(detail)}`)
}

const list = await api<{ list: Array<{ id: number }> }>('/api/discussion/topics')
if (!list.list.some((item) => item.id === topic.topic.id)) {
  throw new Error(`Discussion topic missing from list: ${topic.topic.id}`)
}
const legacyDetail = await api<{ topic: { linkedProblemId: number | null } }>(
  `/api/discussion/topics/${legacyLinkedTopic.topic.id}`
)
if (legacyDetail.topic.linkedProblemId !== null) {
  throw new Error(`hidden linked problem leaked in detail: ${JSON.stringify(legacyDetail.topic)}`)
}
const legacyListRow = (
  await api<{ list: Array<{ id: number; linkedProblemId: number | null }> }>(
    '/api/discussion/topics'
  )
).list.find((item) => item.id === legacyLinkedTopic.topic.id)
if (!legacyListRow || legacyListRow.linkedProblemId !== null) {
  throw new Error(`hidden linked problem leaked in list: ${JSON.stringify(legacyListRow)}`)
}

console.log({
  topicId: topic.topic.id,
  replies: detail.replies.length,
  hiddenLinkStatus,
  hiddenLinkSanitized: true
})

async function api<T>(path: string, init: RequestInit = {}) {
  const headers = new Headers(init.headers)
  if (!headers.has('content-type') && init.body) headers.set('content-type', 'application/json')
  const response = await fetch(`${apiBase}${path}`, { ...init, headers })
  if (!response.ok) throw new Error(`${path} failed: ${response.status} ${await response.text()}`)
  return (await response.json()) as T
}

async function apiStatus(path: string, init: RequestInit = {}) {
  const headers = new Headers(init.headers)
  if (!headers.has('content-type') && init.body) headers.set('content-type', 'application/json')
  const response = await fetch(`${apiBase}${path}`, { ...init, headers })
  return response.status
}

function authHeaders(token: string) {
  return {
    'content-type': 'application/json',
    authorization: `Bearer ${token}`
  }
}
