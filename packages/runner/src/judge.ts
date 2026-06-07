import { createHash } from 'node:crypto'
import type { JudgeAgentProgress, JudgeAgentResult } from '@doj/shared/agent'
import type { JudgeLimit, ProblemTestCase } from '@doj/shared/judge'
import type { JudgeStatus } from '@doj/shared/status'
import type { Runner } from './types'

export interface PackageJudgeInput {
  scopeId: string
  // Submission (B) build context; must include a `Dockerfile`.
  testerFiles: Record<string, string | Uint8Array>
  // Problem (A) build context; must include a `Dockerfile`. When omitted, the
  // engine runs the default mode (engine-controlled stdin/stdout compare).
  problemFiles?: Record<string, string | Uint8Array> | null
  // Test cases (used in default mode for input/answer; in custom mode only the
  // count and optional per-case `points` weights matter — A reads its own data).
  testCases: ProblemTestCase[]
  // For custom packages whose A generates its own data, the number of cases to
  // run when there are no explicit test cases. Falls back to testCases.length.
  caseCount?: number
  limits: JudgeLimit
  // Submission source, exposed to A as the `code` env (e.g. Quine checkers).
  code?: string
  // Disabled by default because B contains untrusted user source. Agents may
  // opt in when their Docker image retention policy allows it.
  cacheSubmissionPackage?: boolean
  signal?: AbortSignal
  onProgress?: (progress: JudgeAgentProgress) => void | Promise<void>
}

// Package-based judging with the A (problem) / B (submission) container model.
// Mode is auto-detected by whether the problem package ships a Dockerfile:
//   - custom mode: build A and B, duel per case (A is interactor + checker).
//   - default mode: build B only, engine pipes input -> B -> compare answer.
export async function judgePackage(
  runner: Runner,
  input: PackageJudgeInput
): Promise<JudgeAgentResult> {
  const testerScope = `${input.scopeId}-b`
  const problemScope = `${input.scopeId}-a`
  const custom = !!input.problemFiles && 'Dockerfile' in input.problemFiles

  throwIfCancelled(input.signal)
  await input.onProgress?.({
    phase: 'building',
    message: 'Building submission package',
    completedCases: 0,
    totalCases: totalCases(input)
  })
  const testerBuild = await runner.buildPackage({
    scopeId: testerScope,
    files: input.testerFiles,
    limits: { timeMs: input.limits.timeMs, memoryBytes: input.limits.memoryBytes },
    trusted: false,
    cacheKey: input.cacheSubmissionPackage ? packageCacheKey(input.testerFiles) : undefined,
    signal: input.signal
  })
  throwIfCancelled(input.signal)

  try {
    if (!testerBuild.ok || !testerBuild.imageId) {
      return {
        status: 'CE',
        timeMs: 0,
        memoryBytes: 0,
        score: 0,
        maxScore: 100,
        message: testerBuild.logs,
        cases: []
      }
    }

    if (!custom) {
      return await judgeDefaultCases({
        runner,
        scopeId: testerScope,
        imageId: testerBuild.imageId,
        testCases: input.testCases,
        limits: input.limits,
        signal: input.signal,
        onProgress: input.onProgress
      })
    }

    await input.onProgress?.({
      phase: 'building',
      message: 'Building problem package',
      completedCases: 0,
      totalCases: totalCases(input)
    })
    const problemBuild = await runner.buildPackage({
      scopeId: problemScope,
      files: input.problemFiles as Record<string, string | Uint8Array>,
      trusted: true,
      cacheKey: packageCacheKey(input.problemFiles as Record<string, string | Uint8Array>),
      signal: input.signal
    })
    throwIfCancelled(input.signal)
    if (!problemBuild.ok || !problemBuild.imageId) {
      return {
        status: 'SE',
        timeMs: 0,
        memoryBytes: 0,
        score: 0,
        maxScore: 100,
        message: `problem package build failed:\n${problemBuild.logs}`,
        cases: []
      }
    }

    // Custom packages whose A generates its own data have no explicit test
    // cases; synthesize `caseCount` empty cases so the duel runs that many times.
    const customCases =
      input.testCases.length > 0
        ? input.testCases
        : Array.from({ length: Math.max(1, input.caseCount ?? 1) }, (_, index) => ({
            name: String(index + 1),
            input: '',
            output: '',
            hidden: true
          }))

    return await judgeCustomCases({
      runner,
      scopeId: input.scopeId,
      judgeImageId: problemBuild.imageId,
      testerImageId: testerBuild.imageId,
      testCases: customCases,
      limits: input.limits,
      code: input.code ?? '',
      signal: input.signal,
      onProgress: input.onProgress
    })
  } finally {
    await runner.cleanup({ scopeId: testerScope })
    if (custom) await runner.cleanup({ scopeId: problemScope })
  }
}

