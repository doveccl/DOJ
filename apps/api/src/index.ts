import { Hono } from 'hono'
import { logger } from 'hono/logger'
import { and, desc, eq, inArray } from 'drizzle-orm'
import { z, ZodError } from 'zod'
import { db, schema } from '@doj/db/client'
import { enqueueJudgeTask } from '@doj/db/queue'
import { languageDescriptors } from '@doj/shared/languages'
import { config } from './config'
import {
  authMiddleware,
  createToken,
  findUserByNameOrEmail,
  getAuthUser,
  getGroupByKey,
  hashPassword,
  requireAuthUser,
  requireGroup,
  verifyPassword
} from './auth'

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

app.get('/api/languages', (c) =>
  c.json({
    list: languageDescriptors.map((language) => ({
      id: language.id,
      name: language.name
    }))
  })
)

const registerSchema = z.object({
  name: z.string().regex(/^[a-zA-Z0-9][a-zA-Z0-9_]{2,31}$/),
  email: z.email(),
  password: z.string().min(8).max(128)
})

app.post('/api/auth/register', async (c) => {
  const body = registerSchema.parse(await c.req.json())
  const existing = await findUserByNameOrEmail(body.name)
  if (existing || (await findUserByNameOrEmail(body.email))) {
    return c.json({ code: 'USER_EXISTS', message: 'User name or email already exists' }, 409)
  }

  const group = await getGroupByKey('user')
  if (!group) throw new Error('builtin group missing: user')

  const result = await db.transaction(async (tx) => {
    const [user] = await tx
      .insert(schema.users)
      .values({
        name: body.name,
        email: body.email,
        passwordHash: await hashPassword(body.password)
      })
      .returning()

    await tx.insert(schema.userGroups).values({ userId: user.id, groupId: group.id })
    return user
  })

  const user = await getAuthUser(result.id)
  if (!user) throw new Error('registered user missing')

  return c.json({ token: await createToken(user.id), user }, 201)
})

const loginSchema = z.object({
  user: z.string().min(1),
  password: z.string().min(1)
})

app.post('/api/auth/login', async (c) => {
  const body = loginSchema.parse(await c.req.json())
  const record = await findUserByNameOrEmail(body.user)
  if (!record || !(await verifyPassword(body.password, record.passwordHash))) {
    return c.json({ code: 'INVALID_CREDENTIALS', message: 'Invalid user or password' }, 401)
  }

  const user = await getAuthUser(record.id)
  if (!user) return c.json({ code: 'USER_DISABLED', message: 'User is disabled' }, 403)

  return c.json({ token: await createToken(user.id), user })
})

app.get('/api/auth/self', authMiddleware, async (c) => c.json(await requireAuthUser(c)))

app.get('/api/groups', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const list = await db.select().from(schema.groups).orderBy(schema.groups.key)
  return c.json({ total: list.length, list })
})

const createGroupSchema = z.object({
  key: z.string().regex(/^[a-z][a-z0-9_-]{1,63}$/),
  name: z.string().min(1).max(128),
  description: z.string().max(500).default('')
})

app.post('/api/groups', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const body = createGroupSchema.parse(await c.req.json())
  const [group] = await db
    .insert(schema.groups)
    .values({
      key: body.key,
      name: body.name,
      description: body.description
    })
    .returning()

  return c.json(group, 201)
})

app.get('/api/users', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const list = await db
    .select({
      id: schema.users.id,
      name: schema.users.name,
      email: schema.users.email,
      createdAt: schema.users.createdAt
    })
    .from(schema.users)
    .orderBy(desc(schema.users.createdAt))
    .limit(50)

  return c.json({ total: list.length, list })
})

app.get('/api/groups/:id/users', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const groupId = c.req.param('id')
  const list = await db
    .select({
      id: schema.users.id,
      name: schema.users.name,
      email: schema.users.email,
      manager: schema.userGroups.manager,
      createdAt: schema.userGroups.createdAt
    })
    .from(schema.userGroups)
    .innerJoin(schema.users, eq(schema.userGroups.userId, schema.users.id))
    .where(eq(schema.userGroups.groupId, groupId))
    .orderBy(schema.users.name)

  return c.json({ total: list.length, list })
})

