import { Hono } from 'hono'
import { logger } from 'hono/logger'
import { and, asc, desc, eq, inArray, sql } from 'drizzle-orm'
import { z, ZodError } from 'zod'
import { db, schema } from '@doj/db/client'
import { enqueueJudgeTask } from '@doj/db/queue'
import { DockerRunner } from '@doj/runner/docker-runner'
import { parseZipTestCases } from '@doj/shared/testdata'
import { putObject } from '@doj/storage/client'
import { storageConfig } from '@doj/storage/config'
import { config } from './config'
import {
  authMiddleware,
  createToken,
  findUserByNameOrEmail,
  getAuthUser,
  getGroupByKey,
  getOptionalAuthUser,
  hashPassword,
  requireAuthUser,
  requireGroup,
  verifyPassword
} from './auth'
import { createCoachingResponse } from './ai'

const app = new Hono()
const numericId = z.coerce.number().int().positive()
const maxTestdataUploadBytes = 64 * 1024 * 1024

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

app.get('/api/dashboard', async (c) => {
  const [problemStats, submissionStats, userStats, contestStats, assignmentStats] =
    await Promise.all([
      countRows(schema.problems, sql`${schema.problems.visible} = true`),
      countRows(schema.submissions),
      countRows(schema.users, sql`${schema.users.disabledAt} is null`),
      countRows(schema.contests),
      countRows(schema.assignments)
    ])

  const recentSubmissions = await db
    .select({
      id: schema.submissions.id,
      status: schema.submissions.status,
      languageId: schema.submissions.languageId,
      timeMs: schema.submissions.timeMs,
      memoryBytes: schema.submissions.memoryBytes,
      createdAt: schema.submissions.createdAt,
      userId: schema.users.id,
      userName: schema.users.name,
      problemId: schema.problems.id,
      problemTitle: schema.problems.title
    })
    .from(schema.submissions)
    .innerJoin(schema.users, eq(schema.submissions.userId, schema.users.id))
    .innerJoin(schema.problems, eq(schema.submissions.problemId, schema.problems.id))
    .orderBy(desc(schema.submissions.createdAt))
    .limit(8)

  return c.json({
    stats: {
      problems: problemStats,
      submissions: submissionStats,
      users: userStats,
      contests: contestStats,
      assignments: assignmentStats
    },
    recentSubmissions
  })
})

app.get('/api/languages', async (c) => {
  const list = await db
    .select({
      id: schema.judgeLanguages.id,
      name: schema.judgeLanguages.name,
      sourceFile: schema.judgeLanguages.sourceFile
    })
    .from(schema.judgeLanguages)
    .where(eq(schema.judgeLanguages.enabled, true))
    .orderBy(asc(schema.judgeLanguages.sortOrder), asc(schema.judgeLanguages.id))

  return c.json({ list })
})

app.get('/api/rank', async (c) => {
  const list = await db
    .select({
      id: schema.users.id,
      name: schema.users.name,
      solvedCount: schema.users.solvedCount,
      submissionCount: schema.users.submissionCount,
      introduction: schema.users.introduction
    })
    .from(schema.users)
    .orderBy(
      desc(schema.users.solvedCount),
      asc(schema.users.submissionCount),
      asc(schema.users.id)
    )
    .limit(100)

  return c.json({ total: list.length, list })
})

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
      solvedCount: schema.users.solvedCount,
      submissionCount: schema.users.submissionCount,
      disabledAt: schema.users.disabledAt,
      createdAt: schema.users.createdAt
    })
    .from(schema.users)
    .orderBy(desc(schema.users.createdAt))
    .limit(50)

  return c.json({ total: list.length, list })
})

const updateUserSchema = z.object({
  disabled: z.boolean()
})

app.patch('/api/users/:id', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const authUser = await requireAuthUser(c)
  const id = numericId.parse(c.req.param('id'))
  const body = updateUserSchema.parse(await c.req.json())
  if (id === authUser.id && body.disabled) {
    return c.json({ code: 'CANNOT_DISABLE_SELF', message: 'Cannot disable your own account' }, 400)
  }

  const [user] = await db
    .update(schema.users)
    .set({
      disabledAt: body.disabled ? new Date() : null,
      updatedAt: new Date()
    })
    .where(eq(schema.users.id, id))
    .returning({
      id: schema.users.id,
      name: schema.users.name,
      email: schema.users.email,
      solvedCount: schema.users.solvedCount,
      submissionCount: schema.users.submissionCount,
      disabledAt: schema.users.disabledAt,
      createdAt: schema.users.createdAt
    })

  if (!user) return c.notFound()
  return c.json(user)
})

app.get('/api/groups/:id/users', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const groupId = numericId.parse(c.req.param('id'))
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
  userId: numericId,
  manager: z.boolean().default(false)
})

