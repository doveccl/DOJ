import { Hono } from 'hono'
import { and, asc, desc, eq, isNull, or, sql } from 'drizzle-orm'
import { createHash } from 'node:crypto'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { enqueueJudgeTask } from '@doj/db/queue'
import { createCoachingResponse } from '../ai'
import { authMiddleware, denyGuestAccess, getOptionalAuthUser, requireAuthUser } from '../auth'
import { apiError, notFound } from '../errors'
import { checkRateLimit, clientIp } from '../rate-limit'
import { validateAssignmentSubmission } from '../services/assignments'
import { validateContestSubmission } from '../services/contests'
import { getRuntimeSettings } from '../settings'
import { recordSubmissionCreated } from '../services/stats'
import { getJudgeProgress } from '../progress'
import { listQuerySchema, numericId, pageOffset } from '../validation'

const submitSchema = z.object({
  problemId: numericId,
  languageId: z.string().min(1).max(64),
  code: z.string().min(1).max(256 * 1024),
  public: z.boolean().optional(),
  contestId: numericId.optional(),
  assignmentId: numericId.optional()
})

const judgeStatusSchema = z.enum(['WAITING', 'JUDGING', 'AC', 'WA', 'PE', 'TLE', 'MLE', 'OLE', 'RE', 'CE', 'SE'])

const submissionListQuerySchema = listQuerySchema.extend({
  status: judgeStatusSchema.optional(),
  problemId: numericId.optional(),
  userId: numericId.optional(),
  languageId: z.string().min(1).max(64).optional(),
  contestId: numericId.optional(),
  assignmentId: numericId.optional()
})

