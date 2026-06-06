const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const adminUser = process.env.DOJ_ADMIN_NAME ?? 'admin'
const adminPassword = process.env.DOJ_ADMIN_PASSWORD ?? 'admin12345'
const runId = crypto.randomUUID()

const admin = await api<{ token: string }>('/api/auth/login', {
  method: 'POST',
  body: JSON.stringify({ user: adminUser, password: adminPassword })
})
const headers = authHeaders(admin.token)

const created = await api<{
  problem: { id: number; title: string; visible: boolean }
}>('/api/problems', {
  method: 'POST',
  headers,
  body: JSON.stringify({
    title: `Editable Problem ${runId.slice(0, 8)}`,
    tags: ['smoke'],
    statementMarkdown: '# Editable\n\nVersion one.',
    timeLimitMs: 1000,
    memoryLimitBytes: 64 * 1024 * 1024,
    testCases: [{ name: 'sample', input: '1\n', output: '1\n', hidden: false }]
  })
})

const hidden = await api<{
  problem: { visible: boolean; title: string; statementMarkdown: string }
  testCases: Array<{ hidden: boolean }>
}>(`/api/problems/${created.problem.id}`, {
  method: 'PATCH',
  headers,
  body: JSON.stringify({
    title: `Edited Problem ${runId.slice(0, 8)}`,
    visible: false,
    statementMarkdown: '# Editable\n\nVersion two.',
    testCases: [{ name: 'hidden', input: '2\n', output: '2\n', hidden: true }]
  })
})

if (hidden.problem.visible || !hidden.problem.statementMarkdown.includes('Version two')) {
  throw new Error(`problem update failed: ${JSON.stringify(hidden)}`)
}

const publicHiddenResponse = await fetch(`${apiBase}/api/problems/${created.problem.id}`)
if (publicHiddenResponse.status !== 404) {
  throw new Error(
    `hidden problem should 404 publicly, got ${publicHiddenResponse.status}: ${await publicHiddenResponse.text()}`
  )
}

const adminDetail = await api<{
  problem: { visible: boolean }
  testCases: Array<{ hidden: boolean }>
}>(`/api/admin/problems/${created.problem.id}`, { headers })
if (adminDetail.problem.visible || adminDetail.testCases[0]?.hidden !== true) {
  throw new Error(`admin detail did not include hidden test cases: ${JSON.stringify(adminDetail)}`)
}

const visible = await api<{ problem: { visible: boolean } }>(`/api/problems/${created.problem.id}`, {
  method: 'PATCH',
  headers,
  body: JSON.stringify({ visible: true })
})
if (!visible.problem.visible) {
  throw new Error(`visibility update failed: ${JSON.stringify(visible)}`)
}

const publicDetail = await api<{
  problem: { id: number; visible: boolean }
  testCases: Array<{ hidden: boolean }>
}>(`/api/problems/${created.problem.id}`)
if (!publicDetail.problem.visible) {
  throw new Error(`visible problem missing from public detail: ${JSON.stringify(publicDetail)}`)
}
if (publicDetail.testCases.length !== 0) {
  throw new Error(`hidden test cases leaked publicly: ${JSON.stringify(publicDetail)}`)
}

console.log({
  problemId: created.problem.id,
  hiddenVisible: hidden.problem.visible,
  publicHiddenStatus: publicHiddenResponse.status,
  visible: visible.problem.visible
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
