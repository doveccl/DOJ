import { eq, sql } from 'drizzle-orm'
import { db, schema, sqlClient } from '@doj/db/client'
import { claimJudgeTask, completeJudgeTask, failJudgeTask, judgeTaskChannel } from '@doj/db/queue'
import { getRuntimeSettings } from '@doj/db/settings'
import type { JudgeAgentPayload, JudgeAgentProgress, JudgeAgentResult } from '@doj/shared/agent'
import { getLanguage } from './languages'
import { JudgeAgentServer } from './agent-server'

const workerId = process.env.DOJ_WORKER_ID || `worker-${crypto.randomUUID()}`
const wakePollMs = Number(process.env.DOJ_WORKER_POLL_MS ?? 1000)
const workerSlots = Number(process.env.DOJ_WORKER_SLOTS ?? 4)
const agentPort = Number(process.env.DOJ_WORKER_AGENT_PORT ?? 7975)
const agentJobTimeoutMs = Number(process.env.DOJ_AGENT_JOB_TIMEOUT_MS ?? 120_000)
const judgeTaskLeaseSeconds = Math.ceil(agentJobTimeoutMs / 1000) + 60

let wakeResolver: () => void = () => {}
let wakePromise = createWakePromise()

const agentServer = new JudgeAgentServer({
  port: agentPort,
  jobTimeoutMs: agentJobTimeoutMs,
  onWake: wakeWorkers
})
agentServer.start()
console.log(`DOJ worker listening for judge agents on ws://localhost:${agentPort}/agents/connect`)

async function handleOne() {
  const agent = agentServer.reserveAvailableAgent()
  if (!agent) return false

  const task = await claimJudgeTask({ workerId, leaseSeconds: judgeTaskLeaseSeconds })
  if (!task) {
    agentServer.releaseAgent(agent)
    return false
  }
  let submissionId = task.submissionId
  let sentToAgent = false

  try {
    const payload = await preparePayload(task.submissionId)
    submissionId = payload.submissionId

    await db
      .update(schema.submissions)
      .set({ status: 'JUDGING', updatedAt: new Date() })
      .where(eq(schema.submissions.id, payload.submissionId))
    await db
      .delete(schema.submissionCases)
      .where(eq(schema.submissionCases.submissionId, payload.submissionId))

    sentToAgent = true
    const judged = await agentServer.runJob(agent, payload, {
      onProgress: (progress) => persistJudgeProgress(payload.submissionId, progress)
    })
    await persistJudgeResult(payload.submissionId, payload.problemId, judged)
    await completeJudgeTask(task.id)
  } catch (error) {
    await db
      .update(schema.submissions)
      .set({
        status: 'SE',
        message: error instanceof Error ? error.message : String(error),
        judgeProgress: null,
        updatedAt: new Date()
      })
      .where(eq(schema.submissions.id, submissionId))
    await failJudgeTask(task.id, error)
    if (!sentToAgent) agentServer.releaseAgent(agent)
  }

  return true
}

async function preparePayload(
  submissionId: number
): Promise<JudgeAgentPayload & { problemId: number }> {
  const [submission] = await db
    .select()
    .from(schema.submissions)
    .where(eq(schema.submissions.id, submissionId))
    .limit(1)

  if (!submission) throw new Error(`submission not found: ${submissionId}`)

  const [problem] = await db
    .select()
    .from(schema.problems)
    .where(eq(schema.problems.id, submission.problemId))
    .limit(1)

  if (!problem) throw new Error(`problem not found: ${submission.problemId}`)

  const language = await getLanguage(submission.languageId)
  const packageFiles = await getProblemPackageFiles(problem.id)
  const settings = await getRuntimeSettings()

  return {
    submissionId: submission.id,
    problemId: submission.problemId,
    scopeId: `submission-${submission.id}`,
    code: submission.sourceCode,
    limits: {
      timeMs: problem.timeLimitMs,
      memoryBytes: problem.memoryLimitBytes,
      outputBytes: settings.outputLimitBytes
    },
    // Submission (B) build context.
    testerFiles: {
      Dockerfile: language.dockerfile,
      [language.sourceFile]: submission.sourceCode
    },
    problemFiles: packageFiles,
    inlineTestCases: problem.testCases,
    caseCount: problem.caseCount
  }
}