export function registerSubmissionRoutes(app: Hono) {
  app.get('/api/submissions', async (c) => {
    const denied = await denyGuestAccess(c, 'Sign in to view submissions')
    if (denied) return denied

    const query = submissionListQuerySchema.parse(c.req.query())
    const authUser = await getOptionalAuthUser(c)
    const isAdmin = authUser?.admin ?? false
    const where = visibleSubmissionListWhere(authUser?.id, isAdmin, query)
    const [total, rows] = await Promise.all([
      countSubmissionRows(where),
      db
        .select({
          id: schema.submissions.id,
          userId: schema.submissions.userId,
          userName: schema.users.name,
          userEmail: schema.users.email,
          problemId: schema.submissions.problemId,
          problemTitle: schema.problems.title,
          languageId: schema.submissions.languageId,
          status: schema.submissions.status,
          timeMs: schema.submissions.timeMs,
          memoryBytes: schema.submissions.memoryBytes,
          score: schema.submissions.score,
          public: schema.submissions.public,
          contestId: schema.submissions.contestId,
          assignmentId: schema.submissions.assignmentId,
          createdAt: schema.submissions.createdAt,
          updatedAt: schema.submissions.updatedAt,
          problemVisible: schema.problems.visible,
          problemDeletedAt: schema.problems.deletedAt,
          contestType: schema.contests.type,
          contestStartAt: schema.contests.startAt,
          contestEndAt: schema.contests.endAt,
          contestFreezeAt: schema.contests.freezeAt
        })
        .from(schema.submissions)
        .innerJoin(schema.problems, eq(schema.submissions.problemId, schema.problems.id))
        .leftJoin(schema.contests, eq(schema.submissions.contestId, schema.contests.id))
        .innerJoin(schema.users, eq(schema.submissions.userId, schema.users.id))
        .where(where)
        .orderBy(desc(schema.submissions.createdAt))
        .limit(query.pageSize)
        .offset(pageOffset(query.page, query.pageSize))
    ])

    return c.json({
      items: rows.map((row) => {
        const canViewProblem = isAdmin || (row.problemVisible && !row.problemDeletedAt)
        return formatSubmissionListItem(row, {
          canViewProblem,
          cropped: cropSubmissionResult(row, { isAdmin, viewerUserId: authUser?.id })
        })
      }),
      page: query.page,
      pageSize: query.pageSize,
      total
    })
  })

  app.post('/api/submissions', authMiddleware, async (c) => {
    const user = await requireAuthUser(c)
    const userLimited = await checkRateLimit(c, 'submission:create:user', `user:${user.id}`, 30, 60 * 1000)
    if (userLimited) return userLimited
    const ipLimited = await checkRateLimit(c, 'submission:create:ip', clientIp(c), 120, 60 * 1000)
    if (ipLimited) return ipLimited

    const body = submitSchema.parse(await c.req.json())
    const settings = await getRuntimeSettings()

    const [language] = await db
      .select({ id: schema.languages.id })
      .from(schema.languages)
      .where(eq(schema.languages.id, body.languageId))
      .limit(1)

    if (!language) {
      return apiError(c, 400, 'LANGUAGE_DISABLED', 'Language is not enabled')
    }

    const [target] = await db
      .select({
        problemId: schema.problems.id,
        visible: schema.problems.visible,
        deletedAt: schema.problems.deletedAt
      })
      .from(schema.problems)
      .where(eq(schema.problems.id, body.problemId))
      .limit(1)
    if (!target) {
      return notFound(c, 'Problem does not exist')
    }

    if (!target.visible || target.deletedAt) {
      return notFound(c)
    }
    const contestId = body.contestId ?? await chooseActiveContest(body.problemId)
    if (contestId) {
      const contestCheck = await validateContestSubmission(contestId, body.problemId)
      if (contestCheck) return apiError(c, contestCheck.status, contestCheck.code, contestCheck.message)
    }
    const assignmentId = body.assignmentId ?? await chooseActiveAssignment(user.id, body.problemId)
    if (assignmentId) {
      const assignmentCheck = await validateAssignmentSubmission(user.id, assignmentId, body.problemId)
      if (assignmentCheck) return apiError(c, assignmentCheck.status, assignmentCheck.code, assignmentCheck.message)
    }

    const [submission] = await db
      .insert(schema.submissions)
      .values({
        userId: user.id,
        problemId: body.problemId,
        languageId: body.languageId,
        code: body.code,
        public: body.public ?? settings.general.publicCode,
        contestId: contestId ?? null,
        assignmentId: assignmentId ?? null
      })
      .returning()
    await recordSubmissionCreated(submission)
    await enqueueJudgeTask(submission.id)
    return c.json({ id: submission.id, status: submission.status, createdAt: submission.createdAt.toISOString() }, 201)
  })

  app.get('/api/submissions/:id', async (c) => {
    const denied = await denyGuestAccess(c, 'Sign in to view submissions')
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const [submission] = await db
      .select({
        id: schema.submissions.id,
        userId: schema.submissions.userId,
        userName: schema.users.name,
        userEmail: schema.users.email,
        problemId: schema.submissions.problemId,
        problemTitle: schema.problems.title,
        languageId: schema.submissions.languageId,
        code: schema.submissions.code,
        public: schema.submissions.public,
        status: schema.submissions.status,
        timeMs: schema.submissions.timeMs,
        memoryBytes: schema.submissions.memoryBytes,
        score: schema.submissions.score,
        message: schema.submissions.message,
        contestId: schema.submissions.contestId,
        assignmentId: schema.submissions.assignmentId,
        createdAt: schema.submissions.createdAt,
        updatedAt: schema.submissions.updatedAt,
        problemVisible: schema.problems.visible,
        problemDeletedAt: schema.problems.deletedAt,
        contestType: schema.contests.type,
        contestStartAt: schema.contests.startAt,
        contestEndAt: schema.contests.endAt,
        contestFreezeAt: schema.contests.freezeAt
      })
      .from(schema.submissions)
      .innerJoin(schema.problems, eq(schema.submissions.problemId, schema.problems.id))
      .innerJoin(schema.users, eq(schema.submissions.userId, schema.users.id))
      .leftJoin(schema.contests, eq(schema.submissions.contestId, schema.contests.id))
      .where(eq(schema.submissions.id, id))
      .limit(1)

    if (!submission) return notFound(c)
    const authUser = await getOptionalAuthUser(c)
    const isOwnerOrAdmin = submission.userId === authUser?.id || authUser?.admin === true
    if (!canViewSubmissionRow(submission, authUser)) return notFound(c)
    const isAdmin = authUser?.admin === true
    const canViewProblem = isAdmin || (submission.problemVisible && !submission.problemDeletedAt)
    if (submission.assignmentId && !isOwnerOrAdmin) return notFound(c)

    const cropped = cropSubmissionResult(submission, { isAdmin, viewerUserId: authUser?.id })
    const canInspectSource = (!submission.contestId && submission.public) || isOwnerOrAdmin

    const cases = !cropped
      ? await db
          .select()
          .from(schema.submissionCases)
          .where(eq(schema.submissionCases.submissionId, submission.id))
          .orderBy(asc(schema.submissionCases.caseNo))
      : []

    return c.json({
      ...formatSubmissionListItem(submission, { canViewProblem, cropped }),
      code: canInspectSource ? submission.code : null,
      message: cropped ? null : submission.message,
      cases: !cropped ? cases.map((item) => ({
        caseNo: item.caseNo,
        status: item.status,
        timeMs: item.timeMs,
        memoryBytes: item.memoryBytes,
        score: item.score,
        message: item.message || null
      })) : [],
      canCoach: !cropped && isOwnerOrAdmin && await canCoachSubmission(submission.status),
      judgeProgress: !cropped ? await getJudgeProgress(submission.id) : null
    })
  })

  app.post('/api/submissions/:id/coach', authMiddleware, async (c) => {
    const user = await requireAuthUser(c)
    const rateLimited = await checkRateLimit(
      c,
      'submission:coach',
      `user:${user.id}`,
      20,
      60 * 60 * 1000
    )
    if (rateLimited) return rateLimited

    const id = numericId.parse(c.req.param('id'))
    const [submission] = await db
      .select({
        id: schema.submissions.id,
        userId: schema.submissions.userId,
        languageId: schema.submissions.languageId,
        code: schema.submissions.code,
        public: schema.submissions.public,
        status: schema.submissions.status,
        message: schema.submissions.message,
        contestId: schema.submissions.contestId,
        assignmentId: schema.submissions.assignmentId,
        problemVisible: schema.problems.visible,
        problemDeletedAt: schema.problems.deletedAt,
        contestType: schema.contests.type,
        contestStartAt: schema.contests.startAt,
        contestEndAt: schema.contests.endAt,
        contestFreezeAt: schema.contests.freezeAt
      })
      .from(schema.submissions)
      .innerJoin(schema.problems, eq(schema.submissions.problemId, schema.problems.id))
      .leftJoin(schema.contests, eq(schema.submissions.contestId, schema.contests.id))
      .where(eq(schema.submissions.id, id))
      .limit(1)

    if (!submission) return notFound(c)
    if (!canViewSubmissionRow(submission, user)) return notFound(c)
    if (submission.userId !== user.id && !user.admin) {
      return apiError(c, 403, 'FORBIDDEN', 'Cannot coach another user submission')
    }
    if (cropSubmissionResult(submission, { isAdmin: user.admin, viewerUserId: user.id })) {
      return apiError(c, 403, 'FORBIDDEN', 'AI coaching is unavailable for cropped submissions')
    }
    const settings = await getRuntimeSettings()
    if (!settings.ai.enabled) {
      return apiError(c, 403, 'AI_DISABLED', 'AI coaching is disabled')
    }
    if (submission.assignmentId) {
      // Assignment-level AI switches are intentionally centralized in site settings.
    }
    if (!isTerminalStatus(submission.status)) {
      return apiError(c, 400, 'AI_COACHING_UNAVAILABLE', `AI coaching is unavailable for ${submission.status} submissions`)
    }

    const coaching = await createCoachingResponse({
      status: submission.status,
      message: submission.message,
      languageId: submission.languageId,
      sourceCode: submission.code
    })

    return c.json(coaching, 201)
  })
}