interface DefaultCasesInput {
  runner: Runner
  scopeId: string
  imageId: string
  testCases: ProblemTestCase[]
  limits: JudgeLimit
  signal?: AbortSignal
  onProgress?: (progress: JudgeAgentProgress) => void | Promise<void>
}

async function judgeDefaultCases(input: DefaultCasesInput): Promise<JudgeAgentResult> {
  const { runner, testCases, limits, imageId, scopeId } = input
  const weights = caseWeights(testCases)
  const maxScore = weights.reduce((total, weight) => total + weight, 0) || 100

  const cases = []
  let score = 0
  let timeMs = 0
  let memoryBytes = 0
  let firstFailure: { index: number; status: JudgeStatus; message: string; name?: string } | null =
    null

  for (const [index, testCase] of testCases.entries()) {
    throwIfCancelled(input.signal)
    await input.onProgress?.({
      phase: 'testing',
      message: `Testing case ${index + 1}/${testCases.length}`,
      completedCases: cases.length,
      totalCases: testCases.length,
      currentCase: index + 1
    })
    const result = await runner.run({
      scopeId,
      imageId,
      limits,
      stdin: new TextEncoder().encode(testCase.input),
      signal: input.signal
    })
    throwIfCancelled(input.signal)
    const compared =
      result.status === 'AC'
        ? compareOutput(result.stdout, testCase.output)
        : { status: result.status, message: result.stderr }
    const caseStatus = compared.status
    const caseScore = caseStatus === 'AC' ? weights[index] : 0
    const caseMessage = buildCaseMessage({
      status: caseStatus,
      hidden: testCase.hidden === true,
      message: compared.message
    })
    score += caseScore
    timeMs = Math.max(timeMs, result.timeMs)
    memoryBytes = Math.max(memoryBytes, result.memoryBytes)
    cases.push({
      caseIndex: index + 1,
      status: caseStatus,
      timeMs: result.timeMs,
      memoryBytes: result.memoryBytes,
      score: caseScore,
      message: caseMessage
    })
    await input.onProgress?.({
      phase: 'testing',
      message: `Finished case ${index + 1}/${testCases.length}`,
      completedCases: cases.length,
      totalCases: testCases.length,
      currentCase: index + 1,
      case: cases[cases.length - 1]
    })
    if (caseStatus !== 'AC' && !firstFailure) {
      firstFailure = {
        index,
        status: caseStatus,
        message: caseMessage,
        name: testCase.hidden ? undefined : testCase.name
      }
    }
  }

  return finalizeResult({ cases, score, maxScore, timeMs, memoryBytes, firstFailure })
}

interface CustomCasesInput {
  runner: Runner
  scopeId: string
  judgeImageId: string
  testerImageId: string
  testCases: ProblemTestCase[]
  limits: JudgeLimit
  code: string
  signal?: AbortSignal
  onProgress?: (progress: JudgeAgentProgress) => void | Promise<void>
}

