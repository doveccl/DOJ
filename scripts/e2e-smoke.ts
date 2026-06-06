import { closeDb } from '../packages/db/src/client'
import { ensureJudgeServices, stopSpawnedJudgeServices, waitForJudgement } from './judge-services'

const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const adminUser = process.env.DOJ_ADMIN_NAME ?? 'admin'
const adminPassword = process.env.DOJ_ADMIN_PASSWORD ?? 'admin12345'
const runId = crypto.randomUUID()

try {
  await ensureJudgeServices()

  const authResponse = await fetch(`${apiBase}/api/auth/register`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json'
    },
    body: JSON.stringify({
      name: `smoke_${runId.slice(0, 8)}`,
      email: `smoke_${runId}@example.test`,
      password: 'password123'
    })
  })

  if (!authResponse.ok) {
    throw new Error(`auth API failed: ${authResponse.status} ${await authResponse.text()}`)
  }

  const auth = (await authResponse.json()) as { token: string }
  const adminResponse = await fetch(`${apiBase}/api/auth/login`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json'
    },
    body: JSON.stringify({ user: adminUser, password: adminPassword })
  })

  if (!adminResponse.ok) {
    throw new Error(`admin auth API failed: ${adminResponse.status} ${await adminResponse.text()}`)
  }

  const admin = (await adminResponse.json()) as { token: string }

  const problemResponse = await fetch(`${apiBase}/api/problems`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      authorization: `Bearer ${admin.token}`
    },
    body: JSON.stringify({
      title: `Smoke Problem ${runId.slice(0, 8)}`,
      statementMarkdown: '# Smoke Problem\n\nExit successfully.',
      timeLimitMs: 5000,
      memoryLimitBytes: 128 * 1024 * 1024
    })
  })

  if (!problemResponse.ok) {
    throw new Error(`problem API failed: ${problemResponse.status} ${await problemResponse.text()}`)
  }

  const { problem, version } = (await problemResponse.json()) as {
    problem: { id: number }
    version: { id: number }
  }

  const response = await fetch(`${apiBase}/api/submissions`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      authorization: `Bearer ${auth.token}`
    },
    body: JSON.stringify({
      problemId: problem.id,
      problemVersionId: version.id,
      languageId: 'cc',
      sourceCode:
        '#include <bits/stdc++.h>\nusing namespace std;\nint main(){ cout << "accepted\\n"; return 0; }\n'
    })
  })

  if (!response.ok) {
    throw new Error(`submission API failed: ${response.status} ${await response.text()}`)
  }

  const submission = (await response.json()) as { id: number }
  const judged = await waitForJudgement(submission.id)

  if (judged.status !== 'AC') {
    throw new Error(`expected AC, got ${judged.status}: ${judged.message}`)
  }

  const dashboardResponse = await fetch(`${apiBase}/api/dashboard`, {
    headers: {
      authorization: `Bearer ${auth.token}`
    }
  })
  if (!dashboardResponse.ok) {
    throw new Error(
      `dashboard API failed: ${dashboardResponse.status} ${await dashboardResponse.text()}`
    )
  }
  const dashboard = (await dashboardResponse.json()) as {
    recentProblems: Array<{ id: number }>
  }
  if (dashboard.recentProblems.some((item) => item.id === problem.id)) {
    throw new Error(`solved problem leaked into dashboard recommendations: ${problem.id}`)
  }

  console.log({
    submissionId: judged.id,
    status: judged.status,
    message: judged.message.trim(),
    timeMs: judged.timeMs,
    memoryBytes: judged.memoryBytes
  })
} finally {
  await stopSpawnedJudgeServices()
  await closeDb()
}