async function getProblemPackageFiles(problemId: number) {
  const rows = await db
    .select({
      path: schema.problemFiles.path,
      bucket: schema.files.bucket,
      objectKey: schema.files.objectKey
    })
    .from(schema.problemFiles)
    .innerJoin(schema.files, eq(schema.problemFiles.fileId, schema.files.id))
    .where(eq(schema.problemFiles.problemId, problemId))
  return rows.map((row) => ({ path: row.path, bucket: row.bucket, objectKey: row.objectKey }))
}

async function persistJudgeResult(
  submissionId: number,
  problemId: number,
  judged: JudgeAgentResult
) {
  const [submission] = await db
    .update(schema.submissions)
    .set({
      status: judged.status,
      timeMs: judged.timeMs,
      memoryBytes: judged.memoryBytes,
      score: judged.score,
      message: judged.message,
      judgeProgress: null,
      updatedAt: new Date()
    })
    .where(eq(schema.submissions.id, submissionId))
    .returning({
      userId: schema.submissions.userId
    })

  if (!submission) throw new Error(`submission not found while saving result: ${submissionId}`)

  if (judged.cases.length) {
    await db
      .delete(schema.submissionCases)
      .where(eq(schema.submissionCases.submissionId, submissionId))
    await db.insert(schema.submissionCases).values(
      judged.cases.map((item) => ({
        submissionId,
        caseIndex: item.caseIndex,
        status: item.status,
        timeMs: item.timeMs,
        memoryBytes: item.memoryBytes,
        score: item.score,
        message: item.message ?? ''
      }))
    )
  }

  if (judged.status === 'AC') {
    await recordSolved(submission.userId, problemId, submissionId)
  }
}

async function persistJudgeProgress(submissionId: number, progress: JudgeAgentProgress) {
  await db
    .update(schema.submissions)
    .set({
      judgeProgress: progress,
      message: progress.message,
      updatedAt: new Date()
    })
    .where(eq(schema.submissions.id, submissionId))

  if (progress.case) {
    await db
      .insert(schema.submissionCases)
      .values({
        submissionId,
        caseIndex: progress.case.caseIndex,
        status: progress.case.status,
        timeMs: progress.case.timeMs,
        memoryBytes: progress.case.memoryBytes,
        score: progress.case.score,
        message: progress.case.message
      })
      .onConflictDoUpdate({
        target: [schema.submissionCases.submissionId, schema.submissionCases.caseIndex],
        set: {
          status: progress.case.status,
          timeMs: progress.case.timeMs,
          memoryBytes: progress.case.memoryBytes,
          score: progress.case.score,
          message: progress.case.message
        }
      })
  }
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

async function loop(slot: number) {
  console.log(`judge worker ${workerId} slot ${slot} started`)
  for (;;) {
    const claimed = await handleOne()
    if (!claimed) await waitForWake()
  }
}

function createWakePromise() {
  return new Promise<void>((resolve) => {
    wakeResolver = resolve
  })
}

function wakeWorkers() {
  wakeResolver()
  wakePromise = createWakePromise()
}

async function waitForWake() {
  await Promise.race([wakePromise, Bun.sleep(wakePollMs)])
}

await sqlClient.listen(
  judgeTaskChannel,
  () => wakeWorkers(),
  () => {
    console.log(`judge worker ${workerId} listening on ${judgeTaskChannel}`)
    wakeWorkers()
  }
)

if (process.env.DOJ_WORKER_ONCE === '1') {
  await handleOne()
  process.exit(0)
}

await Promise.all(Array.from({ length: workerSlots }, (_, index) => loop(index)))
