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
  points?: number
}

export interface CaseResult {
  caseIndex: number
  status: JudgeStatus
  timeMs: number
  memoryBytes: number
  score?: number
  message?: string
}

export interface SubmissionResult {
  status: JudgeStatus
  timeMs: number
  memoryBytes: number
  score?: number
  message?: string
  cases: CaseResult[]
}