app.post('/api/groups/:id/users', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const groupId = numericId.parse(c.req.param('id'))
  const body = addGroupUserSchema.parse(await c.req.json())
  const [group, user] = await Promise.all([
    db
      .select({ id: schema.groups.id })
      .from(schema.groups)
      .where(eq(schema.groups.id, groupId))
      .limit(1),
    db
      .select({ id: schema.users.id })
      .from(schema.users)
      .where(eq(schema.users.id, body.userId))
      .limit(1)
  ])

  if (!group.length)
    return c.json({ code: 'GROUP_NOT_FOUND', message: 'Group does not exist' }, 404)
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

app.get('/api/admin/languages', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const list = await db
    .select()
    .from(schema.judgeLanguages)
    .orderBy(asc(schema.judgeLanguages.sortOrder), asc(schema.judgeLanguages.id))

  return c.json({ total: list.length, list })
})

const languageConfigSchema = z.object({
  id: z.string().regex(/^[a-z][a-z0-9_-]{1,63}$/),
  name: z.string().min(1).max(128),
  enabled: z.boolean().default(true),
  sourceFile: z.string().min(1).max(128),
  dockerfile: z.string().min(1).max(20_000),
  command: z.array(z.string().min(1).max(200)).default([]),
  sortOrder: z.number().int().min(0).default(100)
})

app.post('/api/admin/languages', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const body = languageConfigSchema.parse(await c.req.json())
  const [language] = await db
    .insert(schema.judgeLanguages)
    .values(body)
    .onConflictDoUpdate({
      target: schema.judgeLanguages.id,
      set: {
        name: body.name,
        enabled: body.enabled,
        sourceFile: body.sourceFile,
        dockerfile: body.dockerfile,
        command: body.command,
        sortOrder: body.sortOrder,
        updatedAt: new Date()
      }
    })
    .returning()

  return c.json(language, 201)
})

const updateLanguageConfigSchema = languageConfigSchema.omit({ id: true }).partial()

app.patch('/api/admin/languages/:id', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const body = updateLanguageConfigSchema.parse(await c.req.json())
  const [language] = await db
    .update(schema.judgeLanguages)
    .set({
      ...body,
      updatedAt: new Date()
    })
    .where(eq(schema.judgeLanguages.id, c.req.param('id')))
    .returning()

  if (!language) return c.notFound()
  return c.json(language)
})

app.get('/api/admin/runners', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const list = await db
    .select()
    .from(schema.judgeRunners)
    .orderBy(asc(schema.judgeRunners.sortOrder), asc(schema.judgeRunners.id))

  return c.json({ total: list.length, list })
})

const runnerConfigSchema = z.object({
  key: z.string().regex(/^[a-z][a-z0-9_-]{1,63}$/),
  name: z.string().min(1).max(128),
  enabled: z.boolean().default(true),
  kind: z.enum(['docker']).default('docker'),
  endpoint: z.string().max(1000).optional(),
  concurrency: z.number().int().positive().default(2),
  sortOrder: z.number().int().min(0).default(100)
})

app.post('/api/admin/runners', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const body = runnerConfigSchema.parse(await c.req.json())
  const [runner] = await db
    .insert(schema.judgeRunners)
    .values({
      ...body,
      endpoint: body.endpoint || null,
      authHeader: null
    })
    .onConflictDoUpdate({
      target: schema.judgeRunners.key,
      set: {
        name: body.name,
        enabled: body.enabled,
        kind: body.kind,
        endpoint: body.endpoint || null,
        authHeader: null,
        concurrency: body.concurrency,
        sortOrder: body.sortOrder,
        updatedAt: new Date()
      }
    })
    .returning()

  return c.json(runner, 201)
})

app.post('/api/admin/runners/:id/check', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const id = numericId.parse(c.req.param('id'))
  const [runner] = await db
    .select()
    .from(schema.judgeRunners)
    .where(eq(schema.judgeRunners.id, id))
    .limit(1)
  if (!runner) return c.notFound()
  if (runner.kind !== 'docker') {
    return c.json(
      { code: 'UNSUPPORTED_RUNNER', message: `Unsupported runner kind: ${runner.kind}` },
      400
    )
  }

  const result = await new DockerRunner({
    endpoint: runner.endpoint,
    authHeader: runner.authHeader
  }).check()

  return c.json({ ok: true, runnerId: runner.id, ...result })
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
  groupIds: z.array(numericId).min(1),
  problems: z
    .array(
      z.object({
        problemId: numericId,
        score: z.number().int().positive().default(100)
      })
    )
    .min(1)
})

app.get('/api/assignments', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const list = await db
    .select()
    .from(schema.assignments)
    .orderBy(desc(schema.assignments.createdAt))
    .limit(50)
  return c.json({ total: list.length, list })
})

app.post('/api/assignments', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const body = createAssignmentSchema.parse(await c.req.json())
  const groupIds = [...new Set(body.groupIds)]
  const problemIds = [...new Set(body.problems.map((problem) => problem.problemId))]

  const [groups, problems] = await Promise.all([
    db
      .select({ id: schema.groups.id })
      .from(schema.groups)
      .where(inArray(schema.groups.id, groupIds)),
    db
      .select({ id: schema.problems.id })
      .from(schema.problems)
      .where(inArray(schema.problems.id, problemIds))
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

  const assignment = await getAssignmentDetail(numericId.parse(c.req.param('id')))
  if (!assignment) return c.notFound()
  return c.json(assignment)
})

app.get('/api/assignments/:id/report', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const report = await getAssignmentReport(numericId.parse(c.req.param('id')))
  if (!report) return c.notFound()
  return c.json(report)
})

