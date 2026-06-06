import { Hono } from 'hono'
import { asc, desc, eq, sql } from 'drizzle-orm'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { getOptionalAuthUser } from '../auth'
import { getUserAssignments } from '../services/assignments'
import { getRecentTopics } from '../services/discussion'
import { countRows, countVisibleSubmissions } from '../services/stats'

export function registerPublicRoutes(app: Hono) {
  app.get('/api/dashboard', async (c) => {
    const authUser = await getOptionalAuthUser(c)
    const isAdmin = authUser?.groups.includes('admin') ?? false
    // The aggregate stat board is an operator-facing overview; only admins see it.
    const [problemStats, submissionStats, contestStats, userStats, assignmentStats] =
      await Promise.all([
        isAdmin
          ? countRows(schema.problems, sql`${schema.problems.visible} = true`)
          : Promise.resolve(null),
        isAdmin ? countVisibleSubmissions() : Promise.resolve(null),
        isAdmin ? countRows(schema.contests) : Promise.resolve(null),
        isAdmin
          ? countRows(schema.users, sql`${schema.users.disabledAt} is null`)
          : Promise.resolve(null),
        isAdmin ? countRows(schema.assignments) : Promise.resolve(null)
      ])

    const stats: Record<string, number> = {}
    if (problemStats !== null) stats.problems = problemStats
    if (submissionStats !== null) stats.submissions = submissionStats
    if (contestStats !== null) stats.contests = contestStats
    if (assignmentStats !== null) stats.assignments = assignmentStats
    if (userStats !== null) stats.users = userStats

    const [recentSubmissions, recentProblems, recentTopics, recentContests, myAssignments] =
      await Promise.all([
        db
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
          .limit(8),
        db
          .select({
            id: schema.problems.id,
            title: schema.problems.title,
            tags: schema.problems.tags,
            solvedCount: schema.problems.solvedCount,
            createdAt: schema.problems.createdAt
          })
          .from(schema.problems)
          .where(
            authUser
              ? sql`${schema.problems.visible} = true and not exists (
                  select 1 from ${schema.solvedProblems}
                  where ${schema.solvedProblems.userId} = ${authUser.id}
                    and ${schema.solvedProblems.problemId} = ${schema.problems.id}
                )`
              : eq(schema.problems.visible, true)
          )
          .orderBy(desc(schema.problems.createdAt), desc(schema.problems.id))
          .limit(6),
        getRecentTopics(6),
        db
          .select({
            id: schema.contests.id,
            title: schema.contests.title,
            type: schema.contests.type,
            startAt: schema.contests.startAt,
            endAt: schema.contests.endAt
          })
          .from(schema.contests)
          .orderBy(desc(schema.contests.startAt), desc(schema.contests.createdAt))
          .limit(5),
        authUser ? getUserAssignments(authUser.id, 5) : Promise.resolve([])
      ])

    return c.json({
      stats,
      recentSubmissions,
      recentProblems,
      recentTopics,
      recentContests,
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
}
