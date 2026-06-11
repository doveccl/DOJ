import { eq } from 'drizzle-orm'
import { claimJudgeTask, completeJudgeTask, failJudgeTask, renewJudgeTaskLease } from '@doj/db/queue'
import { db, schema } from '@doj/db/client'
import type { JudgeStatus } from '@doj/shared/status'
import type { JudgeProgress, JudgeResult } from '@doj/shared/judge'
import { broadcastSubmissionResult } from './browserWs'
import { clearJudgeProgress, saveJudgeProgress } from './progress'
import {
  AgentJobRetryableError,
  dispatchJudgeToAgent,
  getProblemBundleInfo,
  hasAvailableAgent
} from './routes/agents'
import { recordSubmissionFinal } from './services/stats'

const workerId = `server-${crypto.randomUUID()}`
const pollMs = Number(process.env.JUDGE_POLL_MS ?? 1000)
const leaseSeconds = 60
const renewMs = 20_000
let started = false

export function startJudgeScheduler() {
  if (started) return
  started = true
  void loop()
}

async function loop() {
  console.log(`DOJ judge scheduler started as ${workerId}`)
  for (;;) {
    const claimed = await handleOne().catch((error) => {
      console.error('judge scheduler error:', error)
      return false
    })
    if (!claimed) await Bun.sleep(pollMs)
  }
}

async function handleOne() {
  if (!hasAvailableAgent()) return false
  const task = await claimJudgeTask({ workerId, leaseSeconds })
  if (!task) return false

  const renew = setInterval(() => {
    void renewJudgeTaskLease(task.id, leaseSeconds).catch((error) => {
      console.error(`failed to renew judge task ${task.id}:`, error)
    })
  }, renewMs)
  try {
    await runSubmission(task.submissionId)
    await completeJudgeTask(task.id)
  } catch (error) {
    if (error instanceof AgentJobRetryableError) {
      return true
    }
    await markSystemError(task.submissionId, error)
    await failJudgeTask(task.id, error)
  } finally {
    clearInterval(renew)
  }

  return true
}

async function runSubmission(submissionId: number) {
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

  const [language] = await db
    .select()
    .from(schema.languages)
    .where(eq(schema.languages.id, submission.languageId))
    .limit(1)
  if (!language) throw new Error(`language not found: ${submission.languageId}`)

  await db
    .delete(schema.submissionCases)
    .where(eq(schema.submissionCases.submissionId, submissionId))

  const bundle = await getProblemBundleInfo(problem.id)
  const cases = buildAgentCases(bundle.entries)
  if (!cases.length) throw new Error(`problem ${problem.id} has no data cases`)
  await saveJudgeProgress(submissionId, {
    phase: 'queued',
    message: 'Judge task dispatched',
    totalCases: cases.length,
    completedCases: 0
  })

  const agentResult = await dispatchJudgeToAgent(
    {
      submissionId,
      problemId: problem.id,
      bundleHash: bundle.bundleHash,
      judger:
        problem.mode === 'custom'
          ? { kind: 'custom', bundleHash: bundle.bundleHash }
          : {
              kind: 'prebuilt',
              image: 'doveccl/doj:judger',
              check: problem.mode === 'strict' ? 'pe' : 'trim'
            },
      cases,
      limits: {
        timeLimit: problem.timeLimit,
        memoryLimit: problem.memoryLimit
      },
      submission: {
        languageId: language.id,
        source: language.source,
        code: submission.code,
        dockerfile: language.dockerfile
      }
    },
    {
      onProgress: (progress) => saveJudgeProgress(submissionId, normalizeProgress(progress, cases.length))
    }
  )
  if (agentResult) {
    await persistJudgeResult(submissionId, agentResult)
    await saveJudgeProgress(submissionId, {
      phase: 'finished',
      status: agentResult.status,
      message: agentResult.message,
      totalCases: cases.length,
      completedCases: cases.length
    })
    await broadcastSubmissionResult(submissionId)
    await clearJudgeProgress(submissionId)
    return
  }

  throw new AgentJobRetryableError('no online judge agent accepted the task')
}