const createContestSchema = z.object({
  title: z.string().min(1).max(160),
  description: z.string().max(10_000).default(''),
  type: z.enum(['OI', 'ICPC']).default('OI'),
  startAt: dateString,
  endAt: dateString,
  freezeAt: dateString.optional(),
  problems: z
    .array(
      z.object({
        problemId: numericId,
        key: z.string().min(1).max(32),
        score: z.number().int().positive().default(100)
      })
    )
    .min(1)
})

app.get('/api/contests', async (c) => {
  const list = await db
    .select()
    .from(schema.contests)
    .orderBy(desc(schema.contests.startAt))
    .limit(50)
  return c.json({ total: list.length, list })
})

app.post('/api/contests', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const body = createContestSchema.parse(await c.req.json())
  const startAt = new Date(body.startAt)
  const endAt = new Date(body.endAt)
  const freezeAt = body.freezeAt ? new Date(body.freezeAt) : null
  if (endAt <= startAt) {
    return c.json({ code: 'INVALID_CONTEST_TIME', message: 'endAt must be after startAt' }, 400)
  }
  if (freezeAt && (freezeAt <= startAt || freezeAt >= endAt)) {
    return c.json(
      { code: 'INVALID_CONTEST_FREEZE', message: 'freezeAt must be between startAt and endAt' },
      400
    )
  }

  const problemIds = [...new Set(body.problems.map((problem) => problem.problemId))]
  const problems = await db
    .select({ id: schema.problems.id })
    .from(schema.problems)
    .where(inArray(schema.problems.id, problemIds))
  if (problems.length !== problemIds.length) {
    return c.json({ code: 'PROBLEM_NOT_FOUND', message: 'One or more problems do not exist' }, 400)
  }

  const result = await db.transaction(async (tx) => {
    const [contest] = await tx
      .insert(schema.contests)
      .values({
        title: body.title,
        description: body.description,
        type: body.type,
        startAt,
        endAt,
        freezeAt
      })
      .returning()

    await tx.insert(schema.contestProblems).values(
      body.problems.map((problem, index) => ({
        contestId: contest.id,
        problemId: problem.problemId,
        key: problem.key,
        score: problem.score,
        sortOrder: index
      }))
    )

    return contest
  })

  return c.json(await getContestDetail(result.id), 201)
})

app.get('/api/contests/:id', async (c) => {
  const contest = await getContestDetail(numericId.parse(c.req.param('id')))
  if (!contest) return c.notFound()
  return c.json(contest)
})

app.get('/api/contests/:id/scoreboard', async (c) => {
  const scoreboard = await getContestScoreboard(numericId.parse(c.req.param('id')))
  if (!scoreboard) return c.notFound()
  return c.json(scoreboard)
})

app.get('/api/contests/:id/scoreboard/reveal', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const scoreboard = await getContestScoreboard(numericId.parse(c.req.param('id')), {
    reveal: true
  })
  if (!scoreboard) return c.notFound()
  return c.json(scoreboard)
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
    .innerJoin(
      schema.assignmentGroups,
      eq(schema.assignmentGroups.assignmentId, schema.assignments.id)
    )
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
  const assignment = await getUserAssignmentDetail(user.id, numericId.parse(c.req.param('id')))
  if (!assignment) return c.notFound()
  return c.json(assignment)
})

app.get('/api/problems', async (c) => {
  const list = await db
    .select()
    .from(schema.problems)
    .where(eq(schema.problems.visible, true))
    .orderBy(asc(schema.problems.id))
    .limit(50)
  return c.json({ total: list.length, list })
})

app.get('/api/admin/problems', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const list = await db.select().from(schema.problems).orderBy(asc(schema.problems.id)).limit(100)
  return c.json({ total: list.length, list })
})

app.get('/api/admin/problems/:id', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const detail = await getProblemDetail(numericId.parse(c.req.param('id')), {
    includeHiddenCases: true,
    includeHiddenProblem: true
  })
  if (!detail) return c.notFound()
  return c.json(detail)
})

app.get('/api/problems/:id', async (c) => {
  const detail = await getProblemDetail(numericId.parse(c.req.param('id')))
  if (!detail) return c.notFound()
  return c.json(detail)
})

const problemTestCaseSchema = z.object({
  name: z.string().max(120).optional(),
  input: z.string().max(256 * 1024),
  output: z.string().max(256 * 1024),
  hidden: z.boolean().default(false)
})

const createProblemSchema = z.object({
  title: z.string().min(1).max(160),
  slug: z.string().min(1).max(160).optional(),
  tags: z.array(z.string()).default([]),
  statementMarkdown: z.string().min(1),
  timeLimitMs: z.number().int().positive().default(1000),
  memoryLimitBytes: z
    .number()
    .int()
    .positive()
    .default(256 * 1024 * 1024),
  outputLimitBytes: z
    .number()
    .int()
    .positive()
    .default(64 * 1024 * 1024),
  testCases: z.array(problemTestCaseSchema).max(100).default([])
})

