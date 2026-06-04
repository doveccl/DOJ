import { asc, eq, sql } from 'drizzle-orm'
import { db, schema } from '@doj/db/client'
import { claimJudgeTask, completeJudgeTask, failJudgeTask } from '@doj/db/queue'
import { DockerRunner } from '@doj/runner/docker-runner'
import type { JudgeStatus } from '@doj/shared/status'
import { getLanguage } from './languages'

const workerId = `worker-${crypto.randomUUID()}`
const concurrency = Number(process.env.DOJ_JUDGE_CONCURRENCY ?? 2)

async function handleOne() {
  const task = await claimJudgeTask({ workerId, leaseSeconds: 120 })
  if (!task) return false
  let submissionId = task.submissionId

  try {
    const [submission] = await db
      .select()
      .from(schema.submissions)
      .where(eq(schema.submissions.id, task.submissionId))
      .limit(1)

    if (!submission) throw new Error(`submission not found: ${task.submissionId}`)
    submissionId = submission.id

    const [version] = await db
      .select()
      .from(schema.problemVersions)
      .where(eq(schema.problemVersions.id, submission.problemVersionId))
      .limit(1)

    if (!version) throw new Error(`problem version not found: ${submission.problemVersionId}`)

    const language = await getLanguage(submission.languageId)
    const runner = await getRunner()
    const scopeId = `submission-${submission.id}`

    await db
      .update(schema.submissions)
      .set({ status: 'JUDGING', updatedAt: new Date() })
      .where(eq(schema.submissions.id, submission.id))

    const build = await runner.build({
      scopeId,
      dockerfile: language.dockerfile,
      files: {
        [language.sourceFile]: submission.sourceCode
      },
      limits: {
        timeMs: version.timeLimitMs,
        memoryBytes: version.memoryLimitBytes
      }
    })

    if (!build.ok || !build.imageId) {
      await db
        .update(schema.submissions)
        .set({
          status: 'CE',
          message: build.logs,
          updatedAt: new Date()
        })
        .where(eq(schema.submissions.id, submission.id))
      await completeJudgeTask(task.id)
      await runner.cleanup({ scopeId })
      return true
    }

    const judged = await judgeCases({
      runner,
      scopeId,
      imageId: build.imageId,
      command: language.command.length ? language.command : undefined,
      testCases: version.testCases,
      limits: {
        timeMs: version.timeLimitMs,
        memoryBytes: version.memoryLimitBytes,
        outputBytes: version.outputLimitBytes
      }
    })

    await db
      .update(schema.submissions)
      .set({
        status: judged.status,
        timeMs: judged.timeMs,
        memoryBytes: judged.memoryBytes,
        message: judged.message,
        updatedAt: new Date()
      })
      .where(eq(schema.submissions.id, submission.id))

    if (judged.cases.length) {
      await db.insert(schema.submissionCases).values(
        judged.cases.map((item) => ({
          submissionId: submission.id,
          caseIndex: item.caseIndex,
          status: item.status,
          timeMs: item.timeMs,
          memoryBytes: item.memoryBytes,
          message: item.message ?? ''
        }))
      )
    }

    if (judged.status === 'AC') {
      await recordSolved(submission.userId, submission.problemId, submission.id)
    }

    await completeJudgeTask(task.id)
    await runner.cleanup({ scopeId })
  } catch (error) {
    await db
      .update(schema.submissions)
      .set({
        status: 'SE',
        message: error instanceof Error ? error.message : String(error),
        updatedAt: new Date()
      })
      .where(eq(schema.submissions.id, submissionId))
    await failJudgeTask(task.id, error)
  }

  return true
}

async function recordSolved(userId: number, problemId: number, submissionId: number) {
  const inserted = await db
    .insert(schema.solvedProblems)
    .values({ userId, problemId, firstSubmissionId: submissionId })
    .onConflictDoNothing()
    .returning()

  if (!inserted.length) return

  await Promise.all([
    db
      .update(schema.users)
      .set({ solvedCount: sql`${schema.users.solvedCount} + 1`, updatedAt: new Date() })
      .where(eq(schema.users.id, userId)),
    db
      .update(schema.problems)
      .set({ solvedCount: sql`${schema.problems.solvedCount} + 1`, updatedAt: new Date() })
      .where(eq(schema.problems.id, problemId))
  ])
}

