import type {
  AgentToWorkerMessage,
  JudgeAgentPayload,
  WorkerToAgentMessage
} from '@doj/shared/agent'
import { getObjectBytes } from '@doj/shared/storage'
import { parseZipTestCases } from '@doj/shared/testdata'
import { DockerRunner } from '@doj/runner/docker-runner'
import { judgePayload } from '@doj/runner/judge'

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
const activeJobs = new Set<string>()

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
  try {
    const payload = await hydratePayload(message.payload)
    const result = await judgePayload(runner, payload)
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
  }
}

async function hydratePayload(payload: JudgeAgentPayload): Promise<JudgeAgentPayload> {
  if (!payload.testdataFile) return payload

  const bytes = await getObjectBytes(payload.testdataFile.objectKey, payload.testdataFile.bucket)
  const testCases = parseZipTestCases(bytes)
  if (!testCases.length) {
    throw new Error(`testdata file has no cases: ${payload.testdataFile.filename}`)
  }

  return {
    ...payload,
    testCases
  }
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
