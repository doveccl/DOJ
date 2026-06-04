import { eq } from 'drizzle-orm'
import { closeDb, db, schema } from '../packages/db/src/client'

const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const runId = crypto.randomUUID()

try {
  const auth = await registerUser()
  const { problem, version } = await createProblem()

  const accepted = await submitAndJudge(
    auth.token,
    problem.id,
    version.id,
    'print(int(input()) * 2)\n'
  )
  if (accepted.status !== 'AC') {
    throw new Error(`expected AC, got ${accepted.status}: ${accepted.message}`)
  }

  const wrong = await submitAndJudge(auth.token, problem.id, version.id, 'print(0)\n')
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

async function createProblem() {
  const response = await fetch(`${apiBase}/api/problems`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json'
    },
    body: JSON.stringify({
      title: `Case Problem ${runId.slice(0, 8)}`,
      slug: `case-${runId}`,
      statementMarkdown: '# Case Problem\n\nDouble the input integer.',
      timeLimitMs: 5000,
      memoryLimitBytes: 128 * 1024 * 1024,
      outputLimitBytes: 1024 * 1024,
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
      languageId: 'py',
      sourceCode
    })
  })

  if (!response.ok)
    throw new Error(`submission API failed: ${response.status} ${await response.text()}`)
  const submission = (await response.json()) as { id: number }
  return waitForJudgement(submission.id)
}

async function waitForJudgement(submissionId: number) {
  for (let attempt = 0; attempt < 20; attempt += 1) {
    await runWorkerOnce()

    const [judged] = await db
      .select()
      .from(schema.submissions)
      .where(eq(schema.submissions.id, submissionId))
      .limit(1)

    if (!judged) throw new Error(`submission disappeared: ${submissionId}`)
    if (!['WAITING', 'JUDGING'].includes(judged.status)) return judged
    await Bun.sleep(200)
  }

  throw new Error(`submission did not finish judging: ${submissionId}`)
}

async function runWorkerOnce() {
  const worker = Bun.spawn(['bun', 'run', '--cwd', 'apps/worker', 'dev'], {
    env: {
      ...process.env,
      DOJ_WORKER_ONCE: '1'
    },
    stdout: 'pipe',
    stderr: 'pipe'
  })

  const exitCode = await worker.exited
  if (exitCode !== 0) {
    throw new Error(
      [
        `worker failed with exit ${exitCode}`,
        await new Response(worker.stdout).text(),
        await new Response(worker.stderr).text()
      ].join('\n')
    )
  }
}
