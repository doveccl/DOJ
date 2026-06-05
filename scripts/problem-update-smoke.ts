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
  version: { id: number; version: number }
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
  problem: { visible: boolean; title: string }
  version: { version: number; statementMarkdown: string; testCases: unknown[] }
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

if (hidden.problem.visible || hidden.version.version !== 2) {
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
  version: { version: number; testCases: Array<{ hidden: boolean }> }
}>(`/api/admin/problems/${created.problem.id}`, { headers })
if (
  adminDetail.problem.visible ||
  adminDetail.version.version !== 2 ||
  adminDetail.version.testCases[0]?.hidden !== true
) {
  throw new Error(
    `admin detail did not include hidden latest version: ${JSON.stringify(adminDetail)}`
  )
}

const visible = await api<{ problem: { visible: boolean }; version: { version: number } }>(
  `/api/problems/${created.problem.id}`,
  {
    method: 'PATCH',
    headers,
    body: JSON.stringify({ visible: true })
  }
)
if (!visible.problem.visible || visible.version.version !== 2) {
  throw new Error(`visibility-only update should not create a version: ${JSON.stringify(visible)}`)
}

const publicDetail = await api<{
  problem: { id: number; visible: boolean }
  version: { version: number }
}>(`/api/problems/${created.problem.id}`)
if (!publicDetail.problem.visible || publicDetail.version.version !== 2) {
  throw new Error(`visible problem missing from public detail: ${JSON.stringify(publicDetail)}`)
}

console.log({
  problemId: created.problem.id,
  initialVersion: created.version.version,
  editedVersion: hidden.version.version,
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
