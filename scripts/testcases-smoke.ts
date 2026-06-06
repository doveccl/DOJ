import { eq } from 'drizzle-orm'
import { closeDb, db, schema } from '../packages/db/src/client'
import { ensureJudgeServices, stopSpawnedJudgeServices, waitForJudgement } from './judge-services'

const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const adminUser = process.env.DOJ_ADMIN_NAME ?? 'admin'
const adminPassword = process.env.DOJ_ADMIN_PASSWORD ?? 'admin12345'
const runId = crypto.randomUUID()

try {
  await ensureJudgeServices()
  const auth = await registerUser()
  const admin = await loginAdmin()
  const { problem, version } = await createProblem(admin.token)

  const accepted = await submitAndJudge(
    auth.token,
    problem.id,
    version.id,
    '#include <bits/stdc++.h>\nusing namespace std;\nint main(){ long long x; cin >> x; cout << x * 2 << "\\n"; }\n'
  )
  if (accepted.status !== 'AC') {
    throw new Error(`expected AC, got ${accepted.status}: ${accepted.message}`)
  }

  const wrong = await submitAndJudge(
    auth.token,
    problem.id,
    version.id,
    '#include <bits/stdc++.h>\nusing namespace std;\nint main(){ cout << 0 << "\\n"; }\n'
  )
  if (wrong.status !== 'WA') {
    throw new Error(`expected WA, got ${wrong.status}: ${wrong.message}`)
  }

  const cases = await db
    .select()
    .from(schema.submissionCases)
    .where(eq(schema.submissionCases.submissionId, wrong.id))
    .orderBy(schema.submissionCases.caseIndex)

  if (!cases.length || cases[0]?.status !== 'WA') {
    throw new Error(`expected WA case result, got ${JSON.stringify(cases)}`)
  }

  console.log({
    problemId: problem.id,
    accepted: accepted.status,
    wrong: wrong.status,
    wrongCase: cases[0]
  })
} finally {
  await stopSpawnedJudgeServices()
  await closeDb()
}

async function registerUser() {
  const response = await fetch(`${apiBase}/api/auth/register`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json'
    },
    body: JSON.stringify({
      name: `case_${runId.slice(0, 8)}`,
      email: `case_${runId}@example.test`,
      password: 'password123'
    })
  })

  if (!response.ok) throw new Error(`auth API failed: ${response.status} ${await response.text()}`)
  return (await response.json()) as { token: string }
}

async function loginAdmin() {
  const response = await fetch(`${apiBase}/api/auth/login`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json'
    },
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
      title: `Case Problem ${runId.slice(0, 8)}`,
      statementMarkdown: '# Case Problem\n\nDouble the input integer.',
      timeLimitMs: 5000,
      memoryLimitBytes: 128 * 1024 * 1024,
      testCases: [
        {
          name: 'sample',
          input: '21\n',
          output: '42\n'
        },
        {
          name: 'hidden',
          input: '-3\n',
          output: '-6\n',
          hidden: true
        }
      ]
    })
  })

  if (!response.ok)
    throw new Error(`problem API failed: ${response.status} ${await response.text()}`)
  return (await response.json()) as { problem: { id: number }; version: { id: number } }
}

async function submitAndJudge(
  token: string,
  problemId: number,
  problemVersionId: number,
  sourceCode: string
) {
  const response = await fetch(`${apiBase}/api/submissions`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      authorization: `Bearer ${token}`
    },
    body: JSON.stringify({
      problemId,
      problemVersionId,
      languageId: 'cc',
      sourceCode
    })
  })

  if (!response.ok)
    throw new Error(`submission API failed: ${response.status} ${await response.text()}`)
  const submission = (await response.json()) as { id: number }
  return waitForJudgement(submission.id)
}