const addGroupUserSchema = z.object({
  userId: z.string().uuid(),
  manager: z.boolean().default(false)
})

app.post('/api/groups/:id/users', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const groupId = c.req.param('id')
  const body = addGroupUserSchema.parse(await c.req.json())
  const [group, user] = await Promise.all([
    db.select({ id: schema.groups.id }).from(schema.groups).where(eq(schema.groups.id, groupId)).limit(1),
    db.select({ id: schema.users.id }).from(schema.users).where(eq(schema.users.id, body.userId)).limit(1)
  ])

  if (!group.length) return c.json({ code: 'GROUP_NOT_FOUND', message: 'Group does not exist' }, 404)
  if (!user.length) return c.json({ code: 'USER_NOT_FOUND', message: 'User does not exist' }, 404)

  await db
    .insert(schema.userGroups)
    .values({ groupId, userId: body.userId, manager: body.manager })
    .onConflictDoUpdate({
      target: [schema.userGroups.userId, schema.userGroups.groupId],
      set: {
        manager: body.manager
      }
    })

  return c.json({ ok: true }, 201)
})

const dateString = z.string().refine((value) => !Number.isNaN(Date.parse(value)), {
  message: 'Expected a valid date string'
})

const createAssignmentSchema = z.object({
  title: z.string().min(1).max(160),
  description: z.string().max(10_000).default(''),
  startAt: dateString.optional(),
  dueAt: dateString.optional(),
  allowLate: z.boolean().default(false),
  aiCoachingEnabled: z.boolean().default(true),
  groupIds: z.array(z.string().uuid()).min(1),
  problems: z
    .array(
      z.object({
        problemId: z.string().uuid(),
        score: z.number().int().positive().default(100)
      })
    )
    .min(1)
})

app.get('/api/assignments', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const list = await db.select().from(schema.assignments).orderBy(desc(schema.assignments.createdAt)).limit(50)
  return c.json({ total: list.length, list })
})

app.post('/api/assignments', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const body = createAssignmentSchema.parse(await c.req.json())
  const groupIds = [...new Set(body.groupIds)]
  const problemIds = [...new Set(body.problems.map((problem) => problem.problemId))]

  const [groups, problems] = await Promise.all([
    db.select({ id: schema.groups.id }).from(schema.groups).where(inArray(schema.groups.id, groupIds)),
    db.select({ id: schema.problems.id }).from(schema.problems).where(inArray(schema.problems.id, problemIds))
  ])

  if (groups.length !== groupIds.length) {
    return c.json({ code: 'GROUP_NOT_FOUND', message: 'One or more groups do not exist' }, 400)
  }
  if (problems.length !== problemIds.length) {
    return c.json({ code: 'PROBLEM_NOT_FOUND', message: 'One or more problems do not exist' }, 400)
  }

  const result = await db.transaction(async (tx) => {
    const [assignment] = await tx
      .insert(schema.assignments)
      .values({
        title: body.title,
        description: body.description,
        startAt: body.startAt ? new Date(body.startAt) : null,
        dueAt: body.dueAt ? new Date(body.dueAt) : null,
        allowLate: body.allowLate,
        aiCoachingEnabled: body.aiCoachingEnabled
      })
      .returning()

    await tx.insert(schema.assignmentGroups).values(
      groupIds.map((groupId) => ({
        assignmentId: assignment.id,
        groupId
      }))
    )

    await tx.insert(schema.assignmentProblems).values(
      body.problems.map((problem, index) => ({
        assignmentId: assignment.id,
        problemId: problem.problemId,
        score: problem.score,
        sortOrder: index
      }))
    )

    return assignment
  })

  return c.json(await getAssignmentDetail(result.id), 201)
})

app.get('/api/assignments/:id', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const assignment = await getAssignmentDetail(c.req.param('id'))
  if (!assignment) return c.notFound()
  return c.json(assignment)
})

