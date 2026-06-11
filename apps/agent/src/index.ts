import type {
  AgentToServerMessage,
  JudgeAgentCaseResult,
  JudgeAgentPayload,
  JudgeAgentProgress,
  JudgeAgentResult,
  ServerToAgentMessage
} from '@doj/shared/agent'
import { createHash, randomBytes, randomUUID } from 'node:crypto'
import { mkdir, readdir, readFile, rm, writeFile } from 'node:fs/promises'
import { homedir, hostname } from 'node:os'
import { dirname, join } from 'node:path'
import type { JudgeStatus } from '@doj/shared/status'
import { DockerRunner } from './runner/dockerRunner'
import { judgePackage } from './runner/judge'

const agentName = process.env.AGENT_NAME ?? hostname() ?? 'agent'
const key = createAgentKey(agentName)
const token = process.env.SECRET ?? 'local-dev-secret-change-me'
const concurrency = Number(process.env.AGENT_CONCURRENCY ?? 1)
const serverUrl = process.env.SERVER ?? 'http://127.0.0.1:7974'
const workerUrl = buildAgentWsUrl(serverUrl)
const runner = new DockerRunner()
const cacheSubmissionPackage = false
const workDir = process.env.AGENT_WORK_DIR ?? join(homedir(), '.doj-agent')
const bundleCacheDir = join(workDir, 'cache')
const runRootDir = join(workDir, 'runs')
const activeJobs = new Set<string>()
const jobControllers = new Map<string, AbortController>()

for (;;) {
  await connectOnce()
  await Bun.sleep(1500)
}

async function connectOnce() {
  const url = buildWorkerUrl()
  console.log(`judge agent ${key} connecting to ${redactToken(url)}`)

  await new Promise<void>((resolve) => {
    const socket = new (WebSocket as unknown as {
      new (url: string, options: { headers: Record<string, string> }): WebSocket
    })(url, {
      headers: {
        authorization: `Bearer ${token}`
      }
    })
    let opened = false

    socket.addEventListener('open', () => {
      opened = true
      send(socket, {
        type: 'hello',
        info: {
          key,
          name: agentName,
          concurrency,
          version: 'dev'
        }
      } as AgentToServerMessage)
      console.log(`judge agent ${key} connected`)
    })

    socket.addEventListener('message', (event) => {
      const message = parseWorkerMessage(event.data)
      if (!message) return
      if (message.type === 'ping') {
        send(socket, { type: 'pong', activeJobs: activeJobs.size })
        return
      }
      void runJob(socket, message)
    })

    socket.addEventListener('error', () => {
      if (!opened) resolve()
    })

    socket.addEventListener('close', () => {
      console.log(`judge agent ${key} disconnected`)
      resolve()
    })
  })
}

async function runJob(socket: WebSocket, message: Extract<ServerToAgentMessage, { type: 'run' }>) {
  activeJobs.add(message.jobId)
  const controller = new AbortController()
  jobControllers.set(message.jobId, controller)
  try {
    const result = await runPackage(message.payload, controller.signal, (progress) =>
      send(socket, {
        type: 'progress',
        jobId: message.jobId,
        progress
      })
    )
    send(socket, {
      type: 'result',
      jobId: message.jobId,
      result
    })
  } catch (error) {
    send(socket, {
      type: 'result',
      jobId: message.jobId,
      result: {
        status: 'SE',
        timeMs: 0,
        memoryBytes: 0,
        message: error instanceof Error ? error.message : String(error),
        cases: []
      }
    })
  } finally {
    activeJobs.delete(message.jobId)
    jobControllers.delete(message.jobId)
  }
}

async function runPackage(
  payload: JudgeAgentPayload,
  signal: AbortSignal,
  onProgress: (progress: JudgeAgentProgress) => void
) {
  throwIfCancelled(signal)
  const bundle = await loadProblemBundle(payload, signal)
  const sourcePath = await writeRunSource(payload)
  throwIfCancelled(signal)

  const hasDockerfile = bundle.files.has('Dockerfile')
  const testCases = payload.cases.map((testCase) => ({
    caseNo: testCase.caseNo,
    inputPath: testCase.inputPath,
    answerPath: testCase.answerPath
  }))

  try {
    if (payload.judger.kind === 'custom') {
      if (!hasDockerfile) throw new Error('custom bundle is missing Dockerfile')
      const problemFiles: Record<string, Uint8Array> = {}
      for (const [path, bytes] of bundle.files) {
        if (!path.startsWith('data/')) problemFiles[path] = bytes
      }
      return await judgePackage(runner, {
        scopeId: `submission-${payload.submissionId}`,
        testerFiles: {
          Dockerfile: payload.submission.dockerfile,
          [payload.submission.source]: payload.submission.code
        },
        problemFiles,
        dataDir: join(bundle.dir, 'data'),
        sourcePath,
        testCases,
        limits: {
          timeMs: payload.limits.timeLimit,
          memoryBytes: payload.limits.memoryLimit,
          outputBytes: 64 * 1024 * 1024
        },
        problemCacheKey: payload.bundleHash,
        cacheSubmissionPackage,
        signal,
        onProgress
      })
    }

    if (!testCases.length) {
      throw new Error('problem has no test cases')
    }
    if (payload.judger.kind === 'prebuilt') {
      return await runPrebuiltJudge(payload, join(bundle.dir, 'data'), sourcePath, signal, onProgress)
    }
    throw new Error('unsupported judger spec')
  } finally {
    await rm(dirname(sourcePath), { recursive: true, force: true }).catch(() => {})
  }
}

