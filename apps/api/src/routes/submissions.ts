import { Hono } from 'hono'
import { and, asc, desc, eq, isNull, or, sql } from 'drizzle-orm'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { enqueueJudgeTask } from '@doj/db/queue'
import { createCoachingResponse } from '../ai'
import { authMiddleware, getOptionalAuthUser, requireAuthUser } from '../auth'
import { checkRateLimit } from '../rate-limit'
import { validateAssignmentSubmission } from '../services/assignments'
import { validateContestSubmission } from '../services/contests'
import { getRuntimeSettings } from '../settings'
import { numericId } from '../validation'

const submitSchema = z.object({
  problemId: numericId,
  languageId: z.string().min(1).max(64),
  sourceCode: z
    .string()
    .min(1)
    .max(200 * 1024),
  open: z.boolean().optional(),
  contestId: numericId.optional(),
  assignmentId: numericId.optional()
})

export function registerSubmissionRoutes(app: Hono) {
  app.get('/api/submissions', async (c) => {
    const query = z
      .object({
        page: z.coerce.number().int().positive().default(1),
        pageSize: z.coerce.number().int().min(1).max(100).default(50)
      })
      .parse(c.req.query())
    const authUser = await getOptionalAuthUser(c)
    const isAdmin = authUser?.groups.includes('admin') ?? false
    const where = visibleSubmissionListWhere(authUser?.id, isAdmin)
    const [total, rows] = await Promise.all([
      countSubmissionRows(where),
      db
        .select({
          id: schema.submissions.id,
          userId: schema.submissions.userId,
          userName: schema.users.name,
          problemId: schema.submissions.problemId,
          problemTitle: schema.problems.title,
          languageId: schema.submissions.languageId,
          status: schema.submissions.status,
          timeMs: schema.submissions.timeMs,
          memoryBytes: schema.submissions.memoryBytes,
          score: schema.submissions.score,
          message: schema.submissions.message,
          contestId: schema.submissions.contestId,
          assignmentId: schema.submissions.assignmentId,
          createdAt: schema.submissions.createdAt,
          updatedAt: schema.submissions.updatedAt
        })
        .from(schema.submissions)
        .innerJoin(schema.problems, eq(schema.submissions.problemId, schema.problems.id))
        .innerJoin(schema.users, eq(schema.submissions.userId, schema.users.id))
        .where(where)
        .orderBy(desc(schema.submissions.createdAt))
        .limit(query.pageSize)
        .offset((query.page - 1) * query.pageSize)
    ])
    const list = rows.map((row) => ({
      ...row,
      // The judge message can contain compiler output; only the owner/admin may read it.
      message: isAdmin || row.userId === authUser?.id ? row.message : ''
    }))

    return c.json({ total, page: query.page, pageSize: query.pageSize, list })
  })

  app.post('/api/submissions', authMiddleware, async (c) => {
    const user = await requireAuthUser(c)
    const rateLimited = await checkRateLimit(
      c,
      'submission:create',
      `user:${user.id}`,
      120,
      10 * 60 * 1000
    )
    if (rateLimited) return rateLimited

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
        visible: schema.problems.visible
      })
      .from(schema.problems)
      .where(eq(schema.problems.id, body.problemId))
      .limit(1)
    if (!target) {
      return c.json(
        {
          code: 'PROBLEM_NOT_FOUND',
          message: 'Problem does not exist'
        },
        400
      )
    }

    if (body.contestId) {
      const contestCheck = await validateContestSubmission(body.contestId, body.problemId)
      if (contestCheck)
        return c.json(
          { code: contestCheck.code, message: contestCheck.message },
          contestCheck.status
        )
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
        languageId: schema.submissions.languageId,
        sourceCode: schema.submissions.sourceCode,
        open: schema.submissions.open,
        status: schema.submissions.status,
        timeMs: schema.submissions.timeMs,
        memoryBytes: schema.submissions.memoryBytes,
        score: schema.submissions.score,
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
    const isOwnerOrAdmin =
      submission.userId === authUser?.id || authUser?.groups.includes('admin') === true
    const canManageHiddenProblem =
      submission.userId === authUser?.id || authUser?.groups.includes('admin') === true
    if (!submission.problemVisible && !canManageHiddenProblem) return c.notFound()
    if (submission.assignmentId && !isOwnerOrAdmin) return c.notFound()

    const canInspect = !submission.contestId || isOwnerOrAdmin
    const canInspectSource = (!submission.contestId && submission.open) || isOwnerOrAdmin
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
      sourceCode: canInspectSource ? submission.sourceCode : '',
      message: canInspect ? submission.message : '',
      cases,
      restricted: !canInspect,
      sourceRestricted: !canInspectSource
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
          provider: settings.aiProvider
        }
      })
      .returning()

    return c.json(session, 201)
  })
}

function visibleSubmissionListWhere(userId: number | undefined, isAdmin: boolean) {
  const base = eq(schema.problems.visible, true)
  if (isAdmin) return base
  const assignmentScope = userId
    ? or(isNull(schema.submissions.assignmentId), eq(schema.submissions.userId, userId))
    : isNull(schema.submissions.assignmentId)
  return and(base, assignmentScope)
}

async function countSubmissionRows(where: ReturnType<typeof visibleSubmissionListWhere>) {
  const [row] = await db
    .select({ total: sql<number>`count(*)::int` })
    .from(schema.submissions)
    .innerJoin(schema.problems, eq(schema.submissions.problemId, schema.problems.id))
    .where(where)
  return row?.total ?? 0
}
