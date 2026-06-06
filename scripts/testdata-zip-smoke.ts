import { closeDb } from '../packages/db/src/client'
import { ensureJudgeServices, stopSpawnedJudgeServices, waitForJudgement } from './judge-services'

const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const adminUser = process.env.DOJ_ADMIN_NAME ?? 'admin'
const adminPassword = process.env.DOJ_ADMIN_PASSWORD ?? 'admin12345'
const runId = crypto.randomUUID()

try {
  await ensureJudgeServices()
  const [admin, user] = await Promise.all([loginAdmin(), registerUser()])
  const { problem } = await createProblem(admin.token)

  // Upload loose data files (mixed naming) under the `data/` prefix so the agent
  // derives default-mode cases from the problem package.
  const uploaded = await uploadDataFiles(admin.token, problem.id, {
    'input1.txt': '7\n',
    'ans01.txt': '21\n',
    '2.in': '-4\n',
    'output2.txt': '-12\n'
  })
  if (uploaded.length !== 4) throw new Error(`expected 4 package files, got ${uploaded.length}`)

  const listed = await listPackage(admin.token, problem.id)
  if (listed.length !== 4 || !listed.every((file) => file.path.startsWith('data/'))) {
    throw new Error(`package listing is wrong: ${JSON.stringify(listed)}`)
  }

  const accepted = await submitAndJudge(
    user.token,
    problem.id,
    '#include <bits/stdc++.h>\nusing namespace std;\nint main(){ long long x; cin >> x; cout << x * 3 << "\\n"; }\n'
  )
  if (accepted.status !== 'AC')
    throw new Error(`expected AC, got ${accepted.status}: ${accepted.message}`)

  const wrong = await submitAndJudge(
    user.token,
    problem.id,
    '#include <bits/stdc++.h>\nusing namespace std;\nint main(){ cout << 0 << "\\n"; }\n'
  )
  if (wrong.status !== 'WA') throw new Error(`expected WA, got ${wrong.status}: ${wrong.message}`)

  console.log({
    problemId: problem.id,
    packageFiles: listed.length,
    accepted: accepted.status,
    wrong: wrong.status
  })
} finally {
  await stopSpawnedJudgeServices()
  await closeDb()
}

async function loginAdmin() {
  const response = await fetch(`${apiBase}/api/auth/login`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ user: adminUser, password: adminPassword })
  })
  if (!response.ok)
    throw new Error(`admin auth API failed: ${response.status} ${await response.text()}`)
  return (await response.json()) as { token: string }
}

async function registerUser() {
  const response = await fetch(`${apiBase}/api/auth/register`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({
      name: `data_${runId.slice(0, 8)}`,
      email: `data_${runId}@example.test`,
      password: 'password123'
    })
  })
  if (!response.ok) throw new Error(`auth API failed: ${response.status} ${await response.text()}`)
  return (await response.json()) as { token: string }
}

async function createProblem(token: string) {
  const response = await fetch(`${apiBase}/api/problems`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      authorization: `Bearer ${token}`
    },
    body: JSON.stringify({
      title: `Package Data ${runId.slice(0, 8)}`,
      statementMarkdown: '# Package Data\n\nTriple the input integer.',
      timeLimitMs: 5000,
      memoryLimitBytes: 128 * 1024 * 1024
    })
  })
  if (!response.ok)
    throw new Error(`problem API failed: ${response.status} ${await response.text()}`)
  return (await response.json()) as { problem: { id: number } }
}

async function uploadDataFiles(token: string, problemId: number, files: Record<string, string>) {
  const form = new FormData()
  form.set('prefix', 'data/')
  for (const [name, content] of Object.entries(files)) {
    form.append('file', new File([content], name, { type: 'text/plain' }))
  }
  const response = await fetch(`${apiBase}/api/problems/${problemId}/package/upload`, {
    method: 'POST',
    headers: { authorization: `Bearer ${token}` },
    body: form
  })
  if (!response.ok)
    throw new Error(`package upload failed: ${response.status} ${await response.text()}`)
  const payload = (await response.json()) as { files: Array<{ path: string }> }
  return payload.files
}

async function listPackage(token: string, problemId: number) {
  const response = await fetch(`${apiBase}/api/problems/${problemId}/package`, {
    headers: { authorization: `Bearer ${token}` }
  })
  if (!response.ok)
    throw new Error(`package list failed: ${response.status} ${await response.text()}`)
  const payload = (await response.json()) as { files: Array<{ path: string }> }
  return payload.files
}

async function submitAndJudge(token: string, problemId: number, sourceCode: string) {
  const response = await fetch(`${apiBase}/api/submissions`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      authorization: `Bearer ${token}`
    },
    body: JSON.stringify({
      problemId,
      languageId: 'cc',
      sourceCode
    })
  })
  if (!response.ok)
    throw new Error(`submission API failed: ${response.status} ${await response.text()}`)
  const submission = (await response.json()) as { id: number }
  return waitForJudgement(submission.id)
}