function buildWorkerUrl() {
  return workerUrl
}

function buildAgentWsUrl(server: string) {
  const url = new URL('/api/agents/connect', server)
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
  return url.toString()
}

function send(socket: WebSocket, message: AgentToServerMessage) {
  if (socket.readyState !== WebSocket.OPEN) return
  socket.send(JSON.stringify(message))
}

function parseWorkerMessage(raw: unknown): ServerToAgentMessage | null {
  try {
    const text = typeof raw === 'string' ? raw : Buffer.from(raw as ArrayBuffer).toString('utf8')
    return JSON.parse(text) as ServerToAgentMessage
  } catch {
    return null
  }
}

function redactToken(url: string) {
  const parsed = new URL(url)
  if (parsed.searchParams.has('token')) parsed.searchParams.set('token', '***')
  return parsed.toString()
}

function throwIfCancelled(signal: AbortSignal) {
  if (signal.aborted) throw new Error('judge job cancelled')
}

function createAgentKey(name: string) {
  return createHash('sha256')
    .update(name)
    .update(hostname())
    .update(String(Date.now()))
    .update(randomBytes(16))
    .digest('hex')
    .slice(0, 16)
}

async function loadProblemBundle(payload: JudgeAgentPayload, signal: AbortSignal) {
  const dir = bundleCachePath(payload.bundleHash)
  const cached = await readExtractedBundle(dir).catch(() => null)
  if (cached && bundleHasRequiredFiles(cached, payload)) return { dir, files: cached }

  const bytes = await fetchProblemBundle(payload.problemId, payload.bundleHash, signal)
  const files = parseTar(bytes)
  await rm(dir, { recursive: true, force: true })
  await writeExtractedBundle(dir, files)
  return { dir, files }
}

function bundleCachePath(bundleHash: string) {
  if (!/^[a-f0-9]{64}$/.test(bundleHash)) throw new Error(`invalid bundle hash: ${bundleHash}`)
  return join(bundleCacheDir, bundleHash)
}

async function fetchProblemBundle(problemId: number, expectedHash: string, signal: AbortSignal) {
  const response = await fetch(new URL(`/api/agents/bundle/${problemId}`, serverUrl), {
    signal,
    headers: {
      authorization: `Bearer ${token}`
    }
  })
  if (!response.ok) {
    throw new Error(`bundle download failed: ${response.status} ${await response.text()}`)
  }
  const actualHash = response.headers.get('x-bundle-hash')
  if (actualHash !== expectedHash) {
    throw new Error(`bundle hash mismatch: expected ${expectedHash}, got ${actualHash ?? 'missing'}`)
  }
  return new Uint8Array(await response.arrayBuffer())
}

function parseTar(bytes: Uint8Array) {
  const files = new Map<string, Uint8Array>()
  let offset = 0
  while (offset + 512 <= bytes.length) {
    const header = bytes.subarray(offset, offset + 512)
    if (header.every((value) => value === 0)) break
    const name = readTarString(header.subarray(0, 100))
    const sizeText = readTarString(header.subarray(124, 136)).trim()
    const size = Number.parseInt(sizeText || '0', 8)
    const bodyStart = offset + 512
    const bodyEnd = bodyStart + size
    if (name) files.set(name, bytes.slice(bodyStart, bodyEnd))
    offset = bodyStart + Math.ceil(size / 512) * 512
  }
  return files
}

async function writeExtractedBundle(dir: string, files: Map<string, Uint8Array>) {
  for (const [path, bytes] of files) {
    const target = join(dir, path)
    await mkdir(dirname(target), { recursive: true })
    await writeFile(target, bytes)
  }
}

async function readExtractedBundle(dir: string) {
  const files = new Map<string, Uint8Array>()
  await walkBundleDir(dir, '', files)
  return files
}

async function walkBundleDir(root: string, prefix: string, files: Map<string, Uint8Array>) {
  const entries = await readdir(join(root, prefix), { withFileTypes: true })
  for (const entry of entries) {
    const path = prefix ? `${prefix}/${entry.name}` : entry.name
    if (entry.isDirectory()) {
      await walkBundleDir(root, path, files)
    } else if (entry.isFile()) {
      files.set(path, new Uint8Array(await readFile(join(root, path))))
    }
  }
}

