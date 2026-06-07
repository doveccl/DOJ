import type { Server, ServerWebSocket } from 'bun'
import { eq } from 'drizzle-orm'
import { db, schema } from '@doj/db/client'
import type {
  AgentToWorkerMessage,
  JudgeAgentPayload,
  JudgeAgentProgress,
  JudgeAgentResult,
  WorkerToAgentMessage
} from '@doj/shared/agent'

export interface AgentSocketData {
  key: string
  id: number
}

export interface ConnectedJudgeAgent {
  key: string
  id: number
  socket: ServerWebSocket<AgentSocketData>
  name: string
  labels: string[]
  concurrency: number
  activeJobs: number
  connectedAt: number
  lastTouchedAt: number
}

interface PendingJob {
  agentKey: string
  resolve: (result: JudgeAgentResult) => void
  reject: (error: Error) => void
  timeout: Timer
  onProgress?: (progress: JudgeAgentProgress) => void | Promise<void>
}

export interface JudgeAgentServerOptions {
  port: number
  hostname?: string
  jobTimeoutMs: number
  onWake: () => void
}

export class JudgeAgentServer {
  private server: Server<AgentSocketData> | null = null
  private readonly agents = new Map<string, ConnectedJudgeAgent>()
  private readonly pendingJobs = new Map<string, PendingJob>()
  private readonly touchIntervalMs = 30_000

  constructor(private readonly options: JudgeAgentServerOptions) {}

  start() {
    if (this.server) return this.server

    this.server = Bun.serve<AgentSocketData>({
      port: this.options.port,
      hostname: this.options.hostname,
      fetch: async (request, server) => {
        const url = new URL(request.url)
        if (url.pathname === '/health') {
          return Response.json({
            ok: true,
            service: 'doj-worker',
            agents: this.snapshot()
          })
        }
        if (url.pathname !== '/agents/connect') {
          return new Response('Not Found', { status: 404 })
        }

        const credentials = readAgentCredentials(request)
        if (!credentials.key || !credentials.token) {
          return new Response('Missing agent credentials', { status: 401 })
        }

        const agent = await authenticateAgent(credentials.key, credentials.token)
        if (!agent) return new Response('Invalid agent credentials', { status: 403 })

        if (
          server.upgrade(request, {
            data: {
              key: agent.key,
              id: agent.id
            }
          })
        ) {
          return undefined
        }

        return new Response('WebSocket upgrade failed', { status: 400 })
      },
      websocket: {
        open: (socket) => {
          void this.register(socket)
        },
        message: (socket, message) => {
          void this.handleMessage(socket, message)
        },
        close: (socket) => {
          this.unregister(socket)
        }
      }
    })

    return this.server
  }

  close() {
    for (const agent of this.agents.values()) {
      agent.socket.close()
    }
    this.server?.stop()
    this.server = null
  }

  pickAvailableAgent() {
    const available = [...this.agents.values()]
      .filter((agent) => agent.activeJobs < agent.concurrency)
      .sort(
        (left, right) => left.activeJobs - right.activeJobs || left.connectedAt - right.connectedAt
      )

    return available[0] ?? null
  }

  reserveAvailableAgent() {
    const agent = this.pickAvailableAgent()
    if (!agent) return null
    agent.activeJobs += 1
    return agent
  }

  releaseAgent(agent: ConnectedJudgeAgent) {
    const current = this.agents.get(agent.key)
    if (!current) return
    current.activeJobs = Math.max(0, current.activeJobs - 1)
    this.options.onWake()
  }

  hasAvailableAgent() {
    return this.pickAvailableAgent() !== null
  }

  snapshot() {
    return [...this.agents.values()].map((agent) => ({
      key: agent.key,
      name: agent.name,
      labels: agent.labels,
      concurrency: agent.concurrency,
      activeJobs: agent.activeJobs,
      connectedAt: new Date(agent.connectedAt).toISOString()
    }))
  }

  runJob(
    agent: ConnectedJudgeAgent,
    payload: JudgeAgentPayload,
    options: { onProgress?: (progress: JudgeAgentProgress) => void | Promise<void> } = {}
  ) {
    const jobId = crypto.randomUUID()
    const message: WorkerToAgentMessage = {
      type: 'run',
      jobId,
      payload
    }

    const promise = new Promise<JudgeAgentResult>((resolve, reject) => {
      const timeout = setTimeout(() => {
        this.sendCancel(agent, jobId, 'worker job timeout')
        this.pendingJobs.delete(jobId)
        reject(new Error(`judge agent ${agent.key} timed out on job ${jobId}`))
      }, this.options.jobTimeoutMs)

      this.pendingJobs.set(jobId, {
        agentKey: agent.key,
        resolve,
        reject,
        timeout,
        onProgress: options.onProgress
      })

      try {
        agent.socket.send(JSON.stringify(message))
      } catch (error) {
        this.pendingJobs.delete(jobId)
        clearTimeout(timeout)
        reject(error instanceof Error ? error : new Error(String(error)))
      }
    })

    return promise.finally(() => {
      const current = this.agents.get(agent.key)
      if (current) current.activeJobs = Math.max(0, current.activeJobs - 1)
      this.options.onWake()
    })
  }