const updateProblemSchema = z
  .object({
    title: z.string().min(1).max(160).optional(),
    slug: z.string().min(1).max(160).optional(),
    tags: z.array(z.string()).optional(),
    visible: z.boolean().optional(),
    statementMarkdown: z.string().min(1).optional(),
    timeLimitMs: z.number().int().positive().optional(),
    memoryLimitBytes: z.number().int().positive().optional(),
    outputLimitBytes: z.number().int().positive().optional(),
    testCases: z.array(problemTestCaseSchema).max(100).optional()
  })
  .refine((value) => Object.keys(value).length > 0, {
    message: 'At least one field must be updated'
  })

app.patch('/api/problems/:id', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const id = numericId.parse(c.req.param('id'))
  const body = updateProblemSchema.parse(await c.req.json())
  const detail = await getProblemDetail(id, {
    includeHiddenCases: true,
    includeHiddenProblem: true
  })
  if (!detail) return c.notFound()

  const versionFieldsChanged = [
    'statementMarkdown',
    'timeLimitMs',
    'memoryLimitBytes',
    'outputLimitBytes',
    'testCases'
  ].some((key) => key in body)

  await db.transaction(async (tx) => {
    const problemPatch: Partial<typeof schema.problems.$inferInsert> = {}
    if ('title' in body) problemPatch.title = body.title
    if ('slug' in body) problemPatch.slug = body.slug ?? null
    if ('tags' in body) problemPatch.tags = body.tags ?? []
    if ('visible' in body) problemPatch.visible = body.visible
    if (Object.keys(problemPatch).length) {
      problemPatch.updatedAt = new Date()
      await tx.update(schema.problems).set(problemPatch).where(eq(schema.problems.id, id))
    }

    if (versionFieldsChanged && detail.version) {
      await tx.insert(schema.problemVersions).values({
        problemId: id,
        version: detail.version.version + 1,
        statementMarkdown: body.statementMarkdown ?? detail.version.statementMarkdown,
        timeLimitMs: body.timeLimitMs ?? detail.version.timeLimitMs,
        memoryLimitBytes: body.memoryLimitBytes ?? detail.version.memoryLimitBytes,
        outputLimitBytes: body.outputLimitBytes ?? detail.version.outputLimitBytes,
        testCases: body.testCases ?? detail.version.testCases,
        testdataFileId: 'testCases' in body ? null : detail.version.testdataFileId,
        checkerFileId: detail.version.checkerFileId,
        interactorFileId: detail.version.interactorFileId
      })
    }
  })

  const updated = await getProblemDetail(id, {
    includeHiddenCases: true,
    includeHiddenProblem: true
  })
  return c.json(updated)
})

async function getProblemDetail(
  id: number,
  options: { includeHiddenCases?: boolean; includeHiddenProblem?: boolean } = {}
) {
  const [problem] = await db
    .select()
    .from(schema.problems)
    .where(eq(schema.problems.id, id))
    .limit(1)
  if (!problem || (!problem.visible && !options.includeHiddenProblem)) return null

  const [version] = await db
    .select()
    .from(schema.problemVersions)
    .where(eq(schema.problemVersions.problemId, problem.id))
    .orderBy(desc(schema.problemVersions.version))
    .limit(1)

  return {
    problem,
    version: version
      ? {
          ...version,
          testCases: options.includeHiddenCases
            ? version.testCases
            : version.testCases.filter((testCase) => !testCase.hidden)
        }
      : null
  }
}

app.post('/api/problems', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

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
        outputLimitBytes: body.outputLimitBytes,
        testCases: body.testCases
      })
      .returning()

    return { problem, version }
  })

  return c.json(result, 201)
})

app.post('/api/problems/:id/testdata', authMiddleware, async (c) => {
  const denied = await requireGroup(c, 'admin')
  if (denied) return denied

  const problemId = numericId.parse(c.req.param('id'))
  const [version] = await db
    .select()
    .from(schema.problemVersions)
    .where(eq(schema.problemVersions.problemId, problemId))
    .orderBy(desc(schema.problemVersions.version))
    .limit(1)
  if (!version) return c.notFound()

  const form = await c.req.formData()
  const upload = form.get('file')
  if (!(upload instanceof File)) {
    return c.json(
      { code: 'MISSING_FILE', message: 'Expected multipart file field named file' },
      400
    )
  }

  const bytes = new Uint8Array(await upload.arrayBuffer())
  if (bytes.byteLength > maxTestdataUploadBytes) {
    return c.json({ code: 'FILE_TOO_LARGE', message: 'Testdata ZIP is too large' }, 413)
  }

  let cases
  try {
    cases = parseZipTestCases(bytes)
  } catch (cause) {
    return c.json(
      {
        code: 'INVALID_TESTDATA_ZIP',
        message: cause instanceof Error ? cause.message : 'Invalid ZIP testdata'
      },
      400
    )
  }
  if (!cases.length) {
    return c.json({ code: 'EMPTY_TESTDATA', message: 'ZIP must contain .in/.out case pairs' }, 400)
  }

  const objectKey = `problems/${problemId}/testdata/${crypto.randomUUID()}.zip`
  await putObject({
    key: objectKey,
    body: bytes,
    contentType: upload.type || 'application/zip'
  })

  const [file] = await db
    .insert(schema.files)
    .values({
      bucket: storageConfig.bucket,
      objectKey,
      filename: upload.name || `problem-${problemId}-testdata.zip`,
      contentType: upload.type || 'application/zip',
      sizeBytes: bytes.byteLength,
      metadata: {
        problemId,
        caseCount: cases.length,
        kind: 'problem-testdata'
      }
    })
    .returning()

  const [updated] = await db
    .update(schema.problemVersions)
    .set({ testdataFileId: file.id })
    .where(eq(schema.problemVersions.id, version.id))
    .returning()

  return c.json({ file, version: updated, caseCount: cases.length }, 201)
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
  problemId: numericId,
  problemVersionId: numericId,
  languageId: z.string().min(1).max(64),
  sourceCode: z
    .string()
    .min(1)
    .max(200 * 1024),
  contestId: numericId.optional(),
  assignmentId: numericId.optional()
})