app.get('/api/my/assignments', authMiddleware, async (c) => {
  const user = await requireAuthUser(c)
  const list = await db
    .selectDistinct({
      id: schema.assignments.id,
      title: schema.assignments.title,
      description: schema.assignments.description,
      startAt: schema.assignments.startAt,
      dueAt: schema.assignments.dueAt,
      allowLate: schema.assignments.allowLate,
      aiCoachingEnabled: schema.assignments.aiCoachingEnabled,
      createdAt: schema.assignments.createdAt
    })
    .from(schema.assignments)
    .innerJoin(schema.assignmentGroups, eq(schema.assignmentGroups.assignmentId, schema.assignments.id))
    .innerJoin(
      schema.userGroups,
      and(
        eq(schema.userGroups.groupId, schema.assignmentGroups.groupId),
        eq(schema.userGroups.userId, user.id)
      )
    )
    .orderBy(desc(schema.assignments.createdAt))
    .limit(50)

  return c.json({ total: list.length, list })
})

app.get('/api/my/assignments/:id', authMiddleware, async (c) => {
  const user = await requireAuthUser(c)
  const assignment = await getUserAssignmentDetail(user.id, c.req.param('id'))
  if (!assignment) return c.notFound()
  return c.json(assignment)
})

app.get('/api/problems', async (c) => {
  const list = await db.select().from(schema.problems).limit(50)
  return c.json({ total: list.length, list })
})

app.get('/api/problems/:id', async (c) => {
  const id = c.req.param('id')
  const [problem] = await db.select().from(schema.problems).where(eq(schema.problems.id, id)).limit(1)
  if (!problem) return c.notFound()

  const [version] = await db
    .select()
    .from(schema.problemVersions)
    .where(eq(schema.problemVersions.problemId, problem.id))
    .orderBy(desc(schema.problemVersions.version))
    .limit(1)

  return c.json({ problem, version })
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
  problemId: z.string().uuid(),
  problemVersionId: z.string().uuid(),
  languageId: z.string().min(1).max(64),
  sourceCode: z.string().min(1).max(200 * 1024),
  contestId: z.string().uuid().optional(),
  assignmentId: z.string().uuid().optional()
})

app.post('/api/submissions', authMiddleware, async (c) => {
  const user = await requireAuthUser(c)
  const body = submitSchema.parse(await c.req.json())
  const [submission] = await db
    .insert(schema.submissions)
    .values({
      ...body,
      userId: user.id,
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

async function getAssignmentDetail(id: string) {
  const [assignment] = await db.select().from(schema.assignments).where(eq(schema.assignments.id, id)).limit(1)
  if (!assignment) return null

  const [groups, problems] = await Promise.all([
    db
      .select({
        id: schema.groups.id,
        key: schema.groups.key,
        name: schema.groups.name
      })
      .from(schema.assignmentGroups)
      .innerJoin(schema.groups, eq(schema.assignmentGroups.groupId, schema.groups.id))
      .where(eq(schema.assignmentGroups.assignmentId, id))
      .orderBy(schema.groups.key),
    db
      .select({
        id: schema.problems.id,
        title: schema.problems.title,
        score: schema.assignmentProblems.score,
        sortOrder: schema.assignmentProblems.sortOrder
      })
      .from(schema.assignmentProblems)
      .innerJoin(schema.problems, eq(schema.assignmentProblems.problemId, schema.problems.id))
      .where(eq(schema.assignmentProblems.assignmentId, id))
      .orderBy(schema.assignmentProblems.sortOrder)
  ])

  return {
    assignment,
    groups,
    problems
  }
}

async function getUserAssignmentDetail(userId: string, assignmentId: string) {
  const [match] = await db
    .select({ id: schema.assignments.id })
    .from(schema.assignments)
    .innerJoin(schema.assignmentGroups, eq(schema.assignmentGroups.assignmentId, schema.assignments.id))
    .innerJoin(
      schema.userGroups,
      and(
        eq(schema.userGroups.groupId, schema.assignmentGroups.groupId),
        eq(schema.userGroups.userId, userId)
      )
    )
    .where(eq(schema.assignments.id, assignmentId))
    .limit(1)

  if (!match) return null
  return getAssignmentDetail(assignmentId)
}
