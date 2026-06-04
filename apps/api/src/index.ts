import { Hono } from 'hono'
import { logger } from 'hono/logger'
import { desc, eq } from 'drizzle-orm'
import { z, ZodError } from 'zod'
import { db, schema } from '@doj/db/client'
import { enqueueJudgeTask } from '@doj/db/queue'
import { config } from './config'

const app = new Hono()

app.use('*', logger())

app.onError((error, c) => {
  if (error instanceof ZodError) {
    return c.json(
      {
        code: 'BAD_REQUEST',
        message: 'Invalid request payload',
        issues: error.issues
      },
      400
    )
  }

  console.error(error)
  return c.json(
    {
      code: 'INTERNAL_SERVER_ERROR',
      message: error instanceof Error ? error.message : String(error)
    },
    500
  )
})

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

const createProblemSchema = z.object({
  title: z.string().min(1).max(160),
  slug: z.string().min(1).max(160).optional(),
  tags: z.array(z.string()).default([]),
  statementMarkdown: z.string().min(1),
  timeLimitMs: z.number().int().positive().default(1000),
  memoryLimitBytes: z.number().int().positive().default(256 * 1024 * 1024),
  outputLimitBytes: z.number().int().positive().default(64 * 1024 * 1024)
})

app.post('/api/problems', async (c) => {
  const body = createProblemSchema.parse(await c.req.json())

  const result = await db.transaction(async (tx) => {
    const [problem] = await tx
      .insert(schema.problems)
      .values({
        title: body.title,
        slug: body.slug ?? null,
        tags: body.tags
      })
      .returning()

    const [version] = await tx
      .insert(schema.problemVersions)
      .values({
        problemId: problem.id,
        version: 1,
        statementMarkdown: body.statementMarkdown,
        timeLimitMs: body.timeLimitMs,
        memoryLimitBytes: body.memoryLimitBytes,
        outputLimitBytes: body.outputLimitBytes
      })
      .returning()

    return { problem, version }
  })

  return c.json(result, 201)
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