app.post('/api/submissions', authMiddleware, async (c) => {
  const user = await requireAuthUser(c)
  const body = submitSchema.parse(await c.req.json())
  if (body.contestId && body.assignmentId) {
    return c.json(
      { code: 'AMBIGUOUS_SUBMISSION_CONTEXT', message: 'Choose contest or assignment, not both' },
      400
    )
  }

  const [language] = await db
    .select({ id: schema.judgeLanguages.id })
    .from(schema.judgeLanguages)
    .where(
      and(eq(schema.judgeLanguages.id, body.languageId), eq(schema.judgeLanguages.enabled, true))
    )
    .limit(1)

  if (!language) {
    return c.json({ code: 'LANGUAGE_DISABLED', message: 'Language is not enabled' }, 400)
  }

  const [target] = await db
    .select({
      problemId: schema.problems.id,
      visible: schema.problems.visible,
      versionId: schema.problemVersions.id
    })
    .from(schema.problemVersions)
    .innerJoin(schema.problems, eq(schema.problemVersions.problemId, schema.problems.id))
    .where(
      and(
        eq(schema.problemVersions.id, body.problemVersionId),
        eq(schema.problemVersions.problemId, body.problemId)
      )
    )
    .limit(1)
  if (!target) {
    return c.json(
      {
        code: 'PROBLEM_VERSION_NOT_FOUND',
        message: 'Problem version does not belong to this problem'
      },
      400
    )
  }

  if (body.contestId) {
    const contestCheck = await validateContestSubmission(body.contestId, body.problemId)
    if (contestCheck)
      return c.json({ code: contestCheck.code, message: contestCheck.message }, contestCheck.status)
  } else if (body.assignmentId) {
    const assignmentCheck = await validateAssignmentSubmission(
      user.id,
      body.assignmentId,
      body.problemId
    )
    if (assignmentCheck) {
      return c.json(
        { code: assignmentCheck.code, message: assignmentCheck.message },
        assignmentCheck.status
      )
    }
  } else if (!target.visible) {
    return c.notFound()
  }

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
  await Promise.all([
    db
      .update(schema.users)
      .set({ submissionCount: sql`${schema.users.submissionCount} + 1`, updatedAt: new Date() })
      .where(eq(schema.users.id, user.id)),
    db
      .update(schema.problems)
      .set({
        submissionCount: sql`${schema.problems.submissionCount} + 1`,
        updatedAt: new Date()
      })
      .where(eq(schema.problems.id, body.problemId))
  ])
  return c.json(submission, 201)
})

app.get('/api/submissions/:id', async (c) => {
  const id = numericId.parse(c.req.param('id'))
  const [submission] = await db
    .select()
    .from(schema.submissions)
    .where(eq(schema.submissions.id, id))
    .limit(1)

  if (!submission) return c.notFound()
  const authUser = await getOptionalAuthUser(c)
  const canInspect =
    !submission.contestId ||
    submission.userId === authUser?.id ||
    authUser?.groups.includes('admin') === true

  const cases = canInspect
    ? await db
        .select()
        .from(schema.submissionCases)
        .where(eq(schema.submissionCases.submissionId, submission.id))
        .orderBy(asc(schema.submissionCases.caseIndex))
    : []

  return c.json({
    ...submission,
    sourceCode: canInspect ? submission.sourceCode : '',
    message: canInspect ? submission.message : '',
    cases,
    restricted: !canInspect
  })
})

const createTopicSchema = z.object({
  title: z.string().min(1).max(160),
  contentMarkdown: z.string().min(1).max(20_000),
  tags: z.array(z.string().min(1).max(40)).max(12).default([]),
  linkedProblemId: numericId.optional(),
  linkedContestId: numericId.optional()
})

