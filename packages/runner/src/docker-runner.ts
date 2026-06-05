import Docker from 'dockerode'
import { readFile } from 'node:fs/promises'
import { existsSync } from 'node:fs'
import { homedir } from 'node:os'
import { Readable, Writable } from 'node:stream'
import { join, resolve } from 'node:path'
import tar from 'tar-stream'
import type { BuildInput, BuildResult, CleanupScope, RunInput, Runner, RunResult } from './types'

const defaultOutputLimitBytes = 64 * 1024 * 1024

export interface DockerRunnerOptions {
  endpoint?: string | null
  /** @deprecated Prefer endpoint credentials, for example https://user:pass@docker.example.com. */
  authHeader?: string | null
}

export class DockerRunner implements Runner {
  private readonly docker: Docker

  constructor(options: DockerRunnerOptions = {}) {
    this.docker = createDockerClient(options)
  }

  async check() {
    await this.docker.ping()
    const version = await this.docker.version()
    return {
      version: version.Version,
      apiVersion: version.ApiVersion,
      os: version.Os,
      arch: version.Arch
    }
  }

  async build(input: BuildInput): Promise<BuildResult> {
    const tag = `doj-scope-${input.scopeId.toLowerCase()}:latest`
    const context = createBuildContext(input)
    const stream = await this.docker.buildImage(context, {
      t: tag,
      labels: {
        'doj.scope': input.scopeId
      },
      forcerm: true,
      rm: true
    })

    const logs: string[] = []
    let imageId = tag

    await new Promise<void>((resolve, reject) => {
      this.docker.modem.followProgress(
        stream,
        (error: Error | null, output: Array<Record<string, unknown>>) => {
          if (error) {
            reject(error)
            return
          }

          for (const item of output) {
            if (typeof item.stream === 'string') logs.push(item.stream)
            if (typeof item.error === 'string') logs.push(item.error)
            const aux = item.aux as { ID?: string } | undefined
            if (aux?.ID) imageId = aux.ID
          }
          resolve()
        }
      )
    })

    const failed = logs.some((line) => /error|failed/i.test(line))
    return {
      ok: !failed,
      imageId,
      logs: logs.join('')
    }
  }

  async run(input: RunInput): Promise<RunResult> {
    const outputLimit = input.limits.outputBytes || defaultOutputLimitBytes
    const startedAt = Date.now()
    const stdout = new CappedBuffer(outputLimit)
    const stderr = new CappedTextBuffer(outputLimit)
    let peakMemoryBytes = 0
    let timedOut = false
    const effectiveCommand =
      input.stdin && !input.command?.length
        ? await this.getImageCommand(input.imageId)
        : input.command
    const stdinCommand =
      input.stdin && effectiveCommand?.length ? createStdinCommand(effectiveCommand) : null

    const container = await this.docker.createContainer({
      Image: input.imageId,
      Cmd: stdinCommand?.command ?? input.command,
      User: '65534:65534',
      Env: Object.entries({
        ...(input.env ?? {}),
        ...(stdinCommand
          ? { DOJ_STDIN_BASE64: Buffer.from(input.stdin ?? []).toString('base64') }
          : {})
      }).map(([key, value]) => `${key}=${value}`),
      AttachStdin: !stdinCommand,
      AttachStdout: true,
      AttachStderr: true,
      OpenStdin: !!input.stdin && !stdinCommand,
      StdinOnce: true,
      NetworkDisabled: true,
      Labels: {
        'doj.scope': input.scopeId
      },
      HostConfig: {
        AutoRemove: false,
        NetworkMode: 'none',
        Memory: input.limits.memoryBytes,
        PidsLimit: 256,
        ReadonlyRootfs: true,
        CapDrop: ['ALL'],
        SecurityOpt: ['no-new-privileges:true'],
        IpcMode: 'none',
        Tmpfs: {
          '/tmp': 'rw,nosuid,nodev,size=64m',
          '/run': 'rw,nosuid,nodev,size=16m'
        }
      }
    })

    try {
      const attach = await container.attach({
        stream: true,
        stdin: !!input.stdin && !stdinCommand,
        stdout: true,
        stderr: true
      })
      patchDestroySoon(attach)

      this.docker.modem.demuxStream(
        attach,
        writableFrom((chunk) => stdout.push(chunk)),
        writableFrom((chunk) => stderr.push(chunk))
      )

      await container.start()

      if (input.stdin && !stdinCommand) {
        attach.write(Buffer.from(input.stdin))
        attach.end()
      }

      const stats = await watchStats(container, (memoryBytes) => {
        peakMemoryBytes = Math.max(peakMemoryBytes, memoryBytes)
      })

      const timeout = setTimeout(() => {
        timedOut = true
        container.kill().catch(() => {})
      }, input.limits.timeMs)

      const waitResult = await container.wait()
      clearTimeout(timeout)
      stats.stop()
      await stats.done

      const inspect = await container.inspect()
      const exitCode = waitResult.StatusCode ?? inspect.State.ExitCode
      const timeMs = Date.now() - startedAt
      peakMemoryBytes = Math.max(
        peakMemoryBytes,
        await readCgroupPeakMemoryBytes(inspect.State.Pid)
      )
      const oomKilled = inspect.State.OOMKilled || exitCode === 137
      const outputExceeded = stdout.truncated || stderr.truncated

      return {
        status: timedOut
          ? 'TLE'
          : oomKilled
            ? 'MLE'
            : outputExceeded
              ? 'OLE'
              : exitCode === 0
                ? 'AC'
                : 'RE',
        timeMs,
        memoryBytes: peakMemoryBytes,
        exitCode,
        signal: inspect.State.Error || undefined,
        stdout: stdout.bytes(),
        stderr: stderr.text()
      }
    } finally {
      await container.remove({ force: true }).catch(() => {})
    }
  }

