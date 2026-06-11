import { createHash } from 'node:crypto'
import type { JudgeAgentProgress, JudgeAgentResult } from '@doj/shared/agent'
import type { JudgeStatus } from '@doj/shared/status'
import type { Runner, RunnerLimit } from './types'

interface PackageTestCase {
  caseNo: number
  inputPath: string
  answerPath: string
}

export interface PackageJudgeInput {
  scopeId: string
  // Submission (B) build context; must include a `Dockerfile`.
  testerFiles: Record<string, string | Uint8Array>
  // Problem (A) build context; must include a `Dockerfile`.
  problemFiles?: Record<string, string | Uint8Array> | null
  dataDir: string
  sourcePath: string
  testCases: PackageTestCase[]
  limits: RunnerLimit
  problemCacheKey?: string
  // Disabled by default because B contains untrusted user source. Agents may
  // opt in when their Docker image retention policy allows it.
  cacheSubmissionPackage?: boolean
  signal?: AbortSignal
  onProgress?: (progress: JudgeAgentProgress) => void | Promise<void>
}

// Package-based custom judging with the same A/B duel used by prebuilt judging.
// The server, not Agent, owns scoring; Agent only returns verdict/resource facts.
export async function judgePackage(
  runner: Runner,
  input: PackageJudgeInput
): Promise<JudgeAgentResult> {
  const testerScope = `${input.scopeId}-b`
  const problemScope = `${input.scopeId}-a`
  const custom = !!input.problemFiles && 'Dockerfile' in input.problemFiles
  if (!custom) throw new Error('custom judge package requires Dockerfile')

  throwIfCancelled(input.signal)
  await input.onProgress?.({
    phase: 'building-b',
    message: 'Building submission package',
    caseNo: undefined
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
        message: testerBuild.logs,
        cases: []
      }
    }

    await input.onProgress?.({
      phase: 'building-a',
      message: 'Building problem package',
      caseNo: undefined
    })
    const problemBuild = await runner.buildPackage({
      scopeId: problemScope,
      files: input.problemFiles as Record<string, string | Uint8Array>,
      trusted: true,
      cacheKey:
        input.problemCacheKey ??
        packageCacheKey(input.problemFiles as Record<string, string | Uint8Array>),
      signal: input.signal
    })
    throwIfCancelled(input.signal)
    if (!problemBuild.ok || !problemBuild.imageId) {
      return {
        status: 'SE',
        timeMs: 0,
        memoryBytes: 0,
        message: `problem package build failed:\n${problemBuild.logs}`,
        cases: []
      }
    }

    return await judgeCustomCases({
      runner,
      scopeId: input.scopeId,
      judgeImageId: problemBuild.imageId,
      testerImageId: testerBuild.imageId,
      dataDir: input.dataDir,
      sourcePath: input.sourcePath,
      testCases: input.testCases,
      limits: input.limits,
      signal: input.signal,
      onProgress: input.onProgress
    })
  } finally {
    await runner.cleanup({ scopeId: testerScope })
    if (custom) await runner.cleanup({ scopeId: problemScope })
  }
}

interface CustomCasesInput {
  runner: Runner
  scopeId: string
  judgeImageId: string
  testerImageId: string
  dataDir: string
  sourcePath: string
  testCases: PackageTestCase[]
  limits: RunnerLimit
  signal?: AbortSignal
  onProgress?: (progress: JudgeAgentProgress) => void | Promise<void>
}

async function judgeCustomCases(input: CustomCasesInput): Promise<JudgeAgentResult> {
  const { runner, testCases, limits } = input

  const cases = []
  let timeMs = 0
  let memoryBytes = 0
  let firstFailure: { caseNo: number; status: JudgeStatus; message: string } | null = null

  for (const [index, testCase] of testCases.entries()) {
    throwIfCancelled(input.signal)
    await input.onProgress?.({
      phase: 'running',
      message: `Testing case ${index + 1}/${testCases.length}`,
      caseNo: testCase.caseNo
    })
    const result = await runner.duel({
      scopeId: `${input.scopeId}-case-${index}`,
      judgeImageId: input.judgeImageId,
      testerImageId: input.testerImageId,
      dataDir: input.dataDir,
      sourcePath: input.sourcePath,
      limits,
      judgeEnv: {
        CASE_NO: String(testCase.caseNo),
        INPUT: `/data/${testCase.inputPath.replace(/^data\//, '')}`,
        OUT: `/data/${testCase.answerPath.replace(/^data\//, '')}`,
        SOURCE: '/submission/source',
        TIME_LIMIT_MS: String(limits.timeMs),
        MEMORY_LIMIT_BYTES: String(limits.memoryBytes)
      },
      signal: input.signal
    })
    throwIfCancelled(input.signal)
    const caseStatus = result.status
    const caseMessage = truncateMessage(result.message)
    timeMs = Math.max(timeMs, result.timeMs)
    memoryBytes = Math.max(memoryBytes, result.memoryBytes)
    cases.push({
      caseNo: testCase.caseNo,
      status: caseStatus,
      timeMs: result.timeMs,
      memoryBytes: result.memoryBytes,
      message: caseMessage
    })
    await input.onProgress?.({
      phase: 'running',
      message: `Finished case ${index + 1}/${testCases.length}`,
      caseNo: testCase.caseNo,
      status: caseStatus,
      timeMs: result.timeMs,
      memoryBytes: result.memoryBytes
    })
    if (caseStatus !== 'AC' && !firstFailure) {
      firstFailure = {
        caseNo: testCase.caseNo,
        status: caseStatus,
        message: caseMessage
      }
    }
  }

  return finalizeResult({ cases, timeMs, memoryBytes, firstFailure })
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
  timeMs: number
  memoryBytes: number
  firstFailure: { caseNo: number; status: JudgeStatus; message: string } | null
}): JudgeAgentResult {
  const { firstFailure } = input
  const status: JudgeStatus = firstFailure ? firstFailure.status : 'AC'
  const message = firstFailure
    ? `case ${firstFailure.caseNo}: ${firstFailure.message}`
    : 'accepted'
  return {
    status,
    timeMs: input.timeMs,
    memoryBytes: input.memoryBytes,
    message,
    cases: input.cases
  }
}

function truncateMessage(message: string) {
  return message.length > 4096 ? `${message.slice(0, 4076)}\n[message truncated]` : message
}