async function judgeCustomCases(input: CustomCasesInput): Promise<JudgeAgentResult> {
  const { runner, testCases, limits } = input
  const weights = caseWeights(testCases)
  const maxScore = weights.reduce((total, weight) => total + weight, 0) || 100

  const cases = []
  let score = 0
  let timeMs = 0
  let memoryBytes = 0
  let firstFailure: { index: number; status: JudgeStatus; message: string; name?: string } | null =
    null

  for (const [index, testCase] of testCases.entries()) {
    throwIfCancelled(input.signal)
    await input.onProgress?.({
      phase: 'testing',
      message: `Testing case ${index + 1}/${testCases.length}`,
      completedCases: cases.length,
      totalCases: testCases.length,
      currentCase: index + 1
    })
    const result = await runner.duel({
      scopeId: `${input.scopeId}-case-${index}`,
      judgeImageId: input.judgeImageId,
      testerImageId: input.testerImageId,
      limits,
      judgeEnv: { case: String(index), code: input.code },
      signal: input.signal
    })
    throwIfCancelled(input.signal)
    const caseStatus = result.status
    // A may report partial score via its result file; otherwise full weight on AC.
    const caseScore =
      result.score !== null
        ? Math.max(0, Math.min(weights[index], Math.round(result.score)))
        : caseStatus === 'AC'
          ? weights[index]
          : 0
    const caseMessage = buildCaseMessage({
      status: caseStatus,
      hidden: testCase.hidden === true,
      message: result.message
    })
    score += caseScore
    timeMs = Math.max(timeMs, result.timeMs)
    memoryBytes = Math.max(memoryBytes, result.memoryBytes)
    cases.push({
      caseIndex: index + 1,
      status: caseStatus,
      timeMs: result.timeMs,
      memoryBytes: result.memoryBytes,
      score: caseScore,
      message: caseMessage
    })
    await input.onProgress?.({
      phase: 'testing',
      message: `Finished case ${index + 1}/${testCases.length}`,
      completedCases: cases.length,
      totalCases: testCases.length,
      currentCase: index + 1,
      case: cases[cases.length - 1]
    })
    if (caseStatus !== 'AC' && !firstFailure) {
      firstFailure = {
        index,
        status: caseStatus,
        message: caseMessage,
        name: testCase.hidden ? undefined : testCase.name
      }
    }
  }

  return finalizeResult({ cases, score, maxScore, timeMs, memoryBytes, firstFailure })
}

function totalCases(input: PackageJudgeInput) {
  if (input.problemFiles && 'Dockerfile' in input.problemFiles) {
    return input.testCases.length || Math.max(1, input.caseCount ?? 1)
  }
  return input.testCases.length
}

function throwIfCancelled(signal: AbortSignal | undefined) {
  if (signal?.aborted) throw new Error('judge job cancelled')
}

function packageCacheKey(files: Record<string, string | Uint8Array>) {
  const hash = createHash('sha256')
  for (const name of Object.keys(files).sort()) {
    const content = files[name]
    hash.update(name)
    hash.update('\0')
    hash.update(typeof content === 'string' ? Buffer.from(content) : Buffer.from(content))
    hash.update('\0')
  }
  return hash.digest('hex')
}

function finalizeResult(input: {
  cases: JudgeAgentResult['cases']
  score: number
  maxScore: number
  timeMs: number
  memoryBytes: number
  firstFailure: { index: number; status: JudgeStatus; message: string; name?: string } | null
}): JudgeAgentResult {
  const { firstFailure, score, maxScore } = input
  const status: JudgeStatus = firstFailure ? firstFailure.status : 'AC'
  const message = firstFailure
    ? `score ${score}/${maxScore} — case ${firstFailure.index + 1}` +
      `${firstFailure.name ? ` (${firstFailure.name})` : ''}: ${firstFailure.message}`
    : 'accepted'
  return {
    status,
    timeMs: input.timeMs,
    memoryBytes: input.memoryBytes,
    score,
    maxScore,
    message,
    cases: input.cases
  }
}

// Per-case weights: use explicit `points` when any case defines them; otherwise
// split 100 points evenly across cases, giving the remainder to the first cases
// so the weights still sum to exactly 100.
function caseWeights(testCases: ProblemTestCase[]): number[] {
  const hasPoints = testCases.some((testCase) => typeof testCase.points === 'number')
  if (hasPoints) {
    return testCases.map((testCase) => Math.max(0, Math.round(testCase.points ?? 0)))
  }

  const count = testCases.length
  const base = Math.floor(100 / count)
  const remainder = 100 - base * count
  return testCases.map((_, index) => base + (index < remainder ? 1 : 0))
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

function buildCaseMessage(input: { status: JudgeStatus; hidden: boolean; message: string }) {
  if (!input.hidden) return input.message
  if (input.status === 'AC') return 'accepted'
  if (input.status === 'PE') return 'presentation error on hidden test'
  if (input.status === 'WA') return 'wrong answer on hidden test'
  if (input.status === 'TLE') return 'time limit exceeded on hidden test'
  if (input.status === 'MLE') return 'memory limit exceeded on hidden test'
  if (input.status === 'OLE') return 'output limit exceeded on hidden test'
  if (input.status === 'RE') return 'runtime error on hidden test'
  return `${input.status.toLowerCase()} on hidden test`
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
