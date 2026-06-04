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

const before = (await api('/api/languages')) as { list: Array<{ id: string }> }
if (!before.list.some((language) => language.id === 'py')) {
  throw new Error(`py should be enabled before smoke: ${JSON.stringify(before.list)}`)
}

await api('/api/admin/languages', {
  method: 'POST',
  headers: adminHeaders,
  body: JSON.stringify({
    id: 'py',
    name: 'Python',
    enabled: false,
    sourceFile: 'main.py',
    dockerfile: 'FROM python:latest\nWORKDIR /workspace\nCOPY main.py /workspace/main.py\nCMD ["python3", "/workspace/main.py"]',
    command: [],
    sortOrder: 20
  })
})

const disabled = (await api('/api/languages')) as { list: Array<{ id: string }> }
if (disabled.list.some((language) => language.id === 'py')) {
  throw new Error(`py should be hidden while disabled: ${JSON.stringify(disabled.list)}`)
}

const auth = (await api('/api/auth/register', {
  method: 'POST',
  headers: {
    'content-type': 'application/json'
  },
  body: JSON.stringify({
    name: `lang_${runId.slice(0, 8)}`,
    email: `lang_${runId}@example.test`,
    password: 'password123'
  })
})) as { token: string }

const { problem, version } = (await api('/api/problems', {
  method: 'POST',
  headers: {
    'content-type': 'application/json'
  },
  body: JSON.stringify({
    title: `Language Smoke ${runId.slice(0, 8)}`,
    slug: `language-smoke-${runId}`,
    statementMarkdown: '# Language Smoke\n\nLanguage must be enabled.'
  })
})) as { problem: { id: number }; version: { id: number } }

const submissionResponse = await fetch(`${apiBase}/api/submissions`, {
  method: 'POST',
  headers: {
    'content-type': 'application/json',
    authorization: `Bearer ${auth.token}`
  },
  body: JSON.stringify({
    problemId: problem.id,
    problemVersionId: version.id,
    languageId: 'py',
    sourceCode: 'print("accepted")\n'
  })
})

if (submissionResponse.ok) {
  throw new Error(`disabled language submission unexpectedly succeeded: ${await submissionResponse.text()}`)
}

await api('/api/admin/languages', {
  method: 'POST',
  headers: adminHeaders,
  body: JSON.stringify({
    id: 'py',
    name: 'Python',
    enabled: true,
    sourceFile: 'main.py',
    dockerfile: 'FROM python:latest\nWORKDIR /workspace\nCOPY main.py /workspace/main.py\nCMD ["python3", "/workspace/main.py"]',
    command: [],
    sortOrder: 20
  })
})

const after = (await api('/api/languages')) as { list: Array<{ id: string }> }
if (!after.list.some((language) => language.id === 'py')) {
  throw new Error(`py should be enabled after smoke: ${JSON.stringify(after.list)}`)
}

console.log({
  disabledStatus: submissionResponse.status,
  languages: after.list.map((language) => language.id)
})