async function persistJudgeResult(submissionId: number, result: JudgeResult) {
  const cases = [...result.cases].sort((left, right) => left.caseNo - right.caseNo).map((item) => ({
    submissionId,
    caseNo: item.caseNo,
    status: item.status,
    timeMs: item.timeMs,
    memoryBytes: item.memoryBytes,
    score: item.status === 'AC' ? 0 : 0,
    message: truncateMessage(item.message)
  }))
  const scoredCases = scoreCases(cases)
  const finalStatus = scoredCases.length
    ? aggregateStatus(scoredCases.map((item) => item.status))
    : result.status

  if (scoredCases.length) await db.insert(schema.submissionCases).values(scoredCases)
  const [updated] = await db
    .update(schema.submissions)
    .set({
      status: finalStatus,
      timeMs: Math.max(0, ...scoredCases.map((item) => item.timeMs)),
      memoryBytes: Math.max(0, ...scoredCases.map((item) => item.memoryBytes)),
      score: scoredCases.reduce((sum, item) => sum + item.score, 0),
      message: truncateMessage(result.message),
      updatedAt: new Date()
    })
    .where(eq(schema.submissions.id, submissionId))
    .returning()
  if (updated) await recordSubmissionFinal(updated)
}

function scoreCases<T extends { status: JudgeStatus; score: number }>(cases: T[]) {
  if (!cases.length) return cases
  const base = Math.floor(100 / cases.length)
  let rest = 100 - base * cases.length
  return cases.map((item) => {
    const fullScore = base + (rest > 0 ? 1 : 0)
    rest -= 1
    return { ...item, score: item.status === 'AC' ? fullScore : 0 }
  })
}

function buildAgentCases(entries: Array<{ path: string }>) {
  const inputs = new Map<string, string>()
  const answers = new Map<string, string>()
  for (const entry of entries) {
    const inputKey = readCaseKey(entry.path, 'input')
    const answerKey = readCaseKey(entry.path, 'answer')
    if (inputKey) inputs.set(inputKey, entry.path)
    if (answerKey) answers.set(answerKey, entry.path)
  }
  return [...inputs.keys()].sort(compareCaseKey).flatMap((key, index) => {
    const inputPath = inputs.get(key)
    const answerPath = answers.get(key)
    return inputPath && answerPath ? [{ caseNo: index + 1, inputPath, answerPath }] : []
  })
}

function aggregateStatus(statuses: JudgeStatus[]): JudgeStatus {
  return statuses.find((status) => status !== 'AC') ?? 'AC'
}

async function markSystemError(submissionId: number, error: unknown) {
  const [updated] = await db
    .update(schema.submissions)
    .set({
      status: 'SE',
      timeMs: 0,
      memoryBytes: 0,
      score: 0,
      message: truncateMessage(error instanceof Error ? error.message : String(error)),
      updatedAt: new Date()
    })
    .where(eq(schema.submissions.id, submissionId))
    .returning()
  if (updated) await recordSubmissionFinal(updated)
  await saveJudgeProgress(submissionId, {
    phase: 'finished',
    status: 'SE',
    message: error instanceof Error ? error.message : String(error)
  })
  await broadcastSubmissionResult(submissionId)
}

function truncateMessage(message: string) {
  return message.length > 4096 ? `${message.slice(0, 4093)}...` : message
}

function normalizeProgress(progress: JudgeProgress, totalCases: number): JudgeProgress {
  return {
    ...progress,
    totalCases,
    completedCases:
      progress.status && progress.caseNo
        ? Math.min(totalCases, progress.caseNo)
        : progress.phase === 'finished'
          ? totalCases
          : undefined
  }
}

function readCaseKey(path: string, kind: 'input' | 'answer') {
  const name = path.replace(/^.*\//, '').toLowerCase()
  const stem = name.replace(/\.[^.]+$/, '')
  const extension = name.includes('.') ? name.replace(/^.*\./, '.') : ''
  const matches =
    kind === 'input'
      ? extension === '.in' || /input|(^|[^a-z])in([^a-z]|$)/.test(stem)
      : extension === '.out' ||
        extension === '.ans' ||
        /output|answer|ans|(^|[^a-z])out([^a-z]|$)/.test(stem)
  if (!matches) return null
  return stem.match(/\d+/g)?.at(-1) ?? stem
}

function compareCaseKey(left: string, right: string) {
  const leftNumber = Number.parseInt(left, 10)
  const rightNumber = Number.parseInt(right, 10)
  if (Number.isFinite(leftNumber) && Number.isFinite(rightNumber)) return leftNumber - rightNumber
  return left.localeCompare(right)
}
