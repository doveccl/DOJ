import { createHash } from 'node:crypto'
import { and, asc, eq, isNull } from 'drizzle-orm'
import { db, schema } from '@doj/db/client'
import type { JudgeStatus } from '@doj/shared/status'
import { redisGet, redisSet } from '../redis'

// The public scoreboard is an anonymous hot path that many viewers refresh in
// parallel during a live contest. A short TTL collapses bursts of identical
// recomputes into one DB read every few seconds; admin reveal stays uncached.
const scoreboardCacheTtlSeconds = 5

export async function getContestDetail(id: number, options: { includeProblems?: boolean; publicView?: boolean } = {}) {
  const [contest] = await db
    .select()
    .from(schema.contests)
    .where(eq(schema.contests.id, id))
    .limit(1)
  if (!contest) return null

  const problems =
    options.includeProblems === false
      ? []
      : await db
          .select({
            id: schema.problems.id,
            problemId: schema.problems.id,
            title: schema.problems.title,
            visible: schema.problems.visible,
            deletedAt: schema.problems.deletedAt,
            key: schema.contestProblems.key,
            sort: schema.contestProblems.sort
          })
          .from(schema.contestProblems)
          .innerJoin(schema.problems, eq(schema.contestProblems.problemId, schema.problems.id))
          .where(eq(schema.contestProblems.contestId, id))
          .orderBy(schema.contestProblems.sort)

  return {
    contest,
    problems: problems.map((problem) => ({
      id: problem.id,
      problemId: problem.problemId,
      key: problem.key,
      title: options.publicView && (!problem.visible || problem.deletedAt) ? null : problem.title,
      unavailable: options.publicView && (!problem.visible || problem.deletedAt),
      sort: problem.sort
    }))
  }
}

export async function getContestScoreboard(
  id: number,
  options: { reveal?: boolean; page?: number; pageSize?: number } = {}
) {
  const reveal = options.reveal === true
  const page = options.page ?? 1
  const pageSize = options.pageSize ?? 50
  // Admin reveal must always reflect the latest state, so it bypasses the cache.
  if (reveal) return computeContestScoreboard(id, reveal, page, pageSize)

  const cacheKey = `scoreboard:${id}:${page}:${pageSize}`
  const cached = await redisGet(cacheKey)
  if (cached) return JSON.parse(cached) as ScoreboardResult

  const scoreboard = await computeContestScoreboard(id, reveal, page, pageSize)
  if (scoreboard) await redisSet(cacheKey, JSON.stringify(scoreboard), scoreboardCacheTtlSeconds)
  return scoreboard
}

type ScoreboardResult = NonNullable<Awaited<ReturnType<typeof computeContestScoreboard>>>

async function computeContestScoreboard(id: number, reveal: boolean, page: number, pageSize: number) {
  const detail = await getContestDetail(id, { publicView: !reveal })
  if (!detail) return null
  const now = new Date()
  const freezeAt = detail.contest.freezeAt
  const frozen =
    detail.contest.type === 'OI'
      ? now >= detail.contest.startAt && now < detail.contest.endAt
      : !!freezeAt && now >= freezeAt && now < detail.contest.endAt

  const submissions = await db
    .select({
      id: schema.submissions.id,
      userId: schema.submissions.userId,
      userName: schema.users.name,
      userEmail: schema.users.email,
      problemId: schema.submissions.problemId,
      status: schema.submissions.status,
      score: schema.submissions.score,
      createdAt: schema.submissions.createdAt
    })
    .from(schema.submissions)
    .innerJoin(schema.users, eq(schema.submissions.userId, schema.users.id))
    .where(eq(schema.submissions.contestId, id))
    .orderBy(asc(schema.submissions.createdAt))

  const rows =
    detail.contest.type === 'OI'
      ? buildOiRows(detail, submissions, reveal, frozen, now)
      : buildIcpcRows(detail, submissions, reveal, frozen)
  const rankedRows = assignContestRanks(rows, detail.contest.type)
  const offset = (page - 1) * pageSize
  return {
    contest: {
      id: detail.contest.id,
      type: detail.contest.type,
      startAt: detail.contest.startAt.toISOString(),
      endAt: detail.contest.endAt.toISOString(),
      freezeAt: detail.contest.freezeAt?.toISOString() ?? null,
      frozen,
      mode: reveal ? 'full' : 'public'
    },
    problems: detail.problems.map((problem) => ({
      problemId: problem.id,
      key: problem.key,
      title: problem.title ?? '',
      sort: problem.sort
    })),
    rows: rankedRows.slice(offset, offset + pageSize),
    page,
    pageSize,
    total: rankedRows.length,
    generatedAt: new Date().toISOString()
  }
}

