import type { JudgeAgentPayload, JudgeAgentResult } from '@doj/shared/agent'
import type { ProblemTestCase } from '@doj/shared/judge'
import type { JudgeStatus } from '@doj/shared/status'
import type { Runner } from './types'

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

  try {
    if (!build.ok || !build.imageId) {
      return {
        status: 'CE',
        timeMs: 0,
        memoryBytes: 0,
        message: build.logs,
        cases: []
      }
    }

    return await judgeCases({
      runner,
      scopeId: input.scopeId,
      imageId: build.imageId,
      command: input.language.command?.length ? input.language.command : undefined,
      testCases: input.testCases,
      limits: input.limits
    })
  } finally {
    await runner.cleanup({ scopeId: input.scopeId })
  }
}

type RunInput = Parameters<Runner['run']>[0]
type JudgeCasesInput = RunInput & {
  runner: Runner
  testCases: ProblemTestCase[]
}

async function judgeCases(input: JudgeCasesInput): Promise<JudgeAgentResult> {
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
    const caseMessage = buildCaseMessage({
      status: caseStatus,
      hidden: testCase.hidden === true,
      message: result.status === 'AC' ? compared.message : result.stderr
    })
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
      const caseName = testCase.hidden ? '' : testCase.name
      message = `case ${index + 1}${caseName ? ` (${caseName})` : ''}: ${caseMessage}`
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
