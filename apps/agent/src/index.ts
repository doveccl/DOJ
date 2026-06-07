import type {
  AgentToWorkerMessage,
  JudgeAgentPayload,
  JudgeAgentProgress,
  WorkerToAgentMessage
} from '@doj/shared/agent'
import { buildCasesFromPackageData } from '@doj/shared/testdata'
import { DockerRunner } from '@doj/runner/docker-runner'
import { judgePackage } from '@doj/runner/judge'
import { PackageFileCache } from './package-cache'

const key = process.env.DOJ_AGENT_KEY ?? 'local-agent'
const token = process.env.DOJ_AGENT_TOKEN ?? 'local-agent-token'
const name = process.env.DOJ_AGENT_NAME ?? 'Local Agent'
const concurrency = Number(process.env.DOJ_AGENT_CONCURRENCY ?? 2)
const labels = (process.env.DOJ_AGENT_LABELS ?? 'local')
  .split(',')
  .map((label) => label.trim())
  .filter(Boolean)
const workerUrl = process.env.DOJ_WORKER_WS_URL ?? 'ws://localhost:7975/agents/connect'
const runner = new DockerRunner()
const packageFileCache = new PackageFileCache({
  maxBytes: Number(process.env.DOJ_AGENT_PACKAGE_CACHE_BYTES ?? 512 * 1024 * 1024)
})
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
    const socket = new WebSocket(url)
    let opened = false

    socket.addEventListener('open', () => {
      opened = true
      send(socket, {
        type: 'hello',
        info: {
          key,
          name,
          concurrency,
          labels,
          version: 'dev'
        }
      })
      console.log(`judge agent ${key} connected`)
    })

    socket.addEventListener('message', (event) => {
      const message = parseWorkerMessage(event.data)
      if (!message) return
      if (message.type === 'ping') {
        send(socket, { type: 'pong', activeJobs: activeJobs.size })
        return
      }
      if (message.type === 'cancel') {
        jobControllers.get(message.jobId)?.abort(message.reason)
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

async function runJob(socket: WebSocket, message: Extract<WorkerToAgentMessage, { type: 'run' }>) {
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
      type: 'error',
      jobId: message.jobId,
      message: error instanceof Error ? error.message : String(error)
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
  // Fetch problem package files through an agent-local cache keyed by immutable
  // S3 object references, so repeated submissions for one problem avoid S3 I/O.
  const fetched = await Promise.all(
    payload.problemFiles.map(async (file) => ({
      path: file.path,
      bytes: await packageFileCache.get(file)
    }))
  )
  throwIfCancelled(signal)

  const hasDockerfile = fetched.some((file) => file.path === 'Dockerfile')

  if (hasDockerfile) {
    // Custom mode: A is the problem package (interactor + checker).
    const problemFiles: Record<string, Uint8Array> = {}
    for (const file of fetched) problemFiles[file.path] = file.bytes
    return judgePackage(runner, {
      scopeId: payload.scopeId,
      testerFiles: payload.testerFiles,
      problemFiles,
      testCases: payload.inlineTestCases,
      caseCount: payload.caseCount || payload.inlineTestCases.length || 1,
      limits: payload.limits,
      code: payload.code,
      signal,
      onProgress
    })
  }

  // Default mode: derive cases from packaged `data/` files, else inline cases.
  const dataFiles = fetched.filter((file) => file.path.startsWith('data/'))
  const dataCases = dataFiles.length ? buildCasesFromPackageData(dataFiles) : []
  const testCases = dataCases.length ? dataCases : payload.inlineTestCases
  if (!testCases.length) {
    throw new Error('problem has no test cases (no data files and no inline cases)')
  }
  return judgePackage(runner, {
    scopeId: payload.scopeId,
    testerFiles: payload.testerFiles,
    problemFiles: null,
    testCases,
    limits: payload.limits,
    code: payload.code,
    signal,
    onProgress
  })
}

function buildWorkerUrl() {
  const url = new URL(workerUrl)
  url.searchParams.set('key', key)
  url.searchParams.set('token', token)
  return url.toString()
}

function send(socket: WebSocket, message: AgentToWorkerMessage) {
  if (socket.readyState !== WebSocket.OPEN) return
  socket.send(JSON.stringify(message))
}

function parseWorkerMessage(raw: unknown): WorkerToAgentMessage | null {
  try {
    const text = typeof raw === 'string' ? raw : Buffer.from(raw as ArrayBuffer).toString('utf8')
    return JSON.parse(text) as WorkerToAgentMessage
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
