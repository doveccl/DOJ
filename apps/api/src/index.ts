import { Hono } from 'hono'
import { logger } from 'hono/logger'
import { desc, eq } from 'drizzle-orm'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { enqueueJudgeTask } from '@doj/db/queue'
import { config } from './config'

const app = new Hono()

app.use('*', logger())

app.get('/health', (c) =>
  c.json({
    ok: true,
    service: 'doj-api'
  })
)

app.get('/api/config', (c) =>
  c.json({
    registration: true,
    aiCoachingEnabled: config.aiCoachingEnabled
  })
)

app.get('/api/problems', async (c) => {
  const list = await db.select().from(schema.problems).limit(50)
  return c.json({ total: list.length, list })
})

app.get('/api/submissions', async (c) => {
  const list = await db
    .select()
    .from(schema.submissions)
    .orderBy(desc(schema.submissions.createdAt))
    .limit(50)

  return c.json({ total: list.length, list })
})

const submitSchema = z.object({
  userId: z.string().uuid(),
  problemId: z.string().uuid(),
  problemVersionId: z.string().uuid(),
  languageId: z.string().min(1).max(64),
  sourceCode: z.string().min(1).max(200 * 1024),
  contestId: z.string().uuid().optional(),
  assignmentId: z.string().uuid().optional()
})

app.post('/api/submissions', async (c) => {
  const body = submitSchema.parse(await c.req.json())
  const [submission] = await db
    .insert(schema.submissions)
    .values({
      ...body,
      contestId: body.contestId ?? null,
      assignmentId: body.assignmentId ?? null
    })
    .returning()

  await enqueueJudgeTask(submission.id)
  return c.json(submission, 201)
})

app.get('/api/submissions/:id', async (c) => {
  const id = c.req.param('id')
  const [submission] = await db
    .select()
    .from(schema.submissions)
    .where(eq(schema.submissions.id, id))
    .limit(1)

  if (!submission) return c.notFound()
  return c.json(submission)
})

Bun.serve({
  port: config.port,
  fetch: app.fetch
})

console.log(`DOJ API listening on http://localhost:${config.port}`)
