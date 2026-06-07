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

// A single file in a problem's judging package, addressed by its path inside the
// build context (e.g. `Dockerfile`, `data/1.in`, `judge.cc`).
export interface JudgeAgentPackageFile {
  path: string
  bucket: string
  objectKey: string
  sizeBytes: number
}

export interface JudgeAgentPayload {
  submissionId: number
  scopeId: string
  // Submission source, also exposed to the problem (A) container as `code`.
  code: string
  limits: {
    timeMs: number
    memoryBytes: number
    outputBytes: number
  }
  // The submission (B) build context, inline: { Dockerfile, <sourceFile> }.
  testerFiles: Record<string, string>
  // The problem (A) package files, fetched from S3 by the agent. When the
  // package contains a `Dockerfile` the run is custom (A is interactor+checker);
  // otherwise the engine runs default mode against the data/inline cases.
  problemFiles: JudgeAgentPackageFile[]
  // Inline sample cases (used as default-mode data + per-case weights when the
  // package ships no data/*.in files).
  inlineTestCases: ProblemTestCase[]
  // For custom packages whose A generates its own data, how many cases to run.
  caseCount: number
}

export interface JudgeAgentCaseResult {
  caseIndex: number
  status: JudgeStatus
  timeMs: number
  memoryBytes: number
  score: number
  message: string
}

export interface JudgeAgentProgress {
  phase: 'building' | 'testing' | 'finished' | 'cancelled'
  message: string
  completedCases: number
  totalCases: number
  currentCase?: number
  case?: JudgeAgentCaseResult
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
  | {
      type: 'cancel'
      jobId: string
      reason: string
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
      type: 'progress'
      jobId: string
      progress: JudgeAgentProgress
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
