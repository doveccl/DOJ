import type {
  JudgeCase,
  JudgeCaseResult,
  JudgePayload,
  JudgeProgress,
  JudgeResult,
  JudgerSpec,
  SubmissionPackage
} from './judge'

export interface JudgeAgentHello {
  key: string
  name: string
  concurrency: number
  version: string
}

export type JudgeAgentCase = JudgeCase
export type JudgeAgentJudger = JudgerSpec
export type JudgeAgentSubmissionPackage = SubmissionPackage
export type JudgeAgentPayload = JudgePayload
export type JudgeAgentProgress = JudgeProgress
export type JudgeAgentCaseResult = JudgeCaseResult
export type JudgeAgentResult = JudgeResult

export type ServerToAgentMessage =
  | {
      type: 'ping'
    }
  | {
      type: 'run'
      jobId: string
      payload: JudgeAgentPayload
    }

export type AgentToServerMessage =
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
