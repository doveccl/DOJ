import { Hono } from 'hono'
import { asc, eq } from 'drizzle-orm'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { authMiddleware, requireGroup } from '../auth'
import { numericId } from '../validation'

export function registerAdminAgentRoutes(app: Hono) {
  app.get('/api/admin/agents', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const list = await db
      .select()
      .from(schema.judgeAgents)
      .orderBy(asc(schema.judgeAgents.sortOrder), asc(schema.judgeAgents.id))

    return c.json({
      total: list.length,
      list: list.map(({ tokenHash: _tokenHash, ...agent }) => agent)
    })
  })

  app.post('/api/admin/agents', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const body = agentConfigSchema.parse(await c.req.json())
    const token = body.token || createAgentToken()
    const tokenHash = await Bun.password.hash(token, {
      algorithm: 'argon2id',
      memoryCost: 19456,
      timeCost: 2
    })
    const [agent] = await db
      .insert(schema.judgeAgents)
      .values({
        key: body.key,
        name: body.name,
        enabled: body.enabled,
        tokenHash,
        labels: body.labels,
        concurrency: body.concurrency,
        sortOrder: body.sortOrder
      })
      .onConflictDoUpdate({
        target: schema.judgeAgents.key,
        set: {
          name: body.name,
          enabled: body.enabled,
          tokenHash,
          labels: body.labels,
          concurrency: body.concurrency,
          sortOrder: body.sortOrder,
          updatedAt: new Date()
        }
      })
      .returning()

    const { tokenHash: _tokenHash, ...payload } = agent
    return c.json({ ...payload, token }, 201)
  })

  app.post('/api/admin/agents/:id/rotate-token', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const token = createAgentToken()
    const tokenHash = await Bun.password.hash(token, {
      algorithm: 'argon2id',
      memoryCost: 19456,
      timeCost: 2
    })
    const [agent] = await db
      .update(schema.judgeAgents)
      .set({ tokenHash, updatedAt: new Date() })
      .where(eq(schema.judgeAgents.id, id))
      .returning()

    if (!agent) return c.notFound()
    return c.json({ id: agent.id, key: agent.key, token })
  })
}

const agentConfigSchema = z.object({
  key: z.string().regex(/^[a-z][a-z0-9_-]{1,63}$/),
  name: z.string().min(1).max(128),
  enabled: z.boolean().default(true),
  token: z.string().min(12).max(500).optional(),
  labels: z.array(z.string().min(1).max(64)).default([]),
  concurrency: z.number().int().positive().default(2),
  sortOrder: z.number().int().min(0).default(100)
})

function createAgentToken() {
  return Buffer.from(crypto.getRandomValues(new Uint8Array(32))).toString('base64url')
}
