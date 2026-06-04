import type { CaseResult, JudgeLimit } from '@doj/shared/judge'

export interface BuildInput {
  scopeId: string
  dockerfile: string
  files: Record<string, string | Uint8Array>
  limits: Pick<JudgeLimit, 'timeMs' | 'memoryBytes'>
}

export interface BuildResult {
  ok: boolean
  imageId?: string
  logs: string
}

export interface RunInput {
  scopeId: string
  imageId: string
  command?: string[]
  env?: Record<string, string>
  stdin?: Uint8Array
  limits: JudgeLimit
}

export interface RunResult extends Omit<CaseResult, 'caseIndex'> {
  exitCode: number | null
  signal?: string
  stdout: Uint8Array
  stderr: string
}

export interface CleanupScope {
  scopeId: string
}

export interface Runner {
  build(input: BuildInput): Promise<BuildResult>
  run(input: RunInput): Promise<RunResult>
  cleanup(scope: CleanupScope): Promise<void>
}
