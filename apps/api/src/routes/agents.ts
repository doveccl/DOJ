import { Hono } from 'hono'
import type { Server, ServerWebSocket, WebSocketHandler } from 'bun'
import { createHash } from 'node:crypto'
import tar from 'tar-stream'
import { eq } from 'drizzle-orm'
import type { JudgeAgentPayload, JudgeAgentResult } from '@doj/shared/agent'
import type { JudgeProgress } from '@doj/shared/judge'
import { db, schema } from '@doj/db/client'
import { getObjectBytes, listObjects } from '@doj/shared/storage'
import { upsertRuntimeAgent } from './admin-agents'
import { numericId } from '../validation'

export interface AgentSocketData {
  kind: 'agent'
  key: string | null
  name: string
  concurrency: number
  activeJobs: number
  version: string
  connectedAt: string
  authorized: boolean
  lastPongAt: number
}

interface AgentHelloMessage {
  type: 'hello'
  info: {
    key: string
    name: string
    concurrency: number
    version?: string
  }
}

interface AgentPongMessage {
  type: 'pong'
  activeJobs: number
}

interface AgentResultMessage {
  type: 'result'
  jobId: string
  result: JudgeAgentResult
}

interface AgentProgressMessage {
  type: 'progress'
  jobId: string
  progress: JudgeProgress
}

interface PendingJob {
  agentKey: string
  resolve: (result: JudgeAgentResult) => void
  reject: (error: Error) => void
  timeout: Timer
  onProgress?: (progress: JudgeProgress) => void | Promise<void>
}

const sockets = new Map<string, ServerWebSocket<AgentSocketData>>()
const pendingJobs = new Map<string, PendingJob>()
let heartbeatStarted = false

export class AgentJobRetryableError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'AgentJobRetryableError'
  }
}

export function registerAgentRoutes(app: Hono) {
  app.get('/api/agents/bundle/:problemId', async (c) => {
    const denied = requireAgentSecret(c.req.header('authorization'))
    if (denied) return c.json(denied.body, denied.status)

    const problemId = numericId.parse(c.req.param('problemId'))
    const entries = await loadProblemDataEntries(problemId)
    const pack = tar.pack()
    writeProblemData(entries, pack)

    c.header('content-type', 'application/x-tar')
    c.header('cache-control', 'private, max-age=60')
    c.header('x-bundle-hash', hashBundleEntries(entries))
    return c.body(pack as unknown as ReadableStream)
  })
}

export function handleAgentUpgrade(request: Request, server: Server<AgentSocketData>) {
  const denied = requireAgentSecret(request.headers.get('authorization') ?? undefined)
  if (denied) return new Response(JSON.stringify(denied.body), { status: denied.status })

  const upgraded = server.upgrade(request, {
    data: {
      kind: 'agent',
      key: null,
      name: 'agent',
      concurrency: 1,
      activeJobs: 0,
      version: 'unknown',
      connectedAt: new Date().toISOString(),
      authorized: true,
      lastPongAt: Date.now()
    }
  })
  return upgraded ? undefined : new Response('WebSocket upgrade failed', { status: 400 })
}

export const agentWebSocketHandlers: WebSocketHandler<AgentSocketData> = {
  open(ws) {
    startHeartbeat()
    ws.send(JSON.stringify({ type: 'ping' }))
  },
  message(ws, raw) {
    const message = parseAgentMessage(raw)
    if (!message) return

    if (message.type === 'hello') {
      const nextKey = message.info.key.trim()
      if (!nextKey) {
        ws.close(1008, 'missing agent key')
        return
      }
      const existing = sockets.get(nextKey)
      if (existing && existing !== ws) {
        ws.close(1008, 'duplicate agent key')
        return
      }

      if (ws.data.key && ws.data.key !== nextKey) sockets.delete(ws.data.key)
      ws.data.key = nextKey
      ws.data.name = message.info.name || nextKey
      ws.data.concurrency = Math.max(1, Math.floor(message.info.concurrency || 1))
      ws.data.version = message.info.version || 'unknown'
      ws.data.connectedAt = new Date().toISOString()
      sockets.set(nextKey, ws)
      publishAgent(ws, true)
      return
    }

    if (message.type === 'pong') {
      ws.data.activeJobs = Math.max(0, Math.floor(message.activeJobs || 0))
      ws.data.lastPongAt = Date.now()
      publishAgent(ws, true)
      return
    }

    if (message.type === 'progress') {
      void relayAgentProgress(ws, message)
      return
    }

    if (message.type === 'result') {
      settleAgentResult(ws, message)
    }
  },
  close(ws) {
    if (!ws.data.key) return
    sockets.delete(ws.data.key)
    for (const [jobId, pending] of pendingJobs) {
      if (pending.agentKey === ws.data.key) {
        clearTimeout(pending.timeout)
        pendingJobs.delete(jobId)
        pending.reject(new AgentJobRetryableError(`agent ${ws.data.key} disconnected`))
      }
    }
    publishAgent(ws, false)
  }
}