app.get('/api/bbs/topics', async (c) => {
  const list = await db
    .select({
      id: schema.bbsTopics.id,
      title: schema.bbsTopics.title,
      tags: schema.bbsTopics.tags,
      linkedProblemId: schema.bbsTopics.linkedProblemId,
      linkedContestId: schema.bbsTopics.linkedContestId,
      createdAt: schema.bbsTopics.createdAt,
      updatedAt: schema.bbsTopics.updatedAt,
      userId: schema.users.id,
      userName: schema.users.name
    })
    .from(schema.bbsTopics)
    .innerJoin(schema.users, eq(schema.bbsTopics.userId, schema.users.id))
    .orderBy(desc(schema.bbsTopics.updatedAt))
    .limit(50)

  return c.json({ total: list.length, list })
})

app.post('/api/bbs/topics', authMiddleware, async (c) => {
  const user = await requireAuthUser(c)
  const body = createTopicSchema.parse(await c.req.json())

  const topic = await db.transaction(async (tx) => {
    const [created] = await tx
      .insert(schema.bbsTopics)
      .values({
        userId: user.id,
        title: body.title,
        tags: body.tags,
        linkedProblemId: body.linkedProblemId ?? null,
        linkedContestId: body.linkedContestId ?? null
      })
      .returning()

    await tx.insert(schema.bbsReplies).values({
      topicId: created.id,
      userId: user.id,
      contentMarkdown: body.contentMarkdown
    })

    return created
  })

  return c.json(await getTopicDetail(topic.id), 201)
})

app.get('/api/bbs/topics/:id', async (c) => {
  const topic = await getTopicDetail(numericId.parse(c.req.param('id')))
  if (!topic) return c.notFound()
  return c.json(topic)
})

const createReplySchema = z.object({
  contentMarkdown: z.string().min(1).max(20_000)
})

app.post('/api/bbs/topics/:id/replies', authMiddleware, async (c) => {
  const user = await requireAuthUser(c)
  const topicId = numericId.parse(c.req.param('id'))
  const body = createReplySchema.parse(await c.req.json())

  const [topic] = await db
    .select()
    .from(schema.bbsTopics)
    .where(eq(schema.bbsTopics.id, topicId))
    .limit(1)
  if (!topic) return c.notFound()

  const [reply] = await db
    .insert(schema.bbsReplies)
    .values({
      topicId,
      userId: user.id,
      contentMarkdown: body.contentMarkdown
    })
    .returning()

  await db
    .update(schema.bbsTopics)
    .set({ updatedAt: new Date() })
    .where(eq(schema.bbsTopics.id, topicId))

  return c.json(reply, 201)
})