type ContestDetail = NonNullable<Awaited<ReturnType<typeof getContestDetail>>>
type ContestSubmission = {
  id: number
  userId: number
  userName: string
  userEmail: string
  problemId: number
  status: JudgeStatus
  score: number
  createdAt: Date
}

function buildOiRows(detail: ContestDetail, submissions: ContestSubmission[], reveal: boolean, frozen: boolean, now: Date) {
  const hidePublic = !reveal && now < detail.contest.endAt
  const users = contestUsers(submissions)
  const submissionsByUserProblem = groupSubmissions(submissions)
  const rows = [...users.values()].map((user) => {
    let totalScore = 0
    let effectiveAt: Date | null = null
    const problems: Record<string, unknown> = {}
    for (const problem of detail.problems) {
      const last = submissionsByUserProblem.get(`${user.id}:${problem.id}`)?.at(-1) ?? null
      const submitted = Boolean(last)
      const running = last ? last.status === 'WAITING' || last.status === 'JUDGING' : false
      if (last && (!effectiveAt || last.createdAt > effectiveAt)) effectiveAt = last.createdAt
      if (last && !running) totalScore += last.score
      problems[problem.key] = hidePublic
        ? {
            submitted,
            pending: submitted,
            score: null,
            status: null,
            submissionId: null,
            submittedAt: null
          }
        : {
            submitted,
            pending: running,
            score: last && !running ? last.score : last ? 0 : null,
            status: last?.status ?? null,
            submissionId: last?.id ?? null,
            submittedAt: last?.createdAt.toISOString() ?? null
          }
    }
    return {
      user: briefUser(user),
      rank: hidePublic ? null : 0,
      totalScore: hidePublic ? undefined : totalScore,
      effectiveAt: hidePublic ? undefined : effectiveAt?.toISOString() ?? null,
      problems
    }
  })
  return rows.sort((left, right) => {
    if (hidePublic) return left.user.id - right.user.id
    const leftAt = left.effectiveAt ? new Date(left.effectiveAt).getTime() : Number.POSITIVE_INFINITY
    const rightAt = right.effectiveAt ? new Date(right.effectiveAt).getTime() : Number.POSITIVE_INFINITY
    return (right.totalScore ?? 0) - (left.totalScore ?? 0) || leftAt - rightAt || left.user.id - right.user.id
  })
}

function buildIcpcRows(detail: ContestDetail, submissions: ContestSubmission[], reveal: boolean, frozen: boolean) {
  const users = contestUsers(submissions)
  const submissionsByUserProblem = groupSubmissions(submissions)
  const hiddenAfter = !reveal && frozen ? detail.contest.freezeAt : null
  const rows = [...users.values()].map((user) => {
    let solved = 0
    let penalty = 0
    const problems: Record<string, unknown> = {}
    for (const problem of detail.problems) {
      const all = submissionsByUserProblem.get(`${user.id}:${problem.id}`) ?? []
      const visible = hiddenAfter ? all.filter((item) => item.createdAt < hiddenAfter) : all
      const hidden = hiddenAfter ? all.filter((item) => item.createdAt >= hiddenAfter) : []
      const cell = buildIcpcCell(detail, visible, hidden.length > 0)
      if (cell.accepted) {
        solved += 1
        penalty += cell.penalty
      }
      problems[problem.key] = cell
    }
    return {
      user: briefUser(user),
      rank: 0,
      solved,
      penalty,
      problems
    }
  })
  return rows.sort((left, right) => right.solved - left.solved || left.penalty - right.penalty || left.user.id - right.user.id)
}

