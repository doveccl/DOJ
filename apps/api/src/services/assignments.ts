import { and, desc, eq, inArray, sql } from 'drizzle-orm'
import { db, schema } from '@doj/db/client'
import type { JudgeStatus } from '@doj/shared/status'

export async function getAssignmentDetail(id: number) {
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

export async function getAssignmentReport(id: number) {
  const detail = await getAssignmentDetail(id)
  if (!detail) return null

  const students = await db
    .selectDistinct({
      id: schema.users.id,
      name: schema.users.name,
      email: schema.users.email
    })
    .from(schema.assignmentGroups)
    .innerJoin(schema.userGroups, eq(schema.userGroups.groupId, schema.assignmentGroups.groupId))
    .innerJoin(schema.users, eq(schema.users.id, schema.userGroups.userId))
    .where(eq(schema.assignmentGroups.assignmentId, id))
    .orderBy(schema.users.name)

  const problemIds = detail.problems.map((problem) => problem.id)
  const studentIds = students.map((student) => student.id)
  const aggregates =
    problemIds.length && studentIds.length
      ? await db
          .select({
            userId: schema.submissions.userId,
            problemId: schema.submissions.problemId,
            attempts: sql<number>`count(*)::int`,
            status: sql<JudgeStatus>`coalesce(
              (array_agg(${schema.submissions.status} order by ${schema.submissions.createdAt} desc, ${schema.submissions.id} desc)
                filter (where ${schema.submissions.status} = 'AC'))[1],
              (array_agg(${schema.submissions.status} order by ${schema.submissions.createdAt} desc, ${schema.submissions.id} desc))[1]
            )`,
            bestSubmissionId: sql<number>`coalesce(
              (array_agg(${schema.submissions.id} order by ${schema.submissions.createdAt} desc, ${schema.submissions.id} desc)
                filter (where ${schema.submissions.status} = 'AC'))[1],
              (array_agg(${schema.submissions.id} order by ${schema.submissions.createdAt} desc, ${schema.submissions.id} desc))[1]
            )`,
            lastSubmissionId: sql<number>`(array_agg(${schema.submissions.id} order by ${schema.submissions.createdAt} desc, ${schema.submissions.id} desc))[1]`,
            updatedAt: sql<Date>`max(${schema.submissions.createdAt})`
          })
          .from(schema.submissions)
          .where(
            and(
              eq(schema.submissions.assignmentId, id),
              inArray(schema.submissions.problemId, problemIds),
              inArray(schema.submissions.userId, studentIds)
            )
          )
          .groupBy(schema.submissions.userId, schema.submissions.problemId)
      : []

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

  for (const aggregate of aggregates) {
    const row = rowByUser.get(aggregate.userId)
    if (!row) continue
    const cell = row.problems[String(aggregate.problemId)]
    if (!cell) continue

    cell.attempts = aggregate.attempts
    cell.status = aggregate.status
    cell.bestSubmissionId = aggregate.bestSubmissionId
    cell.lastSubmissionId = aggregate.lastSubmissionId
    cell.updatedAt =
      aggregate.updatedAt instanceof Date
        ? aggregate.updatedAt.toISOString()
        : new Date(aggregate.updatedAt).toISOString()
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

export async function getUserAssignments(userId: number, limit: number) {
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

export async function getUserAssignmentDetail(userId: number, assignmentId: number) {
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

export async function validateAssignmentSubmission(
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