export async function dispatchJudgeToAgent(
  payload: JudgeAgentPayload,
  options: { timeoutMs?: number; onProgress?: (progress: JudgeProgress) => void | Promise<void> } = {}
) {
  const selected = selectAgent()
  if (!selected?.data.key) return null

  const jobId = crypto.randomUUID()
  selected.data.activeJobs += 1
  publishAgent(selected, true)

  return await new Promise<JudgeAgentResult>((resolve, reject) => {
    const timeout = setTimeout(() => {
      pendingJobs.delete(jobId)
      selected.data.activeJobs = Math.max(0, selected.data.activeJobs - 1)
      publishAgent(selected, true)
      reject(new AgentJobRetryableError(`agent job ${jobId} exceeded server wait timeout`))
    }, options.timeoutMs ?? 0x7fffffff)

    pendingJobs.set(jobId, {
      agentKey: selected.data.key!,
      resolve,
      reject,
      timeout,
      onProgress: options.onProgress
    })
    selected.send(JSON.stringify({ type: 'run', jobId, payload }))
  })
}

export function hasAvailableAgent() {
  return Boolean(selectAgent())
}

export async function getProblemBundleInfo(problemId: number) {
  const entries = await loadProblemDataEntries(problemId)
  return {
    entries,
    bundleHash: hashBundleEntries(entries)
  }
}

function requireAgentSecret(header: string | undefined) {
  const expected = process.env.SECRET ?? 'local-dev-secret-change-me'
  const token = header?.startsWith('Bearer ') ? header.slice('Bearer '.length) : ''
  if (token !== expected) {
    return {
      status: 401 as const,
      body: { error: { code: 'UNAUTHORIZED', message: 'Invalid agent secret' } }
    }
  }
  return null
}

function startHeartbeat() {
  if (heartbeatStarted) return
  heartbeatStarted = true
  setInterval(() => {
    for (const ws of sockets.values()) {
      if (Date.now() - ws.data.lastPongAt > 30_000) {
        ws.close(1001, 'heartbeat timeout')
        continue
      }
      if (ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type: 'ping' }))
    }
  }, 10_000)
}

function publishAgent(ws: ServerWebSocket<AgentSocketData>, online: boolean) {
  if (!ws.data.key) return
  upsertRuntimeAgent({
    id: ws.data.key,
    name: ws.data.name,
    online,
    concurrency: ws.data.concurrency,
    running: ws.data.activeJobs,
      version: ws.data.version,
      connectedAt: ws.data.connectedAt,
    lastSeenAt: new Date().toISOString()
  })
}

function parseAgentMessage(
  raw: string | Buffer
): AgentHelloMessage | AgentPongMessage | AgentProgressMessage | AgentResultMessage | null {
  try {
    const text = typeof raw === 'string' ? raw : Buffer.from(raw).toString('utf8')
    const message = JSON.parse(text) as { type?: unknown }
    if (message.type === 'hello') return message as AgentHelloMessage
    if (message.type === 'pong') return message as AgentPongMessage
    if (message.type === 'progress') return message as AgentProgressMessage
    if (message.type === 'result') return message as AgentResultMessage
    return null
  } catch {
    return null
  }
}

function selectAgent() {
  return [...sockets.values()]
    .filter(
      (ws) =>
        ws.readyState === WebSocket.OPEN &&
        !!ws.data.key &&
        Date.now() - ws.data.lastPongAt <= 30_000 &&
        ws.data.activeJobs < ws.data.concurrency
    )
    .sort((left, right) => {
      const leftLoad = left.data.activeJobs / left.data.concurrency
      const rightLoad = right.data.activeJobs / right.data.concurrency
      if (leftLoad !== rightLoad) return leftLoad - rightLoad
      if (left.data.lastPongAt !== right.data.lastPongAt) return left.data.lastPongAt - right.data.lastPongAt
      return (left.data.key ?? '').localeCompare(right.data.key ?? '')
    })[0]
}

async function relayAgentProgress(
  ws: ServerWebSocket<AgentSocketData>,
  message: AgentProgressMessage
) {
  const pending = pendingJobs.get(message.jobId)
  if (!pending || pending.agentKey !== ws.data.key) return
  await pending.onProgress?.(message.progress)
}