function visibleSubmissionListWhere(
  userId: number | undefined,
  isAdmin: boolean,
  filters: z.infer<typeof submissionListQuerySchema>
) {
  const base = isAdmin ? undefined : and(eq(schema.problems.visible, true), isNull(schema.problems.deletedAt))
  const filter = and(
    base,
    filters.status ? eq(schema.submissions.status, filters.status) : undefined,
    filters.problemId ? eq(schema.submissions.problemId, filters.problemId) : undefined,
    filters.userId ? eq(schema.submissions.userId, filters.userId) : undefined,
    filters.languageId ? eq(schema.submissions.languageId, filters.languageId) : undefined,
    filters.contestId ? eq(schema.submissions.contestId, filters.contestId) : undefined,
    filters.assignmentId ? eq(schema.submissions.assignmentId, filters.assignmentId) : undefined
  )
  if (isAdmin) return filter
  const assignmentScope = userId
    ? or(isNull(schema.submissions.assignmentId), eq(schema.submissions.userId, userId))
    : isNull(schema.submissions.assignmentId)
  return and(filter, assignmentScope)
}

async function countSubmissionRows(where: ReturnType<typeof visibleSubmissionListWhere>) {
  const [row] = await db
    .select({ total: sql<number>`count(*)::int` })
    .from(schema.submissions)
    .innerJoin(schema.problems, eq(schema.submissions.problemId, schema.problems.id))
    .where(where)
  return row?.total ?? 0
}

function formatSubmissionListItem(row: {
  id: number
  userId: number
  userName: string
  userEmail: string
  problemId: number
  problemTitle: string
  languageId: string
  status: z.infer<typeof judgeStatusSchema>
  timeMs: number
  memoryBytes: number
  score: number
  public: boolean
  contestId: number | null
  assignmentId: number | null
  createdAt: Date
  updatedAt: Date
}, options: { canViewProblem: boolean; cropped: CroppedSubmission | null }) {
  const cropped = options.cropped
  return {
    id: row.id,
    problem: options.canViewProblem ? { id: row.problemId, title: row.problemTitle } : null,
    user: {
      id: row.userId,
      name: row.userName,
      avatarUrl: gravatarUrl(row.userEmail)
    },
    languageId: row.languageId,
    status: cropped ? null : row.status,
    displayStatus: cropped?.displayStatus ?? row.status,
    score: cropped ? null : row.score,
    timeMs: cropped ? null : row.timeMs,
    memoryBytes: cropped ? null : row.memoryBytes,
    public: row.public,
    contestId: row.contestId,
    assignmentId: row.assignmentId,
    createdAt: row.createdAt.toISOString(),
    updatedAt: row.updatedAt.toISOString()
  }
}