  async cleanup(input: CleanupScope): Promise<void> {
    const label = `doj.scope=${input.scopeId}`
    const containers = await this.docker.listContainers({ all: true, filters: { label: [label] } })
    const images = await this.docker.listImages({ filters: { label: [label] } })

    await Promise.all(
      containers.map((item) =>
        this.docker
          .getContainer(item.Id)
          .remove({ force: true })
          .catch(() => {})
      )
    )
    await Promise.all(
      images.flatMap((item) =>
        item.Id
          ? [
              this.docker
                .getImage(item.Id)
                .remove({ force: true })
                .catch(() => {})
            ]
          : []
      )
    )
  }

  private async getImageCommand(imageId: string) {
    try {
      const inspect = await this.docker.getImage(imageId).inspect()
      return inspect.Config?.Cmd ?? undefined
    } catch {
      return undefined
    }
  }
}

function createStdinCommand(command: string[]) {
  return {
    command: [
      '/bin/sh',
      '-lc',
      'printf %s "$DOJ_STDIN_BASE64" | base64 -d | "$@"',
      'doj-run',
      ...command
    ]
  }
}

function patchDestroySoon(stream: unknown) {
  const candidate = stream as {
    socket?: { destroy?: () => void; destroySoon?: () => void }
    _output?: { socket?: { destroy?: () => void; destroySoon?: () => void } }
  }
  const socket = candidate._output?.socket ?? candidate.socket
  if (socket?.destroy && !socket.destroySoon) socket.destroySoon = socket.destroy.bind(socket)
}

function createDockerClient(options: DockerRunnerOptions = {}) {
  const host = options.endpoint || process.env.DOCKER_HOST
  const endpointAuth = host ? readEndpointAuth(host) : undefined
  const headers =
    endpointAuth || options.authHeader
      ? { Authorization: endpointAuth ?? options.authHeader ?? '' }
      : undefined
  if (host?.startsWith('unix://')) {
    return new Docker({ socketPath: host.slice('unix://'.length), headers })
  }

  if (host?.startsWith('http://') || host?.startsWith('https://')) {
    const url = new URL(host)
    return new Docker({
      protocol: url.protocol === 'https:' ? 'https' : 'http',
      host: url.hostname,
      port: url.port || (url.protocol === 'https:' ? 443 : 80),
      headers
    })
  }

  const colimaSocket = join(homedir(), '.colima/default/docker.sock')
  if (!host && existsSync(colimaSocket)) {
    return new Docker({ socketPath: colimaSocket, headers })
  }

  return new Docker(headers ? { headers } : undefined)
}

function readEndpointAuth(endpoint: string) {
  if (!endpoint.startsWith('http://') && !endpoint.startsWith('https://')) return undefined
  const url = new URL(endpoint)
  if (!url.username && !url.password) return undefined
  const user = decodeURIComponent(url.username)
  const password = decodeURIComponent(url.password)
  return `Basic ${Buffer.from(`${user}:${password}`).toString('base64')}`
}