function settleAgentResult(ws: ServerWebSocket<AgentSocketData>, message: AgentResultMessage) {
  const pending = pendingJobs.get(message.jobId)
  if (!pending || pending.agentKey !== ws.data.key) return
  clearTimeout(pending.timeout)
  pendingJobs.delete(message.jobId)
  ws.data.activeJobs = Math.max(0, ws.data.activeJobs - 1)
  publishAgent(ws, true)
  pending.resolve(message.result)
}

export async function loadProblemDataEntries(problemId: number) {
  const [problem] = await db
    .select({ mode: schema.problems.mode })
    .from(schema.problems)
    .where(eq(schema.problems.id, problemId))
    .limit(1)
  if (!problem) throw new Error(`problem not found: ${problemId}`)

  const prefix = `problems/${problemId}/`
  const objects = await listObjects(prefix)
  const paths = objects
    .map((object) => normalizeBundlePath(object.key.slice(prefix.length)))
    .filter((path): path is string => Boolean(path))
    .filter((path) => path.startsWith('data/') || (problem.mode === 'custom' && isCustomJudgeResource(path)))
    .sort((left, right) => left.localeCompare(right))

  validateDataPairs(paths, problemId)
  if (problem.mode === 'custom' && !paths.includes('Dockerfile')) {
    throw new Error(`custom problem ${problemId} is missing Dockerfile`)
  }

  const entries = await Promise.all(
    paths.map(async (path) => ({
      path,
      bytes: await getObjectBytes(`${prefix}${path}`)
    }))
  )
  return entries
}

function writeProblemData(entries: Array<{ path: string; bytes: Uint8Array }>, pack: tar.Pack) {
  try {
    for (const entry of entries) pack.entry({ name: entry.path }, Buffer.from(entry.bytes))
  } finally {
    pack.finalize()
  }
}

export function hashBundleEntries(entries: Array<{ path: string; bytes: Uint8Array }>) {
  const hash = createHash('sha256')
  for (const entry of [...entries].sort((left, right) => left.path.localeCompare(right.path))) {
    const contentHash = createHash('sha256').update(entry.bytes).digest('hex')
    hash.update(entry.path)
    hash.update('\n')
    hash.update(String(entry.bytes.byteLength))
    hash.update('\n')
    hash.update(contentHash)
    hash.update('\n')
  }
  return hash.digest('hex')
}

function normalizeBundlePath(path: string) {
  if (
    !path ||
    path === 'statement.md' ||
    path === 'assets' ||
    path === 'data' ||
    path.startsWith('assets/') ||
    path.startsWith('/') ||
    path.includes('\\') ||
    hasControlCharacter(path) ||
    path.split('/').some((part) => !part || part === '.' || part === '..')
  ) {
    return null
  }
  return path
}

function isCustomJudgeResource(path: string) {
  return !path.startsWith('data/') && !path.startsWith('assets/') && path !== 'statement.md'
}

function hasControlCharacter(value: string) {
  return Array.from(value).some((char) => char.charCodeAt(0) <= 0x1f)
}

function validateDataPairs(paths: string[], problemId: number) {
  const dataFiles = paths.filter((path) => path.startsWith('data/'))
  const inputs = new Set(dataFiles.flatMap((path) => readCaseKey(path, 'input') ?? []))
  const answers = new Set(dataFiles.flatMap((path) => readCaseKey(path, 'answer') ?? []))
  if (!inputs.size && !answers.size) throw new Error(`problem ${problemId} has no data cases`)
  for (const key of inputs) {
    if (!answers.has(key)) throw new Error(`problem ${problemId} is missing answer for data case ${key}`)
  }
  for (const key of answers) {
    if (!inputs.has(key)) throw new Error(`problem ${problemId} is missing input for data case ${key}`)
  }
}

function readCaseKey(path: string, kind: 'input' | 'answer') {
  const name = path.replace(/^.*\//, '').toLowerCase()
  const stem = name.replace(/\.[^.]+$/, '')
  const extension = name.includes('.') ? name.replace(/^.*\./, '.') : ''
  const matches =
    kind === 'input'
      ? extension === '.in' || /input|(^|[^a-z])in([^a-z]|$)/.test(stem)
      : extension === '.out' ||
        extension === '.ans' ||
        /output|answer|ans|(^|[^a-z])out([^a-z]|$)/.test(stem)
  if (!matches) return null
  return stem.match(/\d+/g)?.at(-1) ?? stem
}
