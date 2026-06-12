import { createHash } from 'node:crypto'
import { and, desc, eq, inArray, isNull, or, sql } from 'drizzle-orm'
import { db, schema } from '@doj/db/client'

export async function getAssignmentDetail(id: number) {
  const [assignment] = await db
    .select()
    .from(schema.assignments)
    .where(eq(schema.assignments.id, id))
    .limit(1)
  if (!assignment) return null

  const [groups, users, problems] = await Promise.all([
    db
      .select({
        id: schema.groups.id,
        name: schema.groups.name
      })
      .from(schema.assignmentGroups)
      .innerJoin(schema.groups, eq(schema.assignmentGroups.groupId, schema.groups.id))
      .where(eq(schema.assignmentGroups.assignmentId, id))
      .orderBy(schema.groups.name),
    db
      .select({
        id: schema.users.id,
        name: schema.users.name,
        email: schema.users.email
      })
      .from(schema.assignmentUsers)
      .innerJoin(schema.users, eq(schema.assignmentUsers.userId, schema.users.id))
      .where(eq(schema.assignmentUsers.assignmentId, id))
      .orderBy(schema.users.name),
    db
      .select({
        id: schema.problems.id,
        title: schema.problems.title,
        visible: schema.problems.visible,
        deletedAt: schema.problems.deletedAt,
        sort: schema.assignmentProblems.sort
      })
      .from(schema.assignmentProblems)
      .leftJoin(schema.problems, eq(schema.assignmentProblems.problemId, schema.problems.id))
      .where(eq(schema.assignmentProblems.assignmentId, id))
      .orderBy(schema.assignmentProblems.sort)
  ])

  return {
    assignment,
    groups,
    users,
    problems
  }
}

export async function getAdminAssignments(
  limit: number,
  offset = 0,
  status?: 'current' | 'past'
) {
  const where = assignmentStatusScope(status)
  const [totalRow, assignments] = await Promise.all([
    db
      .select({ total: sql<number>`count(*)::int` })
      .from(schema.assignments)
      .where(where),
    db
      .select()
      .from(schema.assignments)
      .where(where)
      .orderBy(desc(schema.assignments.createdAt))
      .limit(limit)
      .offset(offset)
  ])

  const items = await Promise.all(
    assignments.map(async (assignment) => ({
      ...assignment,
      ...(await getAdminAssignmentOverview(assignment.id))
    }))
  )

  return { items, total: totalRow[0]?.total ?? 0 }
}

export async function getAssignmentReport(id: number) {
  const detail = await getAssignmentDetail(id)
  if (!detail) return null

  const students = await getAssignmentStudents(id)
  const rows = await buildAssignmentRows(id, detail.problems, students)

  return {
    assignment: detail.assignment,
    problems: detail.problems.map(formatAssignmentProblem),
    rows
  }
}

async function getAssignmentStudents(id: number) {
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

  const directStudents = await db
    .selectDistinct({
      id: schema.users.id,
      name: schema.users.name,
      email: schema.users.email
    })
    .from(schema.assignmentUsers)
    .innerJoin(schema.users, eq(schema.users.id, schema.assignmentUsers.userId))
    .where(eq(schema.assignmentUsers.assignmentId, id))

  for (const student of directStudents) {
    if (!students.some((item) => item.id === student.id)) students.push(student)
  }
  students.sort((left, right) => left.name.localeCompare(right.name) || left.id - right.id)
  return students
}

async function buildAssignmentRows(
  assignmentId: number,
  problems: Array<{ id: number | null; title: string | null; visible: boolean | null; deletedAt: Date | null; sort: number }>,
  students: Array<{ id: number; name: string; email: string }>
) {
  const problemIds = problems.map((problem) => problem.id).filter((id): id is number => id !== null)
  const studentIds = students.map((student) => student.id)
  const aggregates =
    problemIds.length && studentIds.length
      ? await db
          .select({
            userId: schema.submissions.userId,
            problemId: schema.submissions.problemId,
            attempts: sql<number>`count(*)::int`,
            bestScore: sql<number>`coalesce(max(${schema.submissions.score}), 0)::int`,
            ac: sql<boolean>`bool_or(${schema.submissions.status} = 'AC')`,
            submissionId: sql<number | null>`(array_agg(${schema.submissions.id} order by ${schema.submissions.score} desc, ${schema.submissions.createdAt} desc, ${schema.submissions.id} desc))[1]`,
            submittedAt: sql<Date | null>`(array_agg(${schema.submissions.createdAt} order by ${schema.submissions.score} desc, ${schema.submissions.createdAt} desc, ${schema.submissions.id} desc))[1]`
          })
          .from(schema.submissions)
          .where(
            and(
              eq(schema.submissions.assignmentId, assignmentId),
              inArray(schema.submissions.problemId, problemIds),
              inArray(schema.submissions.userId, studentIds)
            )
          )
          .groupBy(schema.submissions.userId, schema.submissions.problemId)
      : []

  const rows = students.map((student) => ({
    user: {
      id: student.id,
      name: student.name,
      avatarUrl: gravatarUrl(student.email)
    },
    problems: Object.fromEntries(
      problems.map((problem) => [String(problem.id), emptyAssignmentProgress(problem)])
    )
  }))
  const rowByUser = new Map(rows.map((row) => [row.user.id, row]))

  for (const aggregate of aggregates) {
    const row = rowByUser.get(aggregate.userId)
    if (!row) continue
    const cell = row.problems[String(aggregate.problemId)]
    if (!cell) continue

    cell.attempts = aggregate.attempts
    cell.bestScore = aggregate.bestScore
    cell.ac = aggregate.ac
    cell.submissionId = aggregate.submissionId
    cell.submittedAt = aggregate.submittedAt ? new Date(aggregate.submittedAt).toISOString() : null
  }

  return rows
}

