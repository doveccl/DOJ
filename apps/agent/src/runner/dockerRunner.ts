import Docker from 'dockerode'
import { readFile } from 'node:fs/promises'
import { existsSync } from 'node:fs'
import { homedir } from 'node:os'
import { Readable } from 'node:stream'
import { join, resolve } from 'node:path'
import tar from 'tar-stream'
import type { JudgeStatus } from '@doj/shared/status'
import type {
  BuildInput,
  BuildResult,
  CleanupScope,
  DuelInput,
  DuelResult,
  PackageBuildInput,
  RunInput,
  Runner,
  RunResult
} from './types'

const defaultOutputLimitBytes = 64 * 1024 * 1024
// Compiling untrusted submissions needs far more memory than the program's
// runtime limit (e.g. cc1plus), so build-time memory is a separate generous cap.
const buildMemoryBytes = 2 * 1024 * 1024 * 1024

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

  // Build an image from an arbitrary file set (a "package"). One file MUST be
  // `Dockerfile`. Unlike build(), success/failure is detected from the build
  // stream's error event (not a log regex), and untrusted packages get CPU/mem
  // build caps. Used for the A (problem) / B (submission) container model.
  async buildPackage(input: PackageBuildInput): Promise<BuildResult> {
    throwIfCancelled(input.signal)
    const cacheTag = input.cacheKey ? `doj-cache-${input.cacheKey.slice(0, 48)}:latest` : null
    if (cacheTag && (await this.imageExists(cacheTag))) {
      return {
        ok: true,
        imageId: cacheTag,
        logs: `cache hit: ${cacheTag}`,
        cached: true
      }
    }

    const tag = cacheTag ?? `doj-scope-${input.scopeId.toLowerCase()}:latest`
    const context = createFilesBuildContext(input.files)
    const buildOptions: Docker.ImageBuildOptions = {
      t: tag,
      labels: input.cacheKey
        ? { 'doj.cache': 'package', 'doj.cache.key': input.cacheKey }
        : { 'doj.scope': input.scopeId },
      forcerm: true,
      rm: true
    }
    if (!input.trusted && input.limits) {
      // Cap build-time CPU for untrusted submission packages. Memory uses a
      // generous build cap (not the runtime limit) so the compiler isn't OOM-killed.
      buildOptions.memory = buildMemoryBytes
      buildOptions.cpuperiod = 100_000
      buildOptions.cpuquota = 100_000
    }
    const stream = await this.docker.buildImage(context, buildOptions)

    const logs: string[] = []
    let imageId = tag
    let buildError: string | null = null

    await new Promise<void>((resolve, reject) => {
      const abort = () => {
        ;(stream as unknown as { destroy?: (error?: Error) => void }).destroy?.(cancelledError())
        reject(cancelledError())
      }
      input.signal?.addEventListener('abort', abort, { once: true })
      this.docker.modem.followProgress(
        stream,
        (error: Error | null, output: Array<Record<string, unknown>>) => {
          input.signal?.removeEventListener('abort', abort)
          if (error) {
            reject(error)
            return
          }
          for (const item of output) {
            if (typeof item.stream === 'string') logs.push(item.stream)
            if (typeof item.error === 'string') {
              logs.push(item.error)
              buildError = item.error
            }
            const aux = item.aux as { ID?: string } | undefined
            if (aux?.ID) imageId = aux.ID
          }
          resolve()
        }
      )
    })
    throwIfCancelled(input.signal)

    return {
      ok: buildError === null,
      imageId: buildError === null ? imageId : undefined,
      logs: logs.join(''),
      cached: false
    }
  }

  async run(input: RunInput): Promise<RunResult> {
    throwIfCancelled(input.signal)
    const outputLimit = input.limits.outputBytes || defaultOutputLimitBytes
    const startedAt = Date.now()
    const stdout = new CappedBuffer(outputLimit)
    const stderr = new CappedTextBuffer(outputLimit)
    let peakMemoryBytes = 0
    let cpuTimeNs = 0
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
        NanoCpus: 1_000_000_000,
        PidsLimit: 128,
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

    const abort = () => container.kill().catch(() => {})
    input.signal?.addEventListener('abort', abort, { once: true })

    try {
      const attach = await container.attach({
        stream: true,
        stdin: !!input.stdin && !stdinCommand,
        stdout: true,
        stderr: true
      })
      consumeAttachStream(
        attach as unknown as Readable,
        (chunk: Buffer) => stdout.push(chunk),
        (chunk: Buffer) => stderr.push(chunk)
      )

      await container.start()

      if (input.stdin && !stdinCommand) {
        attach.write(Buffer.from(input.stdin))
        attach.end()
      }

      const stats = await watchStats(container, {
        memory: (memoryBytes) => {
          peakMemoryBytes = Math.max(peakMemoryBytes, memoryBytes)
        },
        cpu: (totalCpuNs) => {
          cpuTimeNs = Math.max(cpuTimeNs, totalCpuNs)
          if (cpuTimeNs >= input.limits.timeMs * 1_000_000) {
            timedOut = true
            container.kill().catch(() => {})
          }
        }
      })

      // Wall-clock safety net: a program that sleeps without burning CPU would
      // never trip the CPU-time limit, so cap real time generously above it.
      const wallClockCapMs = Math.max(input.limits.timeMs * 3, input.limits.timeMs + 5_000, 10_000)
      const timeout = setTimeout(() => {
        timedOut = true
        container.kill().catch(() => {})
      }, wallClockCapMs)

      const waitResult = await container.wait()
      clearTimeout(timeout)
      stats.stop()
      await stats.done
      throwIfCancelled(input.signal)

      const inspect = await container.inspect()
      const exitCode = waitResult.StatusCode ?? inspect.State.ExitCode
      const wallMs = Date.now() - startedAt
      // Prefer CPU time for reproducibility; fall back to wall-clock if the
      // stats stream never reported usage (very short-lived containers).
      const timeMs = cpuTimeNs > 0 ? Math.round(cpuTimeNs / 1_000_000) : wallMs
      peakMemoryBytes = Math.max(
        peakMemoryBytes,
        await readCgroupPeakMemoryBytes(inspect.State.Pid)
      )
      const oomKilled = inspect.State.OOMKilled || exitCode === 137
      const outputExceeded = stdout.truncated || stderr.truncated
      const cpuExceeded = cpuTimeNs >= input.limits.timeMs * 1_000_000

      return {
        status:
          timedOut || cpuExceeded
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
      input.signal?.removeEventListener('abort', abort)
      await container.remove({ force: true }).catch(() => {})
    }
  }

  // The A (problem) / B (submission) duel. A and B run once each, connected by
  // two named FIFOs on a shared volume: A writes a2b / reads b2a, B reads a2b /
  // writes b2a. Because they talk over real pipes (not a multiplexed exec
  // stream), A can half-close its write end so a B blocked on read-until-EOF
  // terminates while A stays alive to read B's answer — this unifies batch,
  // special-judge and interactive problems in one mechanism. A is the
  // interactor + checker: verdict from /tmp/result.json if present, else its
  // exit code (testlib enum 0=AC..8=SE). B (untrusted) is CPU/memory limited;
  // B OOM -> MLE, B nonzero exit -> RE. A.stderr is the message.
  async duel(input: DuelInput): Promise<DuelResult> {
    throwIfCancelled(input.signal)
    await this.ensureImage(input.judgeImageId)
    const startedAt = Date.now()
    let peakMemoryBytes = 0
    let cpuTimeNs = 0
    let timedOut = false

    const volumeName = `doj-duel-${input.scopeId.toLowerCase()}-${Date.now()}`
    await this.docker.createVolume({ Name: volumeName, Labels: { 'doj.scope': input.scopeId } })
    // Pre-create the FIFOs in the shared volume, world-rw so the unprivileged
    // (nobody) submission container can open them.
    await this.runThrowaway(
      volumeName,
      input.scopeId,
      'mkfifo /pipe/a2b /pipe/b2a && chmod 666 /pipe/a2b /pipe/b2a'
    )

    const judgeCmd = await this.getImageCommand(input.judgeImageId)
    const testerCmd = await this.getImageCommand(input.testerImageId)

    const judge = await this.docker.createContainer({
      Image: input.judgeImageId,
      User: '65534:65534',
      // A: stdout -> a2b, stdin <- b2a, stderr stays on the container log.
      Entrypoint: ['/bin/sh', '-c'],
      Cmd: [`exec 1>/pipe/a2b 0</pipe/b2a; exec "$@"`, 'sh', ...wrapCmd(judgeCmd)],
      Env: Object.entries(input.judgeEnv ?? {}).map(([key, value]) => `${key}=${value}`),
      NetworkDisabled: true,
      Labels: { 'doj.scope': input.scopeId },
      HostConfig: {
        NetworkMode: 'none',
        Memory: Math.max(input.limits.memoryBytes * 2, input.limits.memoryBytes + 128 * 1024 * 1024),
        NanoCpus: 1_000_000_000,
        PidsLimit: 128,
        ReadonlyRootfs: true,
        CapDrop: ['ALL'],
        SecurityOpt: ['no-new-privileges:true'],
        IpcMode: 'none',
        Binds: [
          `${volumeName}:/pipe`,
          `${input.dataDir}:/data:ro`,
          `${input.sourcePath}:/submission/source:ro`
        ],
        Tmpfs: { '/tmp': 'rw,nosuid,nodev,size=64m' }
      }
    })
    const tester = await this.docker.createContainer({
      Image: input.testerImageId,
      User: '65534:65534',
      // B: stdin <- a2b, stdout -> b2a, stderr discarded.
      Entrypoint: ['/bin/sh', '-c'],
      Cmd: [`exec 0</pipe/a2b 1>/pipe/b2a 2>/dev/null; exec "$@"`, 'sh', ...wrapCmd(testerCmd)],
      NetworkDisabled: true,
      Labels: { 'doj.scope': input.scopeId },
      HostConfig: {
        NetworkMode: 'none',
        Memory: input.limits.memoryBytes,
        NanoCpus: 1_000_000_000,
        PidsLimit: 128,
        CapDrop: ['ALL'],
        SecurityOpt: ['no-new-privileges:true'],
        IpcMode: 'none',
        Binds: [`${volumeName}:/pipe`],
        Tmpfs: {
          '/tmp': 'rw,nosuid,nodev,size=64m',
          '/run': 'rw,nosuid,nodev,size=16m'
        }
      }
    })

    const abort = () => {
      tester.kill().catch(() => {})
      judge.kill().catch(() => {})
    }
    input.signal?.addEventListener('abort', abort, { once: true })

    try {
      await judge.start()
      await tester.start()

      const stats = await watchStats(tester, {
        memory: (memoryBytes) => {
          peakMemoryBytes = Math.max(peakMemoryBytes, memoryBytes)
        },
        cpu: (totalCpuNs) => {
          cpuTimeNs = Math.max(cpuTimeNs, totalCpuNs)
          if (cpuTimeNs >= input.limits.timeMs * 1_000_000) {
            timedOut = true
            tester.kill().catch(() => {})
            judge.kill().catch(() => {})
          }
        }
      })

      // Wall-clock cap kills malicious sleepers that never burn CPU.
      const wallClockCapMs = Math.max(input.limits.timeMs * 3, input.limits.timeMs + 5_000, 10_000)
      let wallKilled = false
      const timeout = setTimeout(() => {
        timedOut = true
        wallKilled = true
        tester.kill().catch(() => {})
        judge.kill().catch(() => {})
      }, wallClockCapMs)

      const [judgeWait, testerWait] = await Promise.all([judge.wait(), tester.wait()])
      clearTimeout(timeout)
      stats.stop()
      await stats.done
      throwIfCancelled(input.signal)

      const testerInspect = await tester.inspect()
      const testerExit = testerWait.StatusCode ?? testerInspect.State.ExitCode
      peakMemoryBytes = Math.max(
        peakMemoryBytes,
        await readCgroupPeakMemoryBytes(testerInspect.State.Pid)
      )
      const wallMs = Date.now() - startedAt
      // Prefer CPU time; but if we killed a non-CPU-burning sleeper on the wall
      // clock, report wall time (never "TLE 0ms").
      const timeMs = wallKilled
        ? wallMs
        : cpuTimeNs > 0
          ? Math.round(cpuTimeNs / 1_000_000)
          : wallMs
      const testerOom = testerInspect.State.OOMKilled || testerExit === 137

      let status: JudgeStatus
      const detail = (await this.readContainerLog(judge)).trim()

      if (timedOut) {
        status = 'TLE'
      } else if (testerOom) {
        status = 'MLE'
      } else if (testerExit !== 0) {
        status = 'RE'
      } else {
        const judgeInspect = await judge.inspect()
        const judgeExit = judgeWait.StatusCode ?? judgeInspect.State.ExitCode
        status = exitCodeToStatus(judgeExit)
      }

      return { status, timeMs, memoryBytes: peakMemoryBytes, message: detail }
    } finally {
      input.signal?.removeEventListener('abort', abort)
      await judge.remove({ force: true }).catch(() => {})
      await tester.remove({ force: true }).catch(() => {})
      await this.docker
        .getVolume(volumeName)
        .remove({ force: true })
        .catch(() => {})
    }
  }

  // Run a short root command against the shared volume (e.g. to mkfifo).
  private async runThrowaway(volumeName: string, scopeId: string, shellCommand: string) {
    const container = await this.docker.createContainer({
      Image: 'busybox:latest',
      Cmd: ['/bin/sh', '-c', shellCommand],
      Labels: { 'doj.scope': scopeId },
      HostConfig: { Binds: [`${volumeName}:/pipe`], NetworkMode: 'none' }
    })
    try {
      await container.start()
      await container.wait()
    } finally {
      await container.remove({ force: true }).catch(() => {})
    }
  }

  private async readContainerLog(container: Docker.Container) {
    try {
      const buffer = (await container.logs({ stdout: false, stderr: true })) as unknown as Buffer
      // Docker multiplexes logs with an 8-byte header per frame; strip them.
      return demuxDockerLog(buffer)
    } catch {
      return ''
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

  async cleanupPackageCache(cacheKey: string): Promise<void> {
    const images = await this.docker.listImages({
      filters: { label: [`doj.cache.key=${cacheKey}`] }
    })
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
      return [...(inspect.Config?.Entrypoint ?? []), ...(inspect.Config?.Cmd ?? [])]
    } catch {
      return undefined
    }
  }

  private async ensureImage(imageId: string) {
    if (await this.imageExists(imageId)) return
    const stream = await this.docker.pull(imageId)
    await new Promise<void>((resolve, reject) => {
      this.docker.modem.followProgress(stream, (error: Error | null) => {
        if (error) reject(error)
        else resolve()
      })
    })
  }

  private async imageExists(imageId: string) {
    try {
      await this.docker.getImage(imageId).inspect()
      return true
    } catch {
      return false
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

// Build a tar context from an arbitrary file set (the package must already
// include its own `Dockerfile` entry).
function createFilesBuildContext(files: Record<string, string | Uint8Array>) {
  const pack = tar.pack()
  for (const [name, content] of Object.entries(files)) {
    pack.entry({ name }, typeof content === 'string' ? content : Buffer.from(content))
  }
  pack.finalize()
  return pack
}

function cancelledError() {
  return new Error('judge job cancelled')
}

function throwIfCancelled(signal: AbortSignal | undefined) {
  if (signal?.aborted) throw cancelledError()
}

const statusByExitCode: JudgeStatus[] = ['AC', 'WA', 'PE', 'TLE', 'MLE', 'OLE']

function exitCodeToStatus(exitCode: number | null): JudgeStatus {
  if (exitCode === null) return 'SE'
  return statusByExitCode[exitCode] ?? 'SE'
}

// Build the `"$@"` argument list for the FIFO redirect wrapper. Falls back to a
// shell if the image declares no command (the entrypoint already redirected fds).
function wrapCmd(cmd: string[] | undefined): string[] {
  return cmd && cmd.length ? cmd : ['/bin/sh', '-c', 'cat']
}

// Strip Docker's 8-byte stream-multiplexing frame headers from a raw log buffer.
function demuxDockerLog(buffer: Buffer): string {
  if (buffer.length < 8) return buffer.toString('utf8')
  const parts: Buffer[] = []
  let offset = 0
  while (offset + 8 <= buffer.length) {
    const frameType = buffer[offset]
    const length = buffer.readUInt32BE(offset + 4)
    // A valid frame header has type in {0,1,2} and the payload fits.
    if (frameType > 2 || offset + 8 + length > buffer.length) {
      return buffer.toString('utf8')
    }
    parts.push(buffer.subarray(offset + 8, offset + 8 + length))
    offset += 8 + length
  }
  return Buffer.concat(parts).toString('utf8')
}

// Docker attach streams multiplex stdout/stderr as 8-byte framed chunks when TTY
// is disabled. Parse the frames directly instead of relying on dockerode's
// demuxStream internals, which expect a destroySoon-capable socket on some paths.
function consumeAttachStream(
  stream: Readable,
  stdout: (chunk: Buffer) => void,
  stderr: (chunk: Buffer) => void
) {
  let pending = Buffer.alloc(0)
  stream.on('data', (chunk) => {
    pending = Buffer.concat([pending, Buffer.from(chunk)])
    for (;;) {
      if (pending.length < 8) return
      const streamType = pending[0]
      const length = pending.readUInt32BE(4)
      if (![0, 1, 2].includes(streamType)) {
        stdout(pending)
        pending = Buffer.alloc(0)
        return
      }
      if (pending.length < 8 + length) return
      const payload = pending.subarray(8, 8 + length)
      if (streamType === 2) stderr(payload)
      else stdout(payload)
      pending = pending.subarray(8 + length)
    }
  })
  stream.on('end', () => {
    if (pending.length) stdout(pending)
  })
}

async function watchStats(
  container: Docker.Container,
  update: { memory: (memoryBytes: number) => void; cpu: (totalCpuNs: number) => void }
) {
  const stream = (await container.stats({ stream: true })) as Readable
  let buffer = ''

  const handle = (raw: string) => {
    const stats = JSON.parse(raw)
    update.memory(readMemoryUsage(stats))
    update.cpu(readCpuUsageNs(stats))
  }

  const done = new Promise<void>((resolve) => {
    stream.on('data', (chunk) => {
      buffer += Buffer.from(chunk).toString('utf8')
      const lines = buffer.split('\n')
      buffer = lines.pop() ?? ''

      for (const line of lines) {
        if (!line.trim()) continue
        try {
          handle(line)
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
          handle(buffer)
        } catch {
          // Ignore trailing partial JSON.
        }
      }
    }
  }
}

function readCpuUsageNs(stats: unknown) {
  const cpu = (
    stats as {
      cpu_stats?: { cpu_usage?: { total_usage?: number } }
    }
  ).cpu_stats
  return cpu?.cpu_usage?.total_usage ?? 0
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