function buildIcpcCell(detail: ContestDetail, visible: ContestSubmission[], hasHidden: boolean) {
  let attempts = 0
  let wrongAttempts = 0
  let acceptedAt: string | null = null
  let accepted = false
  let penalty = 0
  let submissionId: number | null = null
  let status: JudgeStatus | null = null

  for (const submission of visible) {
    if (accepted) break
    if (submission.status === 'WAITING' || submission.status === 'JUDGING') continue
    attempts += 1
    submissionId = submission.id
    status = submission.status
    if (submission.status === 'AC') {
      accepted = true
      acceptedAt = submission.createdAt.toISOString()
      penalty = Math.max(0, Math.floor((submission.createdAt.getTime() - detail.contest.startAt.getTime()) / 60000)) + wrongAttempts * 20
    } else if (['WA', 'PE', 'TLE', 'MLE', 'OLE', 'RE'].includes(submission.status)) {
      wrongAttempts += 1
    }
  }

  return {
    submitted: visible.length > 0 || hasHidden,
    pending: hasHidden || visible.some((item) => item.status === 'WAITING' || item.status === 'JUDGING'),
    accepted,
    attempts,
    wrongAttempts,
    acceptedAt,
    penalty,
    status,
    submissionId
  }
}

function assignContestRanks(rows: Array<Record<string, any>>, type: 'OI' | 'ICPC') {
  let previousKey = ''
  let previousRank = 0
  return rows.map((row, index) => {
    if (row.rank === null) return row
    const key =
      type === 'OI'
        ? `${row.totalScore ?? 0}:${row.effectiveAt ?? ''}`
        : `${row.solved ?? 0}:${row.penalty ?? 0}`
    const rank = index === 0 || key !== previousKey ? index + 1 : previousRank
    previousKey = key
    previousRank = rank
    return { ...row, rank }
  })
}

function contestUsers(submissions: ContestSubmission[]) {
  const users = new Map<number, { id: number; name: string; email: string }>()
  for (const submission of submissions) {
    users.set(submission.userId, {
      id: submission.userId,
      name: submission.userName,
      email: submission.userEmail
    })
  }
  return users
}

function groupSubmissions(submissions: ContestSubmission[]) {
  const groups = new Map<string, ContestSubmission[]>()
  for (const submission of submissions) {
    const key = `${submission.userId}:${submission.problemId}`
    const list = groups.get(key) ?? []
    list.push(submission)
    groups.set(key, list)
  }
  return groups
}

function briefUser(user: { id: number; name: string; email: string }) {
  return {
    id: user.id,
    name: user.name,
    avatarUrl: gravatarUrl(user.email)
  }
}

function gravatarUrl(email: string) {
  const hash = createHash('md5').update(email.trim().toLowerCase()).digest('hex')
  return `https://www.gravatar.com/avatar/${hash}?d=identicon&s=80`
}

export async function validateContestSubmission(contestId: number, problemId: number) {
  const [contest] = await db
    .select()
    .from(schema.contests)
    .where(and(eq(schema.contests.id, contestId), isNull(schema.contests.deletedAt)))
    .limit(1)
  if (!contest)
    return { status: 404 as const, code: 'CONTEST_NOT_FOUND', message: 'Contest does not exist' }

  const now = new Date()
  if (now < contest.startAt) {
    return { status: 400 as const, code: 'CONTEST_NOT_STARTED', message: 'Contest has not started' }
  }
  if (now >= contest.endAt) {
    return { status: 400 as const, code: 'CONTEST_ENDED', message: 'Contest has ended' }
  }

  const [contestProblem] = await db
    .select({ problemId: schema.contestProblems.problemId })
    .from(schema.contestProblems)
    .innerJoin(schema.problems, eq(schema.contestProblems.problemId, schema.problems.id))
    .where(
      and(
        eq(schema.contestProblems.contestId, contestId),
        eq(schema.contestProblems.problemId, problemId),
        eq(schema.problems.visible, true),
        isNull(schema.problems.deletedAt)
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
