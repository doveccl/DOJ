import type { JudgeStatus } from './status'

export interface JudgeLimit {
  timeMs: number
  memoryBytes: number
  outputBytes: number
}

export interface ProblemTestCase {
  name?: string
  input: string
  output: string
  hidden?: boolean
}

export interface CaseResult {
  caseIndex: number
  status: JudgeStatus
  timeMs: number
  memoryBytes: number
  message?: string
}

export interface SubmissionResult {
  status: JudgeStatus
  timeMs: number
  memoryBytes: number
  message?: string
  cases: CaseResult[]
}

export interface JudgeTaskPayload {
  submissionId: number
  problemId: number
  problemVersionId: number
  testdataObjectKey: string
  languageId: string
  sourceCode: string
  limits: JudgeLimit
}
