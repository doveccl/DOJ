import { eq } from 'drizzle-orm'
import { closeDb, db, schema } from '../packages/db/src/client'
import { ensureJudgeServices, stopSpawnedJudgeServices, waitForJudgement } from './judge-services'

const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const adminUser = process.env.DOJ_ADMIN_NAME ?? 'admin'
const adminPassword = process.env.DOJ_ADMIN_PASSWORD ?? 'admin12345'
const runId = crypto.randomUUID()
const hiddenExpected = `SECRET_EXPECTED_${runId}`

try {
  await ensureJudgeServices()
  const authResponse = await fetch(`${apiBase}/api/auth/register`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json'
    },
    body: JSON.stringify({
      name: `coach_${runId.slice(0, 8)}`,
      email: `coach_${runId}@example.test`,
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
      title: `Coach Problem ${runId.slice(0, 8)}`,
      statementMarkdown: '# Coach Problem\n\nFail intentionally.',
      timeLimitMs: 5000,
      memoryLimitBytes: 128 * 1024 * 1024,
      testCases: [
        {
          name: 'hidden-secret',
          input: '',
          output: `${hiddenExpected}\n`,
          hidden: true
        }
      ]
    })
  })

  if (!problemResponse.ok) {
    throw new Error(`problem API failed: ${problemResponse.status} ${await problemResponse.text()}`)
  }

  const { problem, version } = (await problemResponse.json()) as {
    problem: { id: number }
    version: { id: number }
  }

  const submissionResponse = await fetch(`${apiBase}/api/submissions`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      authorization: `Bearer ${auth.token}`
    },
    body: JSON.stringify({
      problemId: problem.id,
      problemVersionId: version.id,
      languageId: 'cpp',
      sourceCode:
        '#include <bits/stdc++.h>\nusing namespace std;\nint main(){ cout << "wrong\\n"; }\n'
    })
  })

  if (!submissionResponse.ok) {
    throw new Error(
      `submission API failed: ${submissionResponse.status} ${await submissionResponse.text()}`
    )
  }

  const submission = (await submissionResponse.json()) as { id: number }
  const judged = await waitForJudgement(submission.id)
  if (judged.status !== 'WA') {
    throw new Error(`expected WA, got ${judged.status}: ${judged.message}`)
  }
  assertNoSecret('submission message', judged.message)
  assertNoHiddenCaseName('submission message', judged.message)

  const [caseResult] = await db
    .select()
    .from(schema.submissionCases)
    .where(eq(schema.submissionCases.submissionId, submission.id))
    .limit(1)
  if (!caseResult) throw new Error('expected a submission case result')
  assertNoSecret('case message', caseResult.message)
  assertNoHiddenCaseName('case message', caseResult.message)

  const anonymousCoach = await fetch(`${apiBase}/api/submissions/${submission.id}/coach`, {
    method: 'POST'
  })
  if (anonymousCoach.status !== 401) {
    throw new Error(
      `expected anonymous coach 401, got ${anonymousCoach.status}: ${await anonymousCoach.text()}`
    )
  }

  const coachResponse = await fetch(`${apiBase}/api/submissions/${submission.id}/coach`, {
    method: 'POST',
    headers: {
      authorization: `Bearer ${auth.token}`
    }
  })

  if (!coachResponse.ok) {
    throw new Error(`coach API failed: ${coachResponse.status} ${await coachResponse.text()}`)
  }

  const session = (await coachResponse.json()) as {
    id: number
    model: string
    responseMarkdown: string
  }
  assertNoSecret('coach response', session.responseMarkdown)

  console.log({
    sessionId: session.id,
    model: session.model,
    anonymousStatus: anonymousCoach.status,
    submissionMessage: judged.message,
    caseMessage: caseResult.message,
    preview: session.responseMarkdown.split('\n').slice(0, 3).join(' ')
  })
} finally {
  await stopSpawnedJudgeServices()
  await closeDb()
}

function assertNoSecret(label: string, value: string) {
  if (value.includes(hiddenExpected)) {
    throw new Error(`${label} leaked hidden expected output: ${value}`)
  }
}

function assertNoHiddenCaseName(label: string, value: string) {
  if (value.includes('hidden-secret')) {
    throw new Error(`${label} leaked hidden case name: ${value}`)
  }
}
