import { Hono } from 'hono'
import { logger } from 'hono/logger'
import { and, asc, desc, eq, inArray, sql } from 'drizzle-orm'
import { z, ZodError } from 'zod'
import { db, schema } from '@doj/db/client'
import { enqueueJudgeTask } from '@doj/db/queue'
import { config } from './config'
import { registerAdminCoreRoutes } from './routes/admin-core'
import { registerAuthRoutes } from './routes/auth'
import { registerBbsRoutes } from './routes/bbs'
import { registerProblemRoutes } from './routes/problems'
import { getRecentBbsTopics } from './services/bbs'
import { getRuntimeSettings } from './settings'
import { numericId } from './validation'
import { authMiddleware, getOptionalAuthUser, requireAuthUser, requireGroup } from './auth'
import { createCoachingResponse } from './ai'

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

app.get('/api/config', async (c) => {
  const settings = await getRuntimeSettings()
  return c.json({
    registration: settings.registrationEnabled,
    aiCoachingEnabled: settings.aiCoachingEnabled,
    guestProblemsetVisible: settings.guestProblemsetVisible,
    sourceOpenDefault: settings.sourceOpenDefault
  })
})

registerAuthRoutes(app)
registerAdminCoreRoutes(app)
registerProblemRoutes(app)
registerBbsRoutes(app)

