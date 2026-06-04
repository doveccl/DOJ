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

app.post('/api/submissions/:id/coach', async (c) => {
  const id = c.req.param('id')
  const [submission] = await db
    .select()
    .from(schema.submissions)
    .where(eq(schema.submissions.id, id))
    .limit(1)

  if (!submission) return c.notFound()
  if (submission.contestId) {
    return c.json({ code: 'AI_DISABLED_IN_CONTEST', message: 'AI coaching is disabled in contests' }, 403)
  }
  if (['AC', 'WAITING', 'JUDGING', 'FROZEN'].includes(submission.status)) {
    return c.json(
      {
        code: 'AI_COACHING_UNAVAILABLE',
        message: `AI coaching is unavailable for ${submission.status} submissions`
      },
      400
    )
  }

  const [session] = await db
    .insert(schema.aiCoachingSessions)
    .values({
      userId: submission.userId,
      submissionId: submission.id,
      model: 'local-stub',
      promptVersion: 'non-ac-v1',
      responseMarkdown: createCoachingResponse(submission.status, submission.message),
      metadata: {
        status: submission.status,
        languageId: submission.languageId
      }
    })
    .returning()

  return c.json(session, 201)
})

Bun.serve({
  port: config.port,
  fetch: app.fetch
})

console.log(`DOJ API listening on http://localhost:${config.port}`)

function createCoachingResponse(status: string, message: string) {
  switch (status) {
    case 'CE':
      return [
        '### Compile Error',
        '',
        'Your code did not compile. Start by reading the first compiler error, then check syntax, missing imports, and language version assumptions.',
        '',
        message ? `Compiler output:\n\n\`\`\`text\n${message.trim()}\n\`\`\`` : ''
      ].join('\n')
    case 'RE':
      return [
        '### Runtime Error',
        '',
        'Your program crashed or exited with a non-zero status. Check array bounds, division by zero, failed parsing, and assumptions about empty input.',
        '',
        message ? `Runtime output:\n\n\`\`\`text\n${message.trim()}\n\`\`\`` : ''
      ].join('\n')
    case 'TLE':
      return '### Time Limit Exceeded\n\nYour solution ran too long. Revisit the algorithmic complexity and look for loops that may not terminate.'
    case 'MLE':
      return '### Memory Limit Exceeded\n\nYour solution used too much memory. Check large arrays, recursion depth, and accidental unbounded containers.'
    case 'OLE':
      return '### Output Limit Exceeded\n\nYour program printed too much output. Check debug prints and loops that keep writing after the answer is complete.'
    default:
      return [
        `### ${status}`,
        '',
        'The submission did not pass. Compare your program against the statement, sample cases, and edge conditions before looking for implementation details.',
        '',
        message ? `Judge message:\n\n\`\`\`text\n${message.trim()}\n\`\`\`` : ''
      ].join('\n')
  }
}