  private async register(socket: ServerWebSocket<AgentSocketData>) {
    const [row] = await db
      .select()
      .from(schema.judgeAgents)
      .where(eq(schema.judgeAgents.id, socket.data.id))
      .limit(1)

    if (!row?.enabled) {
      socket.close(1008, 'agent disabled')
      return
    }

    const existing = this.agents.get(row.key)
    if (existing) {
      this.rejectAgentJobs(row.key, 'agent reconnected')
      existing.socket.close(1012, 'agent reconnected')
    }

    const connected: ConnectedJudgeAgent = {
      key: row.key,
      id: row.id,
      socket,
      name: row.name,
      labels: row.labels,
      concurrency: row.concurrency,
      activeJobs: 0,
      connectedAt: Date.now(),
      lastTouchedAt: 0
    }
    this.agents.set(row.key, connected)
    await this.touchAgent(connected, true)
    this.options.onWake()
    console.log(`judge agent connected: ${row.key}`)
  }

  private unregister(socket: ServerWebSocket<AgentSocketData>) {
    const agent = this.agents.get(socket.data.key)
    if (agent?.socket !== socket) return
    this.agents.delete(agent.key)
    this.rejectAgentJobs(agent.key, 'agent disconnected')
    this.options.onWake()
    console.log(`judge agent disconnected: ${agent.key}`)
  }

  private async handleMessage(socket: ServerWebSocket<AgentSocketData>, raw: string | Buffer) {
    const message = parseAgentMessage(raw)
    const agent = this.agents.get(socket.data.key)
    if (!agent || !message) return
    await this.touchAgent(agent)

    if (message.type === 'hello') {
      agent.name = message.info.name || agent.name
      agent.labels = message.info.labels
      agent.concurrency = Math.max(1, Math.min(agent.concurrency, message.info.concurrency || 1))
      return
    }

    if (message.type === 'pong') return

    const pending = this.pendingJobs.get(message.jobId)
    if (!pending) return
    if (pending.agentKey !== agent.key) {
      this.pendingJobs.delete(message.jobId)
      clearTimeout(pending.timeout)
      pending.reject(new Error(`job ${message.jobId} was answered by the wrong agent`))
      return
    }

    if (message.type === 'progress') {
      await pending.onProgress?.(message.progress)
      return
    }

    this.pendingJobs.delete(message.jobId)
    clearTimeout(pending.timeout)
    if (message.type === 'result') {
      pending.resolve(message.result)
    } else {
      pending.reject(new Error(message.message))
    }
  }

  private async touchAgent(agent: ConnectedJudgeAgent, force = false) {
    const now = Date.now()
    if (!force && now - agent.lastTouchedAt < this.touchIntervalMs) return
    agent.lastTouchedAt = now
    await db
      .update(schema.judgeAgents)
      .set({ lastSeenAt: new Date(), updatedAt: new Date() })
      .where(eq(schema.judgeAgents.id, agent.id))
      .catch(() => {})
  }

  private rejectAgentJobs(agentKey: string, reason: string) {
    for (const [jobId, job] of this.pendingJobs) {
      if (job.agentKey !== agentKey) continue
      this.pendingJobs.delete(jobId)
      clearTimeout(job.timeout)
      job.reject(new Error(`${agentKey}: ${reason}`))
    }
  }

  private sendCancel(agent: ConnectedJudgeAgent, jobId: string, reason: string) {
    const message: WorkerToAgentMessage = { type: 'cancel', jobId, reason }
    try {
      agent.socket.send(JSON.stringify(message))
    } catch {
      // The timeout path already rejects the job; a failed cancel just means the
      // agent is gone and unregister/reconnect cleanup will handle the slot.
    }
  }
}

async function authenticateAgent(key: string, token: string) {
  const [agent] = await db
    .select()
    .from(schema.judgeAgents)
    .where(eq(schema.judgeAgents.key, key))
    .limit(1)
  if (!agent?.enabled) return null
  if (!(await Bun.password.verify(token, agent.tokenHash))) return null
  return agent
}

function readAgentCredentials(request: Request) {
  const url = new URL(request.url)
  let key = url.searchParams.get('key') ?? ''
  let token = url.searchParams.get('token') ?? ''
  const authorization = request.headers.get('authorization')

  if ((!key || !token) && authorization?.startsWith('Basic ')) {
    const decoded = Buffer.from(authorization.slice('Basic '.length), 'base64').toString('utf8')
    const separator = decoded.indexOf(':')
    if (separator >= 0) {
      key ||= decoded.slice(0, separator)
      token ||= decoded.slice(separator + 1)
    }
  }

  if (!token && authorization?.startsWith('Bearer ')) {
    token = authorization.slice('Bearer '.length)
  }

  return { key, token }
}

function parseAgentMessage(raw: string | Buffer): AgentToWorkerMessage | null {
  try {
    const text = typeof raw === 'string' ? raw : raw.toString('utf8')
    return JSON.parse(text) as AgentToWorkerMessage
  } catch {
    return null
  }
}