export async function getUserAssignments(
  userId: number,
  limit: number,
  offset = 0,
  status?: 'current' | 'past'
) {
  const scope = userAssignmentScope(userId, status)
  const [totalRow, items] = await Promise.all([
    db
      .select({ total: sql<number>`count(distinct ${schema.assignments.id})::int` })
      .from(schema.assignments)
      .leftJoin(
        schema.assignmentGroups,
        eq(schema.assignmentGroups.assignmentId, schema.assignments.id)
      )
      .leftJoin(
        schema.userGroups,
        and(
          eq(schema.userGroups.groupId, schema.assignmentGroups.groupId),
          eq(schema.userGroups.userId, userId)
        )
      )
      .leftJoin(
        schema.assignmentUsers,
        and(
          eq(schema.assignmentUsers.assignmentId, schema.assignments.id),
          eq(schema.assignmentUsers.userId, userId)
        )
      )
      .where(scope),
    db
      .selectDistinct({
        id: schema.assignments.id,
        title: schema.assignments.title,
        description: schema.assignments.description,
        endAt: schema.assignments.endAt,
        createdAt: schema.assignments.createdAt
      })
      .from(schema.assignments)
      .leftJoin(
        schema.assignmentGroups,
        eq(schema.assignmentGroups.assignmentId, schema.assignments.id)
      )
      .leftJoin(
        schema.userGroups,
        and(
          eq(schema.userGroups.groupId, schema.assignmentGroups.groupId),
          eq(schema.userGroups.userId, userId)
        )
      )
      .leftJoin(
        schema.assignmentUsers,
        and(
          eq(schema.assignmentUsers.assignmentId, schema.assignments.id),
          eq(schema.assignmentUsers.userId, userId)
        )
      )
      .where(scope)
      .orderBy(desc(schema.assignments.createdAt))
      .limit(limit)
      .offset(offset)
  ])

  const itemsWithProgress = await Promise.all(
    items.map(async (item) => ({
      ...item,
      ...(await getUserAssignmentOverview(item.id, userId))
    }))
  )

  return { items: itemsWithProgress, total: totalRow[0]?.total ?? 0 }
}

function assignmentStatusScope(status?: 'current' | 'past') {
  return and(
    isNull(schema.assignments.deletedAt),
    status === 'current' ? sql`${schema.assignments.endAt} > now()` : undefined,
    status === 'past' ? sql`${schema.assignments.endAt} <= now()` : undefined
  )
}

function userAssignmentScope(userId: number, status?: 'current' | 'past') {
  return and(
    isNull(schema.assignments.deletedAt),
    status === 'current' ? sql`${schema.assignments.endAt} > now()` : undefined,
    status === 'past' ? sql`${schema.assignments.endAt} <= now()` : undefined,
    or(eq(schema.userGroups.userId, userId), eq(schema.assignmentUsers.userId, userId))
  )
}

async function getUserAssignmentOverview(assignmentId: number, userId: number) {
  const [row] = await db
    .select({
      total: sql<number>`count(distinct ${schema.assignmentProblems.problemId})::int`,
      completed: sql<number>`count(distinct ${schema.assignmentProblems.problemId}) filter (where ${schema.submissions.status} = 'AC')::int`
    })
    .from(schema.assignmentProblems)
    .leftJoin(
      schema.submissions,
      and(
        eq(schema.submissions.assignmentId, assignmentId),
        eq(schema.submissions.problemId, schema.assignmentProblems.problemId),
        eq(schema.submissions.userId, userId)
      )
    )
    .where(eq(schema.assignmentProblems.assignmentId, assignmentId))

  return {
    total: row?.total ?? 0,
    completed: row?.completed ?? 0
  }
}

