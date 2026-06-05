import { and, asc, eq } from 'drizzle-orm'
import { db, schema } from '@doj/db/client'

export async function getContestDetail(id: number) {
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

export async function getContestScoreboard(id: number, options: { reveal?: boolean } = {}) {
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

export async function validateContestSubmission(contestId: number, problemId: number) {
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