type CroppedSubmission = { displayStatus: z.infer<typeof judgeStatusSchema> | 'SUBMITTED' | 'JUDGED' }

function cropSubmissionResult(row: {
  userId: number
  contestId: number | null
  contestType?: 'OI' | 'ICPC' | null
  contestStartAt?: Date | null
  contestEndAt?: Date | null
  contestFreezeAt?: Date | null
  status: z.infer<typeof judgeStatusSchema>
}, viewer: { isAdmin: boolean; viewerUserId?: number }): CroppedSubmission | null {
  if (viewer.isAdmin || !row.contestId) return null
  const now = new Date()
  if (row.contestType === 'OI' && row.contestStartAt && row.contestEndAt && now >= row.contestStartAt && now < row.contestEndAt) {
    return { displayStatus: oiDisplayStatus(row.status) }
  }
  const isOwner = row.userId === viewer.viewerUserId
  if (
    row.contestType === 'ICPC' &&
    !isOwner &&
    row.contestFreezeAt &&
    row.contestEndAt &&
    now >= row.contestFreezeAt &&
    now < row.contestEndAt
  ) {
    return { displayStatus: row.status === 'WAITING' ? 'SUBMITTED' : row.status === 'JUDGING' ? 'JUDGING' : 'JUDGED' }
  }
  return null
}

function oiDisplayStatus(status: z.infer<typeof judgeStatusSchema>) {
  if (status === 'WAITING') return 'SUBMITTED'
  if (status === 'JUDGING') return 'JUDGING'
  return 'JUDGED'
}

function canViewSubmissionRow(row: {
  userId: number
  assignmentId: number | null
  problemVisible: boolean
  problemDeletedAt: Date | null
}, authUser: Awaited<ReturnType<typeof getOptionalAuthUser>>) {
  const isAdmin = authUser?.admin === true
  const isOwner = row.userId === authUser?.id
  if ((!row.problemVisible || row.problemDeletedAt) && !isAdmin && !isOwner) return false
  if (row.assignmentId && !isAdmin && !isOwner) return false
  return true
}

async function canCoachSubmission(status: z.infer<typeof judgeStatusSchema>) {
  const settings = await getRuntimeSettings()
  return settings.ai.enabled && isTerminalStatus(status)
}

function isTerminalStatus(status: z.infer<typeof judgeStatusSchema>) {
  return !['WAITING', 'JUDGING'].includes(status)
}

async function chooseActiveContest(problemId: number) {
  const [row] = await db
    .select({ id: schema.contests.id })
    .from(schema.contestProblems)
    .innerJoin(schema.contests, eq(schema.contestProblems.contestId, schema.contests.id))
    .where(and(
      eq(schema.contestProblems.problemId, problemId),
      isNull(schema.contests.deletedAt),
      sql`${schema.contests.startAt} <= now()`,
      sql`${schema.contests.endAt} > now()`
    ))
    .orderBy(desc(schema.contests.startAt), desc(schema.contests.id))
    .limit(1)
  return row?.id ?? null
}

async function chooseActiveAssignment(userId: number, problemId: number) {
  const [row] = await db
    .select({ id: schema.assignments.id })
    .from(schema.assignmentProblems)
    .innerJoin(schema.assignments, eq(schema.assignmentProblems.assignmentId, schema.assignments.id))
    .leftJoin(schema.assignmentGroups, eq(schema.assignmentGroups.assignmentId, schema.assignments.id))
    .leftJoin(schema.userGroups, and(eq(schema.userGroups.groupId, schema.assignmentGroups.groupId), eq(schema.userGroups.userId, userId)))
    .leftJoin(schema.assignmentUsers, and(eq(schema.assignmentUsers.assignmentId, schema.assignments.id), eq(schema.assignmentUsers.userId, userId)))
    .where(and(
      eq(schema.assignmentProblems.problemId, problemId),
      isNull(schema.assignments.deletedAt),
      sql`${schema.assignments.endAt} > now()`,
      or(eq(schema.userGroups.userId, userId), eq(schema.assignmentUsers.userId, userId))
    ))
    .orderBy(asc(schema.assignments.endAt), asc(schema.assignments.id))
    .limit(1)
  return row?.id ?? null
}

function gravatarUrl(email: string) {
  const hash = createHash('md5').update(email.trim().toLowerCase()).digest('hex')
  return `https://www.gravatar.com/avatar/${hash}?d=identicon&s=80`
}