app.get('/api/dashboard', async (c) => {
  const [problemStats, submissionStats, userStats, contestStats, assignmentStats] =
    await Promise.all([
      countRows(schema.problems, sql`${schema.problems.visible} = true`),
      countVisibleSubmissions(),
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
    .where(eq(schema.problems.visible, true))
    .orderBy(desc(schema.submissions.createdAt))
    .limit(8)
  const recentProblems = await db
    .select({
      id: schema.problems.id,
      title: schema.problems.title,
      tags: schema.problems.tags,
      solvedCount: schema.problems.solvedCount,
      createdAt: schema.problems.createdAt
    })
    .from(schema.problems)
    .where(eq(schema.problems.visible, true))
    .orderBy(desc(schema.problems.createdAt), desc(schema.problems.id))
    .limit(6)
  const recentTopics = await getRecentBbsTopics(6)
  const authUser = await getOptionalAuthUser(c)
  const myAssignments = authUser ? await getUserAssignments(authUser.id, 5) : []

  return c.json({
    stats: {
      problems: problemStats,
      submissions: submissionStats,
      users: userStats,
      contests: contestStats,
      assignments: assignmentStats
    },
    recentSubmissions,
    recentProblems,
    recentTopics,
    myAssignments
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
  const query = z
    .object({
      page: z.coerce.number().int().positive().default(1),
      pageSize: z.coerce.number().int().min(1).max(500).default(100)
    })
    .parse(c.req.query())
  const total = await countRows(schema.users, sql`${schema.users.disabledAt} is null`)
  const list = await db
    .select({
      id: schema.users.id,
      name: schema.users.name,
      solvedCount: schema.users.solvedCount,
      submissionCount: schema.users.submissionCount,
      introduction: schema.users.introduction
    })
    .from(schema.users)
    .where(sql`${schema.users.disabledAt} is null`)
    .orderBy(
      desc(schema.users.solvedCount),
      asc(schema.users.submissionCount),
      asc(schema.users.id)
    )
    .limit(query.pageSize)
    .offset((query.page - 1) * query.pageSize)

  return c.json({ total, page: query.page, pageSize: query.pageSize, list })
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
  const list = await getUserAssignments(user.id, 50)

  return c.json({ total: list.length, list })
})

app.get('/api/my/assignments/:id', authMiddleware, async (c) => {
  const user = await requireAuthUser(c)
  const assignment = await getUserAssignmentDetail(user.id, numericId.parse(c.req.param('id')))
  if (!assignment) return c.notFound()
  return c.json(assignment)
})

app.get('/api/submissions', async (c) => {
  const query = z
    .object({
      page: z.coerce.number().int().positive().default(1),
      pageSize: z.coerce.number().int().min(1).max(100).default(50)
    })
    .parse(c.req.query())
  const total = await countVisibleSubmissions()
  const rows = await db
    .select({
      id: schema.submissions.id,
      userId: schema.submissions.userId,
      userName: schema.users.name,
      problemId: schema.submissions.problemId,
      problemTitle: schema.problems.title,
      problemVersionId: schema.submissions.problemVersionId,
      languageId: schema.submissions.languageId,
      status: schema.submissions.status,
      timeMs: schema.submissions.timeMs,
      memoryBytes: schema.submissions.memoryBytes,
      message: schema.submissions.message,
      contestId: schema.submissions.contestId,
      assignmentId: schema.submissions.assignmentId,
      createdAt: schema.submissions.createdAt,
      updatedAt: schema.submissions.updatedAt
    })
    .from(schema.submissions)
    .innerJoin(schema.problems, eq(schema.submissions.problemId, schema.problems.id))
    .innerJoin(schema.users, eq(schema.submissions.userId, schema.users.id))
    .where(eq(schema.problems.visible, true))
    .orderBy(desc(schema.submissions.createdAt))
    .limit(query.pageSize)
    .offset((query.page - 1) * query.pageSize)
  const list = rows.map((row) => ({
    ...row,
    message: row.contestId ? '' : row.message
  }))

  return c.json({ total, page: query.page, pageSize: query.pageSize, list })
})

const submitSchema = z.object({
  problemId: numericId,
  problemVersionId: numericId,
  languageId: z.string().min(1).max(64),
  sourceCode: z
    .string()
    .min(1)
    .max(200 * 1024),
  open: z.boolean().optional(),
  contestId: numericId.optional(),
  assignmentId: numericId.optional()
})

app.post('/api/submissions', authMiddleware, async (c) => {
  const user = await requireAuthUser(c)
  const body = submitSchema.parse(await c.req.json())
  const settings = await getRuntimeSettings()
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
      userId: user.id,
      problemId: body.problemId,
      problemVersionId: body.problemVersionId,
      languageId: body.languageId,
      sourceCode: body.sourceCode,
      open: body.open ?? settings.sourceOpenDefault,
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
    .select({
      id: schema.submissions.id,
      userId: schema.submissions.userId,
      problemId: schema.submissions.problemId,
      problemVersionId: schema.submissions.problemVersionId,
      languageId: schema.submissions.languageId,
      sourceCode: schema.submissions.sourceCode,
      status: schema.submissions.status,
      timeMs: schema.submissions.timeMs,
      memoryBytes: schema.submissions.memoryBytes,
      message: schema.submissions.message,
      contestId: schema.submissions.contestId,
      assignmentId: schema.submissions.assignmentId,
      createdAt: schema.submissions.createdAt,
      updatedAt: schema.submissions.updatedAt,
      problemVisible: schema.problems.visible
    })
    .from(schema.submissions)
    .innerJoin(schema.problems, eq(schema.submissions.problemId, schema.problems.id))
    .where(eq(schema.submissions.id, id))
    .limit(1)

  if (!submission) return c.notFound()
  const authUser = await getOptionalAuthUser(c)
  const canManageHiddenProblem =
    submission.userId === authUser?.id || authUser?.groups.includes('admin') === true
  if (!submission.problemVisible && !canManageHiddenProblem) return c.notFound()

  const canInspect =
    !submission.contestId ||
    submission.userId === authUser?.id ||
    authUser?.groups.includes('admin') === true
  const { problemVisible: _problemVisible, ...payload } = submission

  const cases = canInspect
    ? await db
        .select()
        .from(schema.submissionCases)
        .where(eq(schema.submissionCases.submissionId, submission.id))
        .orderBy(asc(schema.submissionCases.caseIndex))
    : []

  return c.json({
    ...payload,
    sourceCode: canInspect ? submission.sourceCode : '',
    message: canInspect ? submission.message : '',
    cases,
    restricted: !canInspect
  })
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
  const settings = await getRuntimeSettings()
  if (!settings.aiCoachingEnabled) {
    return c.json({ code: 'AI_DISABLED', message: 'AI coaching is disabled' }, 403)
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

async function countVisibleSubmissions() {
  const [row] = await db
    .select({ total: sql<number>`count(*)::int` })
    .from(schema.submissions)
    .innerJoin(schema.problems, eq(schema.submissions.problemId, schema.problems.id))
    .where(eq(schema.problems.visible, true))
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

async function getUserAssignments(userId: number, limit: number) {
  return db
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
        eq(schema.userGroups.userId, userId)
      )
    )
    .orderBy(desc(schema.assignments.createdAt))
    .limit(limit)
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
