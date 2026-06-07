import { closeDb } from '../packages/db/src/client'
import { ensureJudgeServices, stopSpawnedJudgeServices, waitForJudgement } from './judge-services'

const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const adminUser = process.env.DOJ_ADMIN_NAME ?? 'admin'
const adminPassword = process.env.DOJ_ADMIN_PASSWORD ?? 'admin12345'
const runId = crypto.randomUUID()

try {
  await ensureJudgeServices()
  const user = await registerUser()
  const admin = await loginAdmin()
  const { problem } = await createProblem(admin.token)

  const submission = await submit(user.token, problem.id)
  const progress = await waitForProgress(user.token, submission.id)
  if (progress.totalCases !== 3 || progress.completedCases < 1) {
    throw new Error(`unexpected progress payload: ${JSON.stringify(progress)}`)
  }

  const judged = await waitForJudgement(submission.id)
  if (judged.status !== 'AC' || judged.judgeProgress !== null) {
    throw new Error(`expected AC with cleared progress, got ${JSON.stringify(judged)}`)
  }

  const detail = await api<{
    cases: Array<{ caseIndex: number; status: string }>
    judgeProgress: unknown
  }>(`/api/submissions/${submission.id}`, user.token)
  if (detail.cases.length !== 3 || detail.cases.some((item) => item.status !== 'AC')) {
    throw new Error(`expected 3 AC case rows, got ${JSON.stringify(detail.cases)}`)
  }
  if (detail.judgeProgress !== null) {
    throw new Error(`final detail still has progress: ${JSON.stringify(detail.judgeProgress)}`)
  }

  console.log({
    submissionId: submission.id,
    progress,
    status: judged.status,
    cases: detail.cases.length
  })
} finally {
  await stopSpawnedJudgeServices()
  await closeDb()
}

async function registerUser() {
  const response = await fetch(`${apiBase}/api/auth/register`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({
      name: `progress_${runId.slice(0, 8)}`,
      email: `progress_${runId}@example.test`,
      password: 'password123'
    })
  })
  if (!response.ok) throw new Error(`auth API failed: ${response.status} ${await response.text()}`)
  return (await response.json()) as { token: string }
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

async function createProblem(token: string) {
  const response = await fetch(`${apiBase}/api/problems`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      authorization: `Bearer ${token}`
    },
    body: JSON.stringify({
      title: `Progress Problem ${runId.slice(0, 8)}`,
      statementMarkdown: '# Progress Problem\n\nEcho three separate cases slowly.',
      timeLimitMs: 1000,
      memoryLimitBytes: 128 * 1024 * 1024,
      testCases: [
        { name: 'one', input: '1\n', output: '1\n' },
        { name: 'two', input: '2\n', output: '2\n' },
        { name: 'three', input: '3\n', output: '3\n' }
      ]
    })
  })
  if (!response.ok)
    throw new Error(`problem API failed: ${response.status} ${await response.text()}`)
  return (await response.json()) as { problem: { id: number } }
}

async function submit(token: string, problemId: number) {
  const response = await fetch(`${apiBase}/api/submissions`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      authorization: `Bearer ${token}`
    },
    body: JSON.stringify({
      problemId,
      languageId: 'cc',
      sourceCode:
        '#include <bits/stdc++.h>\nusing namespace std;\nint main(){ this_thread::sleep_for(chrono::milliseconds(600)); int x; cin >> x; cout << x << "\\n"; }\n'
    })
  })
  if (!response.ok)
    throw new Error(`submission API failed: ${response.status} ${await response.text()}`)
  return (await response.json()) as { id: number }
}

async function waitForProgress(token: string, submissionId: number) {
  for (let attempt = 0; attempt < 80; attempt += 1) {
    const detail = await api<{
      judgeProgress: {
        completedCases: number
        totalCases: number
        message: string
      } | null
      status: string
      message: string
    }>(`/api/submissions/${submissionId}`, token)
    if (detail.judgeProgress && detail.judgeProgress.completedCases >= 1) {
      return detail.judgeProgress
    }
    if (!['WAITING', 'JUDGING'].includes(detail.status)) {
      throw new Error(`submission finished before progress was observed: ${JSON.stringify(detail)}`)
    }
    await Bun.sleep(250)
  }
  throw new Error('judge progress was not observed')
}

async function api<T>(path: string, token: string): Promise<T> {
  const response = await fetch(`${apiBase}${path}`, {
    headers: { authorization: `Bearer ${token}` }
  })
  if (!response.ok) throw new Error(`${path} failed: ${response.status} ${await response.text()}`)
  return (await response.json()) as T
}
