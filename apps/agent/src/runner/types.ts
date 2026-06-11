import type { JudgeStatus } from '@doj/shared/status'

export interface RunnerLimit {
  timeMs: number
  memoryBytes: number
  outputBytes?: number
}

export interface RunnerCaseResult {
  caseNo: number
  status: JudgeStatus
  timeMs: number
  memoryBytes: number
  message: string
}

export interface BuildInput {
  scopeId: string
  dockerfile: string
  files: Record<string, string | Uint8Array>
  limits: Pick<RunnerLimit, 'timeMs' | 'memoryBytes'>
}

export interface BuildResult {
  ok: boolean
  imageId?: string
  logs: string
  cached?: boolean
}

export interface RunInput {
  scopeId: string
  imageId: string
  command?: string[]
  env?: Record<string, string>
  stdin?: Uint8Array
  limits: RunnerLimit
  signal?: AbortSignal
}

export interface RunResult {
  status: JudgeStatus
  timeMs: number
  memoryBytes: number
  exitCode: number | null
  signal?: string
  stdout: Uint8Array
  stderr: string
}

export interface CleanupScope {
  scopeId: string
}

export interface PackageBuildInput {
  scopeId: string
  // Files that make up a build context. One of them MUST be `Dockerfile`.
  files: Record<string, string | Uint8Array>
  limits?: Pick<RunnerLimit, 'timeMs' | 'memoryBytes'>
  // Trusted (problem-author) images skip CPU/memory caps; untrusted (submission)
  // images get the configured limits applied at build time.
  trusted: boolean
  // Optional immutable cache key. Only trusted package builds should use this by
  // default, so user source images are not kept around longer than the job.
  cacheKey?: string
  signal?: AbortSignal
}

export interface DuelInput {
  scopeId: string
  // Problem image A (interactor + checker) and submission image B.
  judgeImageId: string
  testerImageId: string
  dataDir: string
  sourcePath: string
  limits: RunnerLimit
  // Extra env for A only. CASE_NO/INPUT/OUT/SOURCE are prepared by the caller.
  judgeEnv?: Record<string, string>
  signal?: AbortSignal
}

export interface DuelResult {
  status: JudgeStatus
  timeMs: number
  memoryBytes: number
  message: string
}

export interface Runner {
  build(input: BuildInput): Promise<BuildResult>
  run(input: RunInput): Promise<RunResult>
  cleanup(scope: CleanupScope): Promise<void>
  buildPackage(input: PackageBuildInput): Promise<BuildResult>
  duel(input: DuelInput): Promise<DuelResult>
}
