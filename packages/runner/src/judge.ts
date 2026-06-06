import type { JudgeAgentPayload, JudgeAgentResult } from '@doj/shared/agent'
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
  limits: JudgeLimit
  // Submission source, exposed to A as the `code` env (e.g. Quine checkers).
  code?: string
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

  const testerBuild = await runner.buildPackage({
    scopeId: testerScope,
    files: input.testerFiles,
    limits: { timeMs: input.limits.timeMs, memoryBytes: input.limits.memoryBytes },
    trusted: false
  })

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
        limits: input.limits
      })
    }

    const problemBuild = await runner.buildPackage({
      scopeId: problemScope,
      files: input.problemFiles as Record<string, string | Uint8Array>,
      trusted: true
    })
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

    return await judgeCustomCases({
      runner,
      scopeId: input.scopeId,
      judgeImageId: problemBuild.imageId,
      testerImageId: testerBuild.imageId,
      testCases: input.testCases,
      limits: input.limits,
      code: input.code ?? ''
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
    const result = await runner.run({
      scopeId,
      imageId,
      limits,
      stdin: new TextEncoder().encode(testCase.input)
    })
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
    const result = await runner.duel({
      scopeId: `${input.scopeId}-case-${index}`,
      judgeImageId: input.judgeImageId,
      testerImageId: input.testerImageId,
      limits,
      judgeEnv: { case: String(index), code: input.code }
    })
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

export async function judgePayload(
  runner: Runner,
  input: JudgeAgentPayload
): Promise<JudgeAgentResult> {
  const build = await runner.build({
    scopeId: input.scopeId,
    dockerfile: input.language.dockerfile,
    files: {
      [input.language.sourceFile]: input.sourceCode
    },
    limits: {
      timeMs: input.limits.timeMs,
      memoryBytes: input.limits.memoryBytes
    }
  })

  const checkerScopeId = `${input.scopeId}-checker`

  try {
    if (!build.ok || !build.imageId) {
      return {
        status: 'CE',
        timeMs: 0,
        memoryBytes: 0,
        score: 0,
        maxScore: 100,
        message: build.logs,
        cases: []
      }
    }

    let checkerImageId: string | undefined
    if (input.checker) {
      const checkerBuild = await runner.build({
        scopeId: checkerScopeId,
        dockerfile: checkerDockerfile,
        files: { 'checker.cc': input.checker.sourceCode },
        limits: {
          timeMs: input.limits.timeMs,
          memoryBytes: input.limits.memoryBytes
        }
      })
      if (!checkerBuild.ok || !checkerBuild.imageId) {
        return {
          status: 'SE',
          timeMs: 0,
          memoryBytes: 0,
          score: 0,
          maxScore: 100,
          message: `checker compile failed:\n${checkerBuild.logs}`,
          cases: []
        }
      }
      checkerImageId = checkerBuild.imageId
    }

    return await judgeCases({
      runner,
      scopeId: input.scopeId,
      imageId: build.imageId,
      command: input.language.command?.length ? input.language.command : undefined,
      testCases: input.testCases,
      limits: input.limits,
      checkerImageId
    })
  } finally {
    await runner.cleanup({ scopeId: input.scopeId })
    if (input.checker) await runner.cleanup({ scopeId: checkerScopeId })
  }
}

const checkerDockerfile = [
  'FROM gcc:latest',
  'WORKDIR /workspace',
  'COPY checker.cc /workspace/checker.cc',
  'RUN g++ -std=c++20 -O2 -pipe -o checker checker.cc',
  'CMD ["/workspace/checker"]'
].join('\n')

type RunInput = Parameters<Runner['run']>[0]
type JudgeCasesInput = RunInput & {
  runner: Runner
  testCases: ProblemTestCase[]
  checkerImageId?: string
}

async function judgeCases(input: JudgeCasesInput): Promise<JudgeAgentResult> {
  const { runner, testCases, checkerImageId, ...runInput } = input

  if (!input.testCases.length) {
    const result = await runner.run(runInput)
    return {
      status: result.status,
      timeMs: result.timeMs,
      memoryBytes: result.memoryBytes,
      score: result.status === 'AC' ? 100 : 0,
      maxScore: 100,
      message: result.stderr || Buffer.from(result.stdout).toString('utf8'),
      cases: []
    }
  }

  const weights = caseWeights(testCases)
  const maxScore = weights.reduce((total, weight) => total + weight, 0)

  const cases = []
  let score = 0
  let timeMs = 0
  let memoryBytes = 0
  let firstFailure: { index: number; status: JudgeStatus; message: string; name?: string } | null =
    null

  for (const [index, testCase] of testCases.entries()) {
    const result = await runner.run({
      ...runInput,
      stdin: new TextEncoder().encode(testCase.input)
    })

    const compared =
      result.status === 'AC'
        ? checkerImageId
          ? await runChecker({
              runner,
              scopeId: runInput.scopeId,
              checkerImageId,
              limits: runInput.limits,
              input: testCase.input,
              output: new TextDecoder().decode(result.stdout),
              answer: testCase.output
            })
          : compareOutput(result.stdout, testCase.output)
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

    if (caseStatus !== 'AC' && !firstFailure) {
      firstFailure = {
        index,
        status: caseStatus,
        message: caseMessage,
        name: testCase.hidden ? undefined : testCase.name
      }
    }
  }

  const status: JudgeStatus = firstFailure ? firstFailure.status : 'AC'
  const message = firstFailure
    ? `score ${score}/${maxScore} — case ${firstFailure.index + 1}` +
      `${firstFailure.name ? ` (${firstFailure.name})` : ''}: ${firstFailure.message}`
    : 'accepted'

  return {
    status,
    timeMs,
    memoryBytes,
    score,
    maxScore,
    message,
    cases
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

interface RunCheckerInput {
  runner: Runner
  scopeId: string
  checkerImageId: string
  limits: RunInput['limits']
  input: string
  output: string
  answer: string
}

async function runChecker(
  input: RunCheckerInput
): Promise<{ status: JudgeStatus; message: string }> {
  // Decode the three artifacts from env into files, then invoke the checker as
  // `checker <input> <output> <answer>` (testlib exit codes).
  const command = [
    '/bin/sh',
    '-lc',
    [
      'set -e',
      'printf %s "$DOJ_CHK_INPUT" | base64 -d > /tmp/input.txt',
      'printf %s "$DOJ_CHK_OUTPUT" | base64 -d > /tmp/output.txt',
      'printf %s "$DOJ_CHK_ANSWER" | base64 -d > /tmp/answer.txt',
      '/workspace/checker /tmp/input.txt /tmp/output.txt /tmp/answer.txt'
    ].join('\n')
  ]

  const result = await input.runner.run({
    scopeId: input.scopeId,
    imageId: input.checkerImageId,
    command,
    limits: input.limits,
    env: {
      DOJ_CHK_INPUT: Buffer.from(input.input).toString('base64'),
      DOJ_CHK_OUTPUT: Buffer.from(input.output).toString('base64'),
      DOJ_CHK_ANSWER: Buffer.from(input.answer).toString('base64')
    }
  })

  const detail = result.stderr.trim() || Buffer.from(result.stdout).toString('utf8').trim()
  if (result.exitCode === 0) {
    return { status: 'AC', message: detail || 'accepted' }
  }
  if (result.exitCode === 2) {
    return { status: 'PE', message: detail || 'presentation error' }
  }
  return { status: 'WA', message: detail || 'wrong answer' }
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