function bundleHasRequiredFiles(files: Map<string, Uint8Array>, payload: JudgeAgentPayload) {
  if (payload.judger.kind === 'custom' && !files.has('Dockerfile')) return false
  return payload.cases.every((testCase) => files.has(testCase.inputPath) && files.has(testCase.answerPath))
}

function readTarString(bytes: Uint8Array) {
  const end = bytes.indexOf(0)
  return new TextDecoder().decode(end >= 0 ? bytes.subarray(0, end) : bytes)
}

async function writeRunSource(payload: JudgeAgentPayload) {
  const dir = join(runRootDir, `${payload.submissionId}-${randomUUID()}`)
  await mkdir(dir, { recursive: true })
  const path = join(dir, payload.submission.source)
  await writeFile(path, payload.submission.code)
  return path
}

async function runPrebuiltJudge(
  payload: JudgeAgentPayload,
  dataDir: string,
  sourcePath: string,
  signal: AbortSignal,
  onProgress: (progress: JudgeAgentProgress) => void
): Promise<JudgeAgentResult> {
  const scopeId = `submission-${payload.submissionId}`
  const testerScope = `${scopeId}-b`
  await onProgress({ phase: 'building-b', message: 'Building submission package' })
  const testerBuild = await runner.buildPackage({
    scopeId: testerScope,
    files: {
      Dockerfile: payload.submission.dockerfile,
      [payload.submission.source]: payload.submission.code
    },
    limits: {
      timeMs: payload.limits.timeLimit,
      memoryBytes: payload.limits.memoryLimit
    },
    trusted: false,
    cacheKey: cacheSubmissionPackage
      ? hashFiles({
          Dockerfile: payload.submission.dockerfile,
          [payload.submission.source]: payload.submission.code
        })
      : undefined,
    signal
  })
  if (!testerBuild.ok || !testerBuild.imageId) {
    return {
      status: 'CE',
      timeMs: 0,
      memoryBytes: 0,
      message: testerBuild.logs,
      cases: []
    }
  }

  const cases: JudgeAgentCaseResult[] = []
  let timeMs = 0
  let memoryBytes = 0
  try {
    for (const [index, testCase] of payload.cases.entries()) {
      throwIfCancelled(signal)
      await onProgress({
        phase: 'running',
        message: `Testing case ${index + 1}/${payload.cases.length}`,
        caseNo: testCase.caseNo
      })
      const result = await runner.duel({
        scopeId: `${scopeId}-case-${index}`,
        judgeImageId: payload.judger.kind === 'prebuilt' ? payload.judger.image : '',
        testerImageId: testerBuild.imageId,
        dataDir,
        sourcePath,
        limits: {
          timeMs: payload.limits.timeLimit,
          memoryBytes: payload.limits.memoryLimit,
          outputBytes: 64 * 1024 * 1024
        },
        judgeEnv: {
          CHECK: payload.judger.kind === 'prebuilt' ? payload.judger.check : 'trim',
          CASE_NO: String(testCase.caseNo),
          INPUT: `/data/${testCase.inputPath.replace(/^data\//, '')}`,
          OUT: `/data/${testCase.answerPath.replace(/^data\//, '')}`,
          SOURCE: '/submission/source',
          TIME_LIMIT_MS: String(payload.limits.timeLimit),
          MEMORY_LIMIT_BYTES: String(payload.limits.memoryLimit)
        },
        signal
      })
      timeMs = Math.max(timeMs, result.timeMs)
      memoryBytes = Math.max(memoryBytes, result.memoryBytes)
      cases.push({
        caseNo: testCase.caseNo,
        status: result.status,
        timeMs: result.timeMs,
        memoryBytes: result.memoryBytes,
        message: result.message
      })
      await onProgress({
        phase: 'running',
        message: `Finished case ${index + 1}/${payload.cases.length}`,
        caseNo: testCase.caseNo,
        status: result.status,
        timeMs: result.timeMs,
        memoryBytes: result.memoryBytes
      })
    }
    return {
      status: cases.find((item) => item.status !== 'AC')?.status ?? ('AC' as JudgeStatus),
      timeMs,
      memoryBytes,
      message: cases.find((item) => item.status !== 'AC')?.message ?? 'accepted',
      cases
    }
  } finally {
    await runner.cleanup({ scopeId: testerScope })
  }
}

function hashFiles(files: Record<string, string | Uint8Array>) {
  const hash = createHash('sha256')
  for (const name of Object.keys(files).sort()) {
    hash.update(name)
    hash.update('\0')
    hash.update(typeof files[name] === 'string' ? files[name] : Buffer.from(files[name]))
    hash.update('\0')
  }
  return hash.digest('hex')
}
