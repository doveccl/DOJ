import { eq, sql } from 'drizzle-orm'
import { db, schema, sqlClient } from '@doj/db/client'
import { claimJudgeTask, completeJudgeTask, failJudgeTask, judgeTaskChannel } from '@doj/db/queue'
import { getRuntimeSettings } from '@doj/db/settings'
import type { JudgeAgentPayload, JudgeAgentResult } from '@doj/shared/agent'
import { getObjectBytes } from '@doj/shared/storage'
import { getLanguage } from './languages'
import { JudgeAgentServer } from './agent-server'

const workerId = process.env.DOJ_WORKER_ID || `worker-${crypto.randomUUID()}`
const wakePollMs = Number(process.env.DOJ_WORKER_POLL_MS ?? 1000)
const workerSlots = Number(process.env.DOJ_WORKER_SLOTS ?? 4)
const agentPort = Number(process.env.DOJ_WORKER_AGENT_PORT ?? 7975)
const agentJobTimeoutMs = Number(process.env.DOJ_AGENT_JOB_TIMEOUT_MS ?? 120_000)

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
  const agent = agentServer.pickAvailableAgent()
  if (!agent) return false

  const task = await claimJudgeTask({ workerId, leaseSeconds: 120 })
  if (!task) return false
  let submissionId = task.submissionId

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

    const judged = await agentServer.runJob(agent, payload)
    await persistJudgeResult(payload.submissionId, payload.problemId, judged)
    await completeJudgeTask(task.id)
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

async function preparePayload(
  submissionId: number
): Promise<JudgeAgentPayload & { problemId: number }> {
  const [submission] = await db
    .select()
    .from(schema.submissions)
    .where(eq(schema.submissions.id, submissionId))
    .limit(1)

  if (!submission) throw new Error(`submission not found: ${submissionId}`)

  const [version] = await db
    .select()
    .from(schema.problemVersions)
    .where(eq(schema.problemVersions.id, submission.problemVersionId))
    .limit(1)

  if (!version) throw new Error(`problem version not found: ${submission.problemVersionId}`)

  const language = await getLanguage(submission.languageId)
  const file = version.testdataFileId ? await getTestdataFileRef(version.testdataFileId) : null
  const checker = version.checkerFileId ? await getCheckerSource(version.checkerFileId) : null
  const settings = await getRuntimeSettings()

  return {
    submissionId: submission.id,
    problemId: submission.problemId,
    scopeId: `submission-${submission.id}`,
    sourceCode: submission.sourceCode,
    language: {
      id: language.id,
      sourceFile: language.sourceFile,
      dockerfile: language.dockerfile,
      command: language.command
    },
    limits: {
      timeMs: version.timeLimitMs,
      memoryBytes: version.memoryLimitBytes,
      outputBytes: settings.outputLimitBytes
    },
    testCases: file ? [] : version.testCases,
    testdataFile: file,
    checker
  }
}

async function getCheckerSource(fileId: number) {
  const [file] = await db.select().from(schema.files).where(eq(schema.files.id, fileId)).limit(1)
  if (!file) throw new Error(`checker file not found: ${fileId}`)
  const bytes = await getObjectBytes(file.objectKey, file.bucket)
  return { sourceCode: Buffer.from(bytes).toString('utf8') }
}

async function getTestdataFileRef(fileId: number) {
  const [file] = await db.select().from(schema.files).where(eq(schema.files.id, fileId)).limit(1)
  if (!file) throw new Error(`testdata file not found: ${fileId}`)
  return {
    bucket: file.bucket,
    objectKey: file.objectKey,
    filename: file.filename,
    sizeBytes: file.sizeBytes
  }
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
      updatedAt: new Date()
    })
    .where(eq(schema.submissions.id, submissionId))
    .returning({
      userId: schema.submissions.userId
    })

  if (!submission) throw new Error(`submission not found while saving result: ${submissionId}`)

  if (judged.cases.length) {
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
