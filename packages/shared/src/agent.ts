import type { ProblemTestCase } from './judge'
import type { JudgeStatus } from './status'

export interface JudgeAgentHello {
  key: string
  name: string
  concurrency: number
  labels: string[]
  version: string
}

export interface JudgeAgentFileRef {
  bucket: string
  objectKey: string
  filename: string
  sizeBytes: number
}

export interface JudgeAgentLanguage {
  id: string
  sourceFile: string
  dockerfile: string
  command?: string[]
}

export interface JudgeAgentPayload {
  submissionId: number
  scopeId: string
  sourceCode: string
  language: JudgeAgentLanguage
  limits: {
    timeMs: number
    memoryBytes: number
    outputBytes: number
  }
  testCases: ProblemTestCase[]
  testdataFile?: JudgeAgentFileRef | null
  checker?: JudgeAgentChecker | null
}

export interface JudgeAgentChecker {
  // Special-judge source. Compiled with C++ and invoked per case as
  // `checker <input> <output> <answer>` (testlib-compatible exit codes:
  // 0 = accepted, 2 = presentation error, anything else = wrong answer).
  sourceCode: string
}

export interface JudgeAgentCaseResult {
  caseIndex: number
  status: JudgeStatus
  timeMs: number
  memoryBytes: number
  score: number
  message: string
}

export interface JudgeAgentResult {
  status: JudgeStatus
  timeMs: number
  memoryBytes: number
  score: number
  maxScore: number
  message: string
  cases: JudgeAgentCaseResult[]
}

export type WorkerToAgentMessage =
  | {
      type: 'ping'
    }
  | {
      type: 'run'
      jobId: string
      payload: JudgeAgentPayload
    }

export type AgentToWorkerMessage =
  | {
      type: 'hello'
      info: JudgeAgentHello
    }
  | {
      type: 'pong'
      activeJobs: number
    }
  | {
      type: 'result'
      jobId: string
      result: JudgeAgentResult
    }
  | {
      type: 'error'
      jobId: string
      message: string
    }