type RunInput = Parameters<DockerRunner['run']>[0]
type JudgeCasesInput = RunInput & {
  runner: DockerRunner
  testCases: Array<{ name?: string; input: string; output: string }>
}

async function judgeCases(input: JudgeCasesInput) {
  const { runner, testCases, ...runInput } = input

  if (!input.testCases.length) {
    const result = await runner.run(runInput)
    return {
      status: result.status,
      timeMs: result.timeMs,
      memoryBytes: result.memoryBytes,
      message: result.stderr || Buffer.from(result.stdout).toString('utf8'),
      cases: []
    }
  }

  const cases = []
  let status: JudgeStatus = 'AC'
  let timeMs = 0
  let memoryBytes = 0
  let message = 'accepted'

  for (const [index, testCase] of testCases.entries()) {
    const result = await runner.run({
      ...runInput,
      stdin: new TextEncoder().encode(testCase.input)
    })

    const compared = compareOutput(result.stdout, testCase.output)
    const caseStatus = result.status === 'AC' ? compared.status : result.status
    const caseMessage = result.status === 'AC' ? compared.message : result.stderr
    timeMs = Math.max(timeMs, result.timeMs)
    memoryBytes = Math.max(memoryBytes, result.memoryBytes)
    cases.push({
      caseIndex: index + 1,
      status: caseStatus,
      timeMs: result.timeMs,
      memoryBytes: result.memoryBytes,
      message: caseMessage
    })

    if (caseStatus !== 'AC') {
      status = caseStatus
      message = `case ${index + 1}${testCase.name ? ` (${testCase.name})` : ''}: ${caseMessage}`
      break
    }
  }

  return {
    status,
    timeMs,
    memoryBytes,
    message,
    cases
  }
}

function compareOutput(stdout: Uint8Array, expected: string) {
  const actual = new TextDecoder().decode(stdout)
  if (normalizeEnd(actual) === normalizeEnd(expected)) {
    return { status: 'AC' as const, message: 'accepted' }
  }

  if (normalizeWhitespace(actual) === normalizeWhitespace(expected)) {
    return { status: 'PE' as const, message: 'presentation error' }
  }

  return {
    status: 'WA' as const,
    message: `wrong answer: expected ${preview(expected)}, got ${preview(actual)}`
  }
}

function normalizeEnd(value: string) {
  return value.replace(/\r\n/g, '\n').trimEnd()
}

function normalizeWhitespace(value: string) {
  return value.trim().split(/\s+/).join(' ')
}

function preview(value: string) {
  const normalized = normalizeEnd(value)
  return JSON.stringify(normalized.length > 120 ? `${normalized.slice(0, 120)}...` : normalized)
}

async function getRunner() {
  const [runner] = await db
    .select()
    .from(schema.judgeRunners)
    .where(eq(schema.judgeRunners.enabled, true))
    .orderBy(asc(schema.judgeRunners.sortOrder), asc(schema.judgeRunners.id))
    .limit(1)

  if (!runner) throw new Error('no enabled judge runner')
  if (runner.kind !== 'docker') throw new Error(`unsupported judge runner kind: ${runner.kind}`)

  return new DockerRunner({
    endpoint: runner.endpoint,
    authHeader: runner.authHeader
  })
}

async function loop(slot: number) {
  console.log(`judge worker ${workerId} slot ${slot} started`)
  for (;;) {
    const claimed = await handleOne()
    if (!claimed) await Bun.sleep(1000)
  }
}

if (process.env.DOJ_WORKER_ONCE === '1') {
  await handleOne()
  process.exit(0)
}

await Promise.all(Array.from({ length: concurrency }, (_, i) => loop(i)))