function createBuildContext(input: BuildInput) {
  const pack = tar.pack()
  pack.entry({ name: 'Dockerfile' }, input.dockerfile)

  for (const [name, content] of Object.entries(input.files)) {
    pack.entry({ name }, typeof content === 'string' ? content : Buffer.from(content))
  }

  pack.finalize()
  return pack
}

function writableFrom(write: (chunk: Buffer) => void) {
  return new Writable({
    write(chunk, _encoding, callback) {
      write(Buffer.from(chunk))
      callback()
    }
  })
}

async function watchStats(container: Docker.Container, update: (memoryBytes: number) => void) {
  const stream = (await container.stats({ stream: true })) as Readable
  let buffer = ''

  const done = new Promise<void>((resolve) => {
    stream.on('data', (chunk) => {
      buffer += Buffer.from(chunk).toString('utf8')
      const lines = buffer.split('\n')
      buffer = lines.pop() ?? ''

      for (const line of lines) {
        if (!line.trim()) continue
        try {
          update(readMemoryUsage(JSON.parse(line)))
        } catch {
          // Docker can split JSON across chunks; incomplete data is kept in buffer.
        }
      }
    })

    stream.on('end', resolve)
    stream.on('close', resolve)
    stream.on('error', resolve)
  })

  return {
    done,
    stop() {
      stream.destroy()
      if (buffer.trim()) {
        try {
          update(readMemoryUsage(JSON.parse(buffer)))
        } catch {
          // Ignore trailing partial JSON.
        }
      }
    }
  }
}

function readMemoryUsage(stats: unknown) {
  const memory = (
    stats as {
      memory_stats?: { usage?: number; max_usage?: number; stats?: Record<string, number> }
    }
  ).memory_stats
  if (!memory?.usage && !memory?.max_usage) return 0

  const usage = memory.usage ?? 0
  const maxUsage = memory.max_usage ?? memory.stats?.max_usage ?? 0
  const inactiveFile = memory.stats?.inactive_file ?? memory.stats?.total_inactive_file ?? 0
  return Math.max(adjustMemoryUsage(usage, inactiveFile), adjustMemoryUsage(maxUsage, inactiveFile))
}

function adjustMemoryUsage(usage: number, inactiveFile: number) {
  return Math.max(0, usage - inactiveFile)
}

async function readCgroupPeakMemoryBytes(pid?: number) {
  if (!pid || pid <= 0) return 0

  try {
    const cgroup = await readFile(`/proc/${pid}/cgroup`, 'utf8')
    const lines = cgroup
      .split('\n')
      .map((line) => line.trim())
      .filter(Boolean)

    const cgroup2 = lines.find((line) => line.startsWith('0::'))
    if (cgroup2) {
      const peak = await readNumericFile(
        cgroupFile('/sys/fs/cgroup', cgroup2.split(':')[2], 'memory.peak')
      )
      if (peak) return peak
    }

    const memoryLine = lines.find((line) => line.split(':')[1]?.split(',').includes('memory'))
    if (memoryLine) {
      const peak = await readNumericFile(
        cgroupFile('/sys/fs/cgroup/memory', memoryLine.split(':')[2], 'memory.max_usage_in_bytes')
      )
      if (peak) return peak
    }
  } catch {
    // Remote Docker daemons and Docker Desktop on macOS do not expose container cgroups here.
  }

  return 0
}

function cgroupFile(base: string, cgroupPath: string | undefined, filename: string) {
  const root = resolve(base)
  const candidate = resolve(root, (cgroupPath ?? '').replace(/^\/+/, ''), filename)
  if (!candidate.startsWith(`${root}/`) && candidate !== root) {
    throw new Error(`invalid cgroup path: ${cgroupPath}`)
  }
  return candidate
}

async function readNumericFile(path: string) {
  const text = await readFile(path, 'utf8')
  const value = Number.parseInt(text.trim(), 10)
  return Number.isFinite(value) ? value : 0
}

class CappedBuffer {
  private chunks: Buffer[] = []
  private size = 0
  truncated = false

  constructor(private readonly cap: number) {}

  push(chunk: Buffer) {
    if (this.size >= this.cap) {
      this.truncated = true
      return
    }

    const remaining = this.cap - this.size
    const next = chunk.length > remaining ? chunk.subarray(0, remaining) : chunk
    this.chunks.push(next)
    this.size += next.length
    this.truncated ||= chunk.length > remaining
  }

  bytes() {
    return new Uint8Array(Buffer.concat(this.chunks, this.size))
  }
}

class CappedTextBuffer extends CappedBuffer {
  text() {
    return Buffer.from(this.bytes()).toString('utf8')
  }
}
