import Docker from 'dockerode'
import type { BuildInput, BuildResult, CleanupScope, RunInput, Runner, RunResult } from './types'

const defaultOutputLimitBytes = 64 * 1024 * 1024

export class DockerRunner implements Runner {
  private readonly docker = new Docker()

  async build(input: BuildInput): Promise<BuildResult> {
    return {
      ok: false,
      logs: [
        `Docker build for scope ${input.scopeId} is not implemented yet.`,
        'The runner boundary is in place so the worker can be built independently first.'
      ].join('\n')
    }
  }

  async run(input: RunInput): Promise<RunResult> {
    const outputLimit = input.limits.outputBytes || defaultOutputLimitBytes

    return {
      status: 'SE',
      timeMs: 0,
      memoryBytes: 0,
      exitCode: null,
      stdout: new Uint8Array(),
      stderr: [
        `Docker run for scope ${input.scopeId} is not implemented yet.`,
        `The configured output cap is ${outputLimit} bytes.`
      ].join('\n')
    }
  }

  async cleanup(input: CleanupScope): Promise<void> {
    const label = `doj.scope=${input.scopeId}`
    const containers = await this.docker.listContainers({ all: true, filters: { label: [label] } })
    await Promise.all(
      containers.map((item) => this.docker.getContainer(item.Id).remove({ force: true }).catch(() => {}))
    )
  }
}
