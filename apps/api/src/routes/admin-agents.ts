import { Hono } from 'hono'
import { authMiddleware, requireAdmin } from '../auth'

interface RuntimeAgent {
  id: string
  name: string
  online: boolean
  concurrency: number
  running: number
  version?: string
  connectedAt?: string
  lastSeenAt: string | null
}

const runtimeAgents = new Map<string, RuntimeAgent>()

export function upsertRuntimeAgent(agent: RuntimeAgent) {
  runtimeAgents.set(agent.id, agent)
}

export function registerAdminAgentRoutes(app: Hono) {
  app.get('/api/admin/agents', authMiddleware, async (c) => {
    const denied = await requireAdmin(c)
    if (denied) return denied

    const items = [...runtimeAgents.values()]
      .sort((a, b) => a.name.localeCompare(b.name))
      .map((agent) => ({
        key: agent.id,
        name: agent.name,
        concurrency: agent.concurrency,
        activeJobs: agent.running,
        version: agent.version ?? '',
        connectedAt: agent.connectedAt ?? agent.lastSeenAt ?? '',
        heartbeatAt: agent.lastSeenAt ?? ''
      }))
    return c.json({ items, page: 1, pageSize: items.length || 50, total: items.length })
  })

  app.get('/api/admin/agents/instructions', authMiddleware, async (c) => {
    const denied = await requireAdmin(c)
    if (denied) return denied

    return c.json({
      server: process.env.SERVER ?? 'http://127.0.0.1:7974',
      secretEnv: 'SECRET',
      command: 'SERVER=http://127.0.0.1:7974 SECRET=<secret> AGENT_NAME=agent-1 AGENT_CONCURRENCY=1 bun run --cwd apps/agent dev'
    })
  })
}