async function getAdminAssignmentOverview(assignmentId: number) {
  const detail = await getAssignmentDetail(assignmentId)
  if (!detail) return { total: 0, completed: 0, assigned: 0 }

  const students = await getAssignmentStudents(assignmentId)
  const rows = await buildAssignmentRows(assignmentId, detail.problems, students)
  const totalProblems = detail.problems.length
  const completedStudents =
    totalProblems === 0
      ? 0
      : rows.filter((row) => Object.values(row.problems).every((problem) => problem.ac)).length

  return {
    total: students.length,
    completed: completedStudents,
    assigned: students.length,
    problemCount: totalProblems
  }
}

export async function getUserAssignmentDetail(userId: number, assignmentId: number) {
  const [match] = await db
    .select({ id: schema.assignments.id })
    .from(schema.assignments)
    .leftJoin(
      schema.assignmentGroups,
      eq(schema.assignmentGroups.assignmentId, schema.assignments.id)
    )
    .leftJoin(
      schema.userGroups,
      and(
        eq(schema.userGroups.groupId, schema.assignmentGroups.groupId),
        eq(schema.userGroups.userId, userId)
      )
    )
    .leftJoin(
      schema.assignmentUsers,
      and(
        eq(schema.assignmentUsers.assignmentId, schema.assignments.id),
        eq(schema.assignmentUsers.userId, userId)
      )
    )
    .where(
      and(
        eq(schema.assignments.id, assignmentId),
        isNull(schema.assignments.deletedAt),
        or(eq(schema.userGroups.userId, userId), eq(schema.assignmentUsers.userId, userId))
      )
    )
    .limit(1)

  if (!match) return null
  const detail = await getAssignmentDetail(assignmentId)
  if (!detail) return null
  const rows = await buildAssignmentRows(assignmentId, detail.problems, [{ id: userId, name: '', email: '' }])
  return {
    id: detail.assignment.id,
    title: detail.assignment.title,
    description: detail.assignment.description,
    endAt: detail.assignment.endAt.toISOString(),
    problems: Object.values(rows[0]?.problems ?? {})
  }
}

export async function validateAssignmentSubmission(
  userId: number,
  assignmentId: number,
  problemId: number
) {
  const [assignment] = await db
    .select({
      id: schema.assignments.id,
      endAt: schema.assignments.endAt
    })
    .from(schema.assignments)
    .leftJoin(
      schema.assignmentGroups,
      eq(schema.assignmentGroups.assignmentId, schema.assignments.id)
    )
    .leftJoin(
      schema.userGroups,
      and(
        eq(schema.userGroups.groupId, schema.assignmentGroups.groupId),
        eq(schema.userGroups.userId, userId)
      )
    )
    .leftJoin(
      schema.assignmentUsers,
      and(
        eq(schema.assignmentUsers.assignmentId, schema.assignments.id),
        eq(schema.assignmentUsers.userId, userId)
      )
    )
    .where(
      and(
        eq(schema.assignments.id, assignmentId),
        isNull(schema.assignments.deletedAt),
        or(eq(schema.userGroups.userId, userId), eq(schema.assignmentUsers.userId, userId))
      )
    )
    .limit(1)

  if (!assignment) {
    return {
      status: 404 as const,
      code: 'ASSIGNMENT_NOT_FOUND',
      message: 'Assignment does not exist for this user'
    }
  }

  const now = new Date()
  if (now >= assignment.endAt) {
    return {
      status: 400 as const,
      code: 'ASSIGNMENT_CLOSED',
      message: 'Assignment deadline has passed'
    }
  }

  const [assignmentProblem] = await db
    .select({ problemId: schema.assignmentProblems.problemId })
    .from(schema.assignmentProblems)
    .innerJoin(schema.problems, eq(schema.assignmentProblems.problemId, schema.problems.id))
    .where(
      and(
        eq(schema.assignmentProblems.assignmentId, assignmentId),
        eq(schema.assignmentProblems.problemId, problemId),
        eq(schema.problems.visible, true),
        isNull(schema.problems.deletedAt)
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

function emptyAssignmentProgress(problem: { id: number | null; title: string | null; visible: boolean | null; deletedAt: Date | null }) {
  return {
    problemId: problem.id,
    title: problem.title,
    unavailable: problem.id === null || problem.visible !== true || problem.deletedAt !== null,
    attempts: 0,
    bestScore: 0,
    ac: false,
    submissionId: null as number | null,
    submittedAt: null as string | null
  }
}

function formatAssignmentProblem(problem: { id: number | null; title: string | null; visible: boolean | null; deletedAt: Date | null; sort: number }) {
  return {
    problemId: problem.id,
    title: problem.title,
    unavailable: problem.id === null || problem.visible !== true || problem.deletedAt !== null,
    sort: problem.sort
  }
}

function gravatarUrl(email: string) {
  const hash = createHash('md5').update(email.trim().toLowerCase()).digest('hex')
  return `https://www.gravatar.com/avatar/${hash}?d=identicon&s=80`
}