app.post('/api/submissions/:id/coach', authMiddleware, async (c) => {
  const user = await requireAuthUser(c)
  const id = numericId.parse(c.req.param('id'))
  const [submission] = await db
    .select()
    .from(schema.submissions)
    .where(eq(schema.submissions.id, id))
    .limit(1)

  if (!submission) return c.notFound()
  if (submission.userId !== user.id && !user.groups.includes('admin')) {
    return c.json({ code: 'FORBIDDEN', message: 'Cannot coach another user submission' }, 403)
  }
  if (submission.contestId) {
    return c.json(
      { code: 'AI_DISABLED_IN_CONTEST', message: 'AI coaching is disabled in contests' },
      403
    )
  }
  if (submission.assignmentId) {
    const [assignment] = await db
      .select({ aiCoachingEnabled: schema.assignments.aiCoachingEnabled })
      .from(schema.assignments)
      .where(eq(schema.assignments.id, submission.assignmentId))
      .limit(1)
    if (assignment && !assignment.aiCoachingEnabled) {
      return c.json(
        {
          code: 'AI_DISABLED_IN_ASSIGNMENT',
          message: 'AI coaching is disabled in this assignment'
        },
        403
      )
    }
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

  const coaching = await createCoachingResponse({
    status: submission.status,
    message: submission.message,
    languageId: submission.languageId,
    sourceCode: submission.sourceCode
  })

  const [session] = await db
    .insert(schema.aiCoachingSessions)
    .values({
      userId: submission.userId,
      submissionId: submission.id,
      model: coaching.model,
      promptVersion: 'non-ac-v1',
      responseMarkdown: coaching.responseMarkdown,
      metadata: {
        status: submission.status,
        languageId: submission.languageId,
        provider: config.aiProvider
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

async function getAssignmentDetail(id: number) {
  const [assignment] = await db
    .select()
    .from(schema.assignments)
    .where(eq(schema.assignments.id, id))
    .limit(1)
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

async function countRows(table: any, where?: any) {
  const query = db.select({ total: sql<number>`count(*)::int` }).from(table)
  const [row] = await (where ? query.where(where) : query)
  return row?.total ?? 0
}

async function getAssignmentReport(id: number) {
  const detail = await getAssignmentDetail(id)
  if (!detail) return null

  const [students, submissions] = await Promise.all([
    db
      .selectDistinct({
        id: schema.users.id,
        name: schema.users.name,
        email: schema.users.email
      })
      .from(schema.assignmentGroups)
      .innerJoin(schema.userGroups, eq(schema.userGroups.groupId, schema.assignmentGroups.groupId))
      .innerJoin(schema.users, eq(schema.users.id, schema.userGroups.userId))
      .where(eq(schema.assignmentGroups.assignmentId, id))
      .orderBy(schema.users.name),
    db
      .select({
        id: schema.submissions.id,
        userId: schema.submissions.userId,
        problemId: schema.submissions.problemId,
        status: schema.submissions.status,
        createdAt: schema.submissions.createdAt
      })
      .from(schema.submissions)
      .where(eq(schema.submissions.assignmentId, id))
      .orderBy(asc(schema.submissions.createdAt))
  ])

  const problemIds = new Set(detail.problems.map((problem) => problem.id))
  const rows = students.map((student) => ({
    userId: student.id,
    userName: student.name,
    email: student.email,
    solved: 0,
    submitted: 0,
    problems: Object.fromEntries(
      detail.problems.map((problem) => [
        String(problem.id),
        {
          status: 'WAITING',
          attempts: 0,
          bestSubmissionId: null as number | null,
          lastSubmissionId: null as number | null,
          updatedAt: null as string | null
        }
      ])
    )
  }))
  const rowByUser = new Map(rows.map((row) => [row.userId, row]))

  for (const submission of submissions) {
    if (!problemIds.has(submission.problemId)) continue
    const row = rowByUser.get(submission.userId)
    if (!row) continue
    const cell = row.problems[String(submission.problemId)]
    if (!cell) continue

    cell.attempts += 1
    cell.lastSubmissionId = submission.id
    cell.updatedAt = submission.createdAt.toISOString()
    if (cell.status !== 'AC') {
      cell.status = submission.status
      cell.bestSubmissionId = submission.id
    }
    if (submission.status === 'AC') {
      cell.status = 'AC'
      cell.bestSubmissionId = submission.id
    }
  }

  for (const row of rows) {
    const cells = Object.values(row.problems)
    row.submitted = cells.filter((cell) => cell.attempts > 0).length
    row.solved = cells.filter((cell) => cell.status === 'AC').length
  }

  return {
    assignment: detail.assignment,
    problems: detail.problems,
    rows
  }
}

async function getContestDetail(id: number) {
  const [contest] = await db
    .select()
    .from(schema.contests)
    .where(eq(schema.contests.id, id))
    .limit(1)
  if (!contest) return null

  const problems = await db
    .select({
      id: schema.problems.id,
      title: schema.problems.title,
      key: schema.contestProblems.key,
      score: schema.contestProblems.score,
      sortOrder: schema.contestProblems.sortOrder
    })
    .from(schema.contestProblems)
    .innerJoin(schema.problems, eq(schema.contestProblems.problemId, schema.problems.id))
    .where(eq(schema.contestProblems.contestId, id))
    .orderBy(schema.contestProblems.sortOrder)

  return {
    contest,
    problems
  }
}

async function getContestScoreboard(id: number, options: { reveal?: boolean } = {}) {
  const detail = await getContestDetail(id)
  if (!detail) return null
  const freezeAt = detail.contest.freezeAt
  const frozen = !!freezeAt && Date.now() >= freezeAt.getTime()
  const reveal = options.reveal === true

  const submissions = await db
    .select({
      id: schema.submissions.id,
      userId: schema.submissions.userId,
      userName: schema.users.name,
      problemId: schema.submissions.problemId,
      status: schema.submissions.status,
      createdAt: schema.submissions.createdAt
    })
    .from(schema.submissions)
    .innerJoin(schema.users, eq(schema.submissions.userId, schema.users.id))
    .where(eq(schema.submissions.contestId, id))
    .orderBy(asc(schema.submissions.createdAt))

  const problemKeyById = new Map(detail.problems.map((problem) => [problem.id, problem.key]))
  const rows = new Map<
    number,
    {
      userId: number
      userName: string
      solved: number
      penalty: number
      problems: Record<
        string,
        { attempts: number; solved: boolean; penalty: number; frozenAttempts: number }
      >
    }
  >()

  for (const submission of submissions) {
    const key = problemKeyById.get(submission.problemId)
    if (!key) continue

    const row = rows.get(submission.userId) ?? {
      userId: submission.userId,
      userName: submission.userName,
      solved: 0,
      penalty: 0,
      problems: {}
    }
    rows.set(submission.userId, row)

    const cell = row.problems[key] ?? {
      attempts: 0,
      solved: false,
      penalty: 0,
      frozenAttempts: 0
    }
    row.problems[key] = cell

    if (frozen && !reveal && freezeAt && submission.createdAt >= freezeAt) {
      cell.frozenAttempts += 1
      continue
    }
    if (cell.solved) continue

    cell.attempts += 1
    if (submission.status === 'AC') {
      cell.solved = true
      cell.penalty =
        Math.max(
          0,
          Math.floor((submission.createdAt.getTime() - detail.contest.startAt.getTime()) / 60000)
        ) +
        (cell.attempts - 1) * 20
      row.solved += 1
      row.penalty += cell.penalty
    }
  }

  return {
    contest: detail.contest,
    frozen,
    revealed: reveal,
    visibleUntil: frozen && !reveal ? freezeAt?.toISOString() : null,
    problems: detail.problems.map((problem) => ({
      id: problem.id,
      key: problem.key,
      title: problem.title
    })),
    rows: [...rows.values()].sort(
      (a, b) => b.solved - a.solved || a.penalty - b.penalty || a.userId - b.userId
    )
  }
}

async function validateContestSubmission(contestId: number, problemId: number) {
  const [contest] = await db
    .select()
    .from(schema.contests)
    .where(eq(schema.contests.id, contestId))
    .limit(1)
  if (!contest)
    return { status: 404 as const, code: 'CONTEST_NOT_FOUND', message: 'Contest does not exist' }

  const now = new Date()
  if (now < contest.startAt) {
    return { status: 400 as const, code: 'CONTEST_NOT_STARTED', message: 'Contest has not started' }
  }
  if (now > contest.endAt) {
    return { status: 400 as const, code: 'CONTEST_ENDED', message: 'Contest has ended' }
  }

  const [contestProblem] = await db
    .select({ problemId: schema.contestProblems.problemId })
    .from(schema.contestProblems)
    .where(
      and(
        eq(schema.contestProblems.contestId, contestId),
        eq(schema.contestProblems.problemId, problemId)
      )
    )
    .limit(1)

  if (!contestProblem) {
    return {
      status: 400 as const,
      code: 'PROBLEM_NOT_IN_CONTEST',
      message: 'Problem does not belong to this contest'
    }
  }

  return null
}

async function validateAssignmentSubmission(
  userId: number,
  assignmentId: number,
  problemId: number
) {
  const [assignment] = await db
    .select({
      id: schema.assignments.id,
      startAt: schema.assignments.startAt,
      dueAt: schema.assignments.dueAt,
      allowLate: schema.assignments.allowLate
    })
    .from(schema.assignments)
    .innerJoin(
      schema.assignmentGroups,
      eq(schema.assignmentGroups.assignmentId, schema.assignments.id)
    )
    .innerJoin(
      schema.userGroups,
      and(
        eq(schema.userGroups.groupId, schema.assignmentGroups.groupId),
        eq(schema.userGroups.userId, userId)
      )
    )
    .where(eq(schema.assignments.id, assignmentId))
    .limit(1)

  if (!assignment) {
    return {
      status: 404 as const,
      code: 'ASSIGNMENT_NOT_FOUND',
      message: 'Assignment does not exist for this user'
    }
  }

  const now = new Date()
  if (assignment.startAt && now < assignment.startAt) {
    return {
      status: 400 as const,
      code: 'ASSIGNMENT_NOT_STARTED',
      message: 'Assignment has not started'
    }
  }
  if (assignment.dueAt && !assignment.allowLate && now > assignment.dueAt) {
    return {
      status: 400 as const,
      code: 'ASSIGNMENT_CLOSED',
      message: 'Assignment deadline has passed'
    }
  }

  const [assignmentProblem] = await db
    .select({ problemId: schema.assignmentProblems.problemId })
    .from(schema.assignmentProblems)
    .where(
      and(
        eq(schema.assignmentProblems.assignmentId, assignmentId),
        eq(schema.assignmentProblems.problemId, problemId)
      )
    )
    .limit(1)

  if (!assignmentProblem) {
    return {
      status: 400 as const,
      code: 'PROBLEM_NOT_IN_ASSIGNMENT',
      message: 'Problem does not belong to this assignment'
    }
  }

  return null
}

async function getTopicDetail(id: number) {
  const [topic] = await db
    .select({
      id: schema.bbsTopics.id,
      title: schema.bbsTopics.title,
      tags: schema.bbsTopics.tags,
      linkedProblemId: schema.bbsTopics.linkedProblemId,
      linkedContestId: schema.bbsTopics.linkedContestId,
      createdAt: schema.bbsTopics.createdAt,
      updatedAt: schema.bbsTopics.updatedAt,
      userId: schema.users.id,
      userName: schema.users.name
    })
    .from(schema.bbsTopics)
    .innerJoin(schema.users, eq(schema.bbsTopics.userId, schema.users.id))
    .where(eq(schema.bbsTopics.id, id))
    .limit(1)

  if (!topic) return null

  const replies = await db
    .select({
      id: schema.bbsReplies.id,
      contentMarkdown: schema.bbsReplies.contentMarkdown,
      createdAt: schema.bbsReplies.createdAt,
      userId: schema.users.id,
      userName: schema.users.name
    })
    .from(schema.bbsReplies)
    .innerJoin(schema.users, eq(schema.bbsReplies.userId, schema.users.id))
    .where(eq(schema.bbsReplies.topicId, id))
    .orderBy(asc(schema.bbsReplies.createdAt))

  return { topic, replies }
}

async function getUserAssignmentDetail(userId: number, assignmentId: number) {
  const [match] = await db
    .select({ id: schema.assignments.id })
    .from(schema.assignments)
    .innerJoin(
      schema.assignmentGroups,
      eq(schema.assignmentGroups.assignmentId, schema.assignments.id)
    )
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
