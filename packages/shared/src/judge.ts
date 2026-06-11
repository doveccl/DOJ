import type { JudgeStatus } from './status'

export type { JudgeStatus } from './status'

export interface JudgeCase {
  caseNo: number
  inputPath: string
  answerPath: string
}

export interface JudgeLimit {
  timeLimit: number
  memoryLimit: number
}

export type JudgerSpec =
  | { kind: 'prebuilt'; image: string; check: 'trim' | 'pe' }
  | { kind: 'custom'; bundleHash: string }

export interface SubmissionPackage {
  languageId: string
  source: string
  code: string
  dockerfile: string
}

export interface JudgePayload {
  submissionId: number
  problemId: number
  bundleHash: string
  judger: JudgerSpec
  cases: JudgeCase[]
  limits: JudgeLimit
  submission: SubmissionPackage
}

export type JudgePhase = 'queued' | 'building-a' | 'building-b' | 'running' | 'finished'

export interface JudgeProgress {
  phase: JudgePhase
  caseNo?: number
  completedCases?: number
  totalCases?: number
  status?: JudgeStatus
  timeMs?: number
  memoryBytes?: number
  message?: string
}

export interface JudgeCaseResult {
  caseNo: number
  status: JudgeStatus
  timeMs: number
  memoryBytes: number
  message: string
}

export interface JudgeResult {
  status: JudgeStatus
  timeMs: number
  memoryBytes: number
  message: string
  cases: JudgeCaseResult[]
}
