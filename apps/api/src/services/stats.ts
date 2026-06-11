import { createHash } from 'node:crypto'
import { and, asc, eq, inArray, isNull, sql } from 'drizzle-orm'
import { db, schema } from '@doj/db/client'
import type { JudgeStatus } from '@doj/shared/status'
import { redisCommand } from '../redis'

export async function countRows(table: any, where?: any) {
  const query = db.select({ total: sql<number>`count(*)::int` }).from(table)
  const [row] = await (where ? query.where(where) : query)
  return row?.total ?? 0
}

export async function countVisibleSubmissions() {
  const [row] = await db
    .select({ total: sql<number>`count(*)::int` })
    .from(schema.submissions)
    .innerJoin(schema.problems, eq(schema.submissions.problemId, schema.problems.id))
    .where(eq(schema.problems.visible, true))
  return row?.total ?? 0
}

type SubmissionLike = {
  id: number
  userId: number
  problemId: number
  status: JudgeStatus
  createdAt: Date
}

type ProblemStats = {
  solved: number
  attempted: number
  submission: number
}

type RankingSourceRow = {
  userId: number
  userName: string
  userEmail: string
  solved: number
  submissions: number
  acAt: Date | string | null
}

const terminalStatuses = new Set<JudgeStatus>(['AC', 'WA', 'PE', 'TLE', 'MLE', 'OLE', 'RE', 'CE', 'SE'])
const repairIntervalMs = Number(process.env.STATS_REPAIR_INTERVAL_MS ?? 10 * 60 * 1000)

let repairRunning = false
let repairCronStarted = false
const memoryStats = createEmptyMemoryStats()

export async function recordSubmissionCreated(submission: SubmissionLike) {
  const solved = await getUserSolvedCount(submission.userId)
  memoryStats.userSubmissions.set(submission.userId, (memoryStats.userSubmissions.get(submission.userId) ?? 0) + 1)
  memoryStats.problemSubmissions.set(
    submission.problemId,
    (memoryStats.problemSubmissions.get(submission.problemId) ?? 0) + 1
  )

  await Promise.all([
    redisCommand('HINCRBY', [`stats:user:${submission.userId}`, 'submission', '1']),
    redisCommand('HINCRBY', [`stats:problem:${submission.problemId}`, 'submission', '1']),
    redisCommand('ZADD', ['rank:global', String(-solved), String(submission.userId)])
  ])
}

export async function recordSubmissionFinal(submission: SubmissionLike) {
  if (!terminalStatuses.has(submission.status)) return

  addToSetMap(memoryStats.problemAttempted, submission.problemId, submission.userId)
  await updateAttemptedRedis(submission.problemId, submission.userId)

  if (submission.status !== 'AC') return

  addToSetMap(memoryStats.userSolved, submission.userId, submission.problemId)
  const memoryProblemSolved = addToSetMap(memoryStats.problemSolved, submission.problemId, submission.userId)
  const currentAcAt = memoryStats.userAcAt.get(submission.userId)
  if (!currentAcAt || submission.createdAt > currentAcAt) memoryStats.userAcAt.set(submission.userId, submission.createdAt)

  const solved = await updateSolvedRedis(submission.userId, submission.problemId, submission.createdAt)
  await Promise.all([
    redisCommand('ZADD', ['rank:global', String(-solved), String(submission.userId)]),
    (await hasProblemSolvedRedis(submission.problemId, submission.userId, memoryProblemSolved))
      ? redisCommand('HINCRBY', [`stats:problem:${submission.problemId}`, 'solved', '1'])
      : Promise.resolve(null)
  ])
}

export async function getProblemStats(problemIds: number[]) {
  const uniqueIds = [...new Set(problemIds)]
  const result = new Map<number, ProblemStats>()
  const missing: number[] = []

  await Promise.all(
    uniqueIds.map(async (problemId) => {
      const hash = await redisHash(`stats:problem:${problemId}`)
      if (hash) {
        result.set(problemId, {
          solved: Number(hash.solved ?? 0),
          attempted: Number(hash.attempted ?? 0),
          submission: Number(hash.submission ?? 0)
        })
      } else if (memoryStats.problemSubmissions.has(problemId)) {
        result.set(problemId, {
          solved: memoryStats.problemSolved.get(problemId)?.size ?? 0,
          attempted: memoryStats.problemAttempted.get(problemId)?.size ?? 0,
          submission: memoryStats.problemSubmissions.get(problemId) ?? 0
        })
      } else {
        missing.push(problemId)
      }
    })
  )

  if (missing.length) {
    for (const row of await queryProblemStats(missing)) {
      result.set(row.problemId, {
        solved: row.solved,
        attempted: row.attempted,
        submission: row.submission
      })
    }
  }

  return result
}

export async function hasUserSolvedProblem(userId: number | undefined, problemId: number) {
  if (!userId) return false

  const redisValue = await redisCommand('SISMEMBER', [`solved:user:${userId}`, String(problemId)])
  if (redisValue !== null) return Number(redisValue) === 1
  if (memoryStats.userSolved.has(userId)) return memoryStats.userSolved.get(userId)!.has(problemId)

  const [row] = await db
    .select({ id: schema.submissions.id })
    .from(schema.submissions)
    .where(and(eq(schema.submissions.userId, userId), eq(schema.submissions.problemId, problemId), eq(schema.submissions.status, 'AC')))
    .limit(1)
  return Boolean(row)
}

export async function getRanking(page: number, pageSize: number) {
  const offset = (page - 1) * pageSize
  const redisIds = await redisCommand('ZRANGE', ['rank:global', '0', '-1'])
  const redisRows = Array.isArray(redisIds) ? await getRedisRankingRows(redisIds.map(String)) : []
  const rows = redisRows.length ? redisRows : await queryRankingRows()
  const sorted = rows.sort(compareRankingRows)
  const items = sorted.slice(offset, offset + pageSize).map((row, index) => ({
    rank: offset + index + 1,
    user: {
      id: row.userId,
      name: row.userName,
      avatarUrl: gravatarUrl(row.userEmail)
    },
    solved: row.solved,
    submissions: row.submissions,
    acAt: row.acAt ? new Date(row.acAt).toISOString() : null
  }))
  return { items, page, pageSize, total: sorted.length }
}

export async function getHeatmap(userId: number, timeZone: string) {
  const tz = validTimeZone(timeZone) ? timeZone : 'UTC'
  const days = rollingDays(tz)
  const start = new Date(`${days[0]}T00:00:00.000Z`).toISOString()
  const rows = await db
    .select({ createdAt: schema.submissions.createdAt })
    .from(schema.submissions)
    .where(and(eq(schema.submissions.userId, userId), sql`${schema.submissions.createdAt} >= ${start}`))
  const countByDate = new Map(days.map((date) => [date, 0]))
  for (const row of rows) {
    const key = formatDateInTimeZone(row.createdAt, tz)
    if (countByDate.has(key)) countByDate.set(key, countByDate.get(key)! + 1)
  }
  return days.map((date) => ({ date, count: countByDate.get(date) ?? 0 }))
}

export async function getRecommendedProblems(userId: number | undefined) {
  const solvedIds = userId ? await getUserSolvedIds(userId) : new Set<number>()
  const rows = await db
    .select({
      id: schema.problems.id,
      title: schema.problems.title,
      tags: schema.problems.tags,
      visible: schema.problems.visible
    })
    .from(schema.problems)
    .where(and(eq(schema.problems.visible, true), isNull(schema.problems.deletedAt)))
    .orderBy(asc(schema.problems.id))
    .limit(userId ? 100 : 10)
  const stats = await getProblemStats(rows.map((row) => row.id))
  return rows
    .filter((row) => !solvedIds.has(row.id))
    .slice(0, 10)
    .map((row) => ({
      id: row.id,
      title: row.title,
      tags: row.tags,
      passRate: passRate(stats.get(row.id)?.solved ?? 0, stats.get(row.id)?.attempted ?? 0),
      solved: false,
      visible: row.visible
    }))
}

export async function repairDerivedStats() {
  if (repairRunning) return getLastRepairStatus()
  repairRunning = true
  const startedAt = Date.now()
  await setRepairStatus({ time: new Date().toISOString(), status: 'RUNNING', duration: '0', message: 'Repair is running' })

  try {
    const rebuilt = createEmptyMemoryStats()
    const rows = await db
      .select({
        id: schema.submissions.id,
        userId: schema.submissions.userId,
        problemId: schema.submissions.problemId,
        status: schema.submissions.status,
        createdAt: schema.submissions.createdAt
      })
      .from(schema.submissions)
      .orderBy(asc(schema.submissions.id))

    for (const row of rows) {
      rebuilt.userSubmissions.set(row.userId, (rebuilt.userSubmissions.get(row.userId) ?? 0) + 1)
      rebuilt.problemSubmissions.set(row.problemId, (rebuilt.problemSubmissions.get(row.problemId) ?? 0) + 1)
      if (!terminalStatuses.has(row.status)) continue
      addToSetMap(rebuilt.problemAttempted, row.problemId, row.userId)
      if (row.status === 'AC') {
        addToSetMap(rebuilt.userSolved, row.userId, row.problemId)
        addToSetMap(rebuilt.problemSolved, row.problemId, row.userId)
        const currentAcAt = rebuilt.userAcAt.get(row.userId)
        if (!currentAcAt || row.createdAt > currentAcAt) rebuilt.userAcAt.set(row.userId, row.createdAt)
      }
    }

    resetMemoryStats(rebuilt)
    await writeRedisSnapshot(rebuilt)
    const status = {
      time: new Date().toISOString(),
      status: 'DONE',
      duration: String(Date.now() - startedAt),
      message: `Rebuilt stats from ${rows.length} submissions`
    }
    await setRepairStatus(status)
    return status
  } catch (error) {
    const status = {
      time: new Date().toISOString(),
      status: 'FAILED',
      duration: String(Date.now() - startedAt),
      message: error instanceof Error ? error.message : String(error)
    }
    await setRepairStatus(status)
    return status
  } finally {
    repairRunning = false
  }
}

export async function getLastRepairStatus() {
  const fromRedis = await redisHash('repair:stats:last')
  if (fromRedis) return fromRedis
  return memoryStats.repairStatus
}

export function startStatsRepairCron() {
  if (repairCronStarted) return
  repairCronStarted = true
  setInterval(() => {
    void repairDerivedStats().catch((error) => console.error('stats repair cron error:', error))
  }, repairIntervalMs)
}

async function updateAttemptedRedis(problemId: number, userId: number) {
  const added = await redisCommand('SADD', [`attempted:problem:${problemId}`, String(userId)])
  if (Number(added) === 1) await redisCommand('HINCRBY', [`stats:problem:${problemId}`, 'attempted', '1'])
}

async function updateSolvedRedis(userId: number, problemId: number, acAt: Date) {
  const userAdded = await redisCommand('SADD', [`solved:user:${userId}`, String(problemId)])
  let solved = await getUserSolvedCount(userId)
  if (Number(userAdded) === 1) {
    const updated = await redisCommand('HINCRBY', [`stats:user:${userId}`, 'solved', '1'])
    solved = Number(updated)
  }
  await redisCommand('HSET', [`stats:user:${userId}`, 'acAt', acAt.toISOString()])
  return solved
}

async function hasProblemSolvedRedis(problemId: number, userId: number, fallback: boolean) {
  const added = await redisCommand('SADD', [`solved:problem:${problemId}`, String(userId)])
  if (added === null) return fallback
  return Number(added) === 1
}

async function getUserSolvedCount(userId: number) {
  const fromRedis = await redisCommand('HGET', [`stats:user:${userId}`, 'solved'])
  if (fromRedis !== null) return Number(fromRedis) || 0
  return memoryStats.userSolved.get(userId)?.size ?? 0
}

async function writeRedisSnapshot(stats: ReturnType<typeof createEmptyMemoryStats>) {
  const keys = await Promise.all(
    ['solved:user:*', 'solved:problem:*', 'attempted:problem:*', 'stats:user:*', 'stats:problem:*', 'rank:global'].map(
      async (pattern) => {
        const value = await redisCommand('KEYS', [pattern])
        return Array.isArray(value) ? value.map(String) : []
      }
    )
  )
  const flatKeys = keys.flat()
  if (flatKeys.length) await redisCommand('DEL', flatKeys)

  for (const [userId, problemIds] of stats.userSolved) {
    if (problemIds.size) await redisCommand('SADD', [`solved:user:${userId}`, ...[...problemIds].map(String)])
  }
  for (const [problemId, userIds] of stats.problemSolved) {
    if (userIds.size) await redisCommand('SADD', [`solved:problem:${problemId}`, ...[...userIds].map(String)])
  }
  for (const [problemId, userIds] of stats.problemAttempted) {
    if (userIds.size) await redisCommand('SADD', [`attempted:problem:${problemId}`, ...[...userIds].map(String)])
  }
  for (const [problemId, submission] of stats.problemSubmissions) {
    await redisCommand('HSET', [
      `stats:problem:${problemId}`,
      'solved',
      String(stats.problemSolved.get(problemId)?.size ?? 0),
      'attempted',
      String(stats.problemAttempted.get(problemId)?.size ?? 0),
      'submission',
      String(submission)
    ])
  }
  for (const [userId, submissions] of stats.userSubmissions) {
    const solved = stats.userSolved.get(userId)?.size ?? 0
    await redisCommand('HSET', [
      `stats:user:${userId}`,
      'solved',
      String(solved),
      'submission',
      String(submissions),
      'acAt',
      stats.userAcAt.get(userId)?.toISOString() ?? ''
    ])
    await redisCommand('ZADD', ['rank:global', String(-solved), String(userId)])
  }
}

async function queryProblemStats(problemIds: number[]) {
  if (!problemIds.length) return []
  return db
    .select({
      problemId: schema.submissions.problemId,
      solved: sql<number>`count(distinct ${schema.submissions.userId}) filter (where ${schema.submissions.status} = 'AC')::int`,
      attempted: sql<number>`count(distinct ${schema.submissions.userId}) filter (where ${schema.submissions.status} not in ('WAITING', 'JUDGING'))::int`,
      submission: sql<number>`count(*)::int`
    })
    .from(schema.submissions)
    .where(inArray(schema.submissions.problemId, problemIds))
    .groupBy(schema.submissions.problemId)
}

async function queryRankingRows(): Promise<RankingSourceRow[]> {
  return db
    .select({
      userId: schema.users.id,
      userName: schema.users.name,
      userEmail: schema.users.email,
      solved: sql<number>`count(distinct ${schema.submissions.problemId}) filter (where ${schema.submissions.status} = 'AC')::int`,
      submissions: sql<number>`count(${schema.submissions.id})::int`,
      acAt: sql<Date | null>`max(${schema.submissions.createdAt}) filter (where ${schema.submissions.status} = 'AC')`
    })
    .from(schema.users)
    .innerJoin(schema.submissions, eq(schema.submissions.userId, schema.users.id))
    .where(isNull(schema.users.disabledAt))
    .groupBy(schema.users.id)
}

async function getRedisRankingRows(userIds: string[]): Promise<RankingSourceRow[]> {
  if (!userIds.length) return []
  const ids = userIds.map(Number).filter(Number.isFinite)
  const users = await db
    .select({
      userId: schema.users.id,
      userName: schema.users.name,
      userEmail: schema.users.email
    })
    .from(schema.users)
    .where(and(inArray(schema.users.id, ids), isNull(schema.users.disabledAt)))
  const rows: RankingSourceRow[] = []
  for (const user of users) {
    const hash = await redisHash(`stats:user:${user.userId}`)
    rows.push({
      ...user,
      solved: Number(hash?.solved ?? 0),
      submissions: Number(hash?.submission ?? 0),
      acAt: hash?.acAt || null
    })
  }
  return rows.filter((row) => row.submissions > 0)
}

async function getUserSolvedIds(userId: number) {
  const redisValue = await redisCommand('SMEMBERS', [`solved:user:${userId}`])
  if (Array.isArray(redisValue)) return new Set(redisValue.map(Number).filter(Number.isFinite))
  if (memoryStats.userSolved.has(userId)) return new Set(memoryStats.userSolved.get(userId)!)
  const rows = await db
    .select({ problemId: schema.submissions.problemId })
    .from(schema.submissions)
    .where(and(eq(schema.submissions.userId, userId), eq(schema.submissions.status, 'AC')))
    .groupBy(schema.submissions.problemId)
  return new Set(rows.map((row) => row.problemId))
}

async function redisHash(key: string) {
  const value = await redisCommand('HGETALL', [key])
  if (!Array.isArray(value) || value.length === 0) return null
  const hash: Record<string, string> = {}
  for (let index = 0; index < value.length; index += 2) {
    hash[String(value[index])] = String(value[index + 1] ?? '')
  }
  return hash
}

async function setRepairStatus(status: Record<string, string>) {
  memoryStats.repairStatus = status
  await redisCommand('HSET', ['repair:stats:last', ...Object.entries(status).flat()])
}

function compareRankingRows(left: RankingSourceRow, right: RankingSourceRow) {
  const leftAcAt = left.acAt ? new Date(left.acAt).getTime() : Number.POSITIVE_INFINITY
  const rightAcAt = right.acAt ? new Date(right.acAt).getTime() : Number.POSITIVE_INFINITY
  return right.solved - left.solved || left.submissions - right.submissions || leftAcAt - rightAcAt || left.userId - right.userId
}

function createEmptyMemoryStats() {
  return {
    userSolved: new Map<number, Set<number>>(),
    problemSolved: new Map<number, Set<number>>(),
    problemAttempted: new Map<number, Set<number>>(),
    userSubmissions: new Map<number, number>(),
    problemSubmissions: new Map<number, number>(),
    userAcAt: new Map<number, Date>(),
    repairStatus: {
      time: '',
      status: 'NEVER',
      duration: '0',
      message: 'Stats repair has not run yet'
    } as Record<string, string>
  }
}

function resetMemoryStats(next: ReturnType<typeof createEmptyMemoryStats>) {
  memoryStats.userSolved = next.userSolved
  memoryStats.problemSolved = next.problemSolved
  memoryStats.problemAttempted = next.problemAttempted
  memoryStats.userSubmissions = next.userSubmissions
  memoryStats.problemSubmissions = next.problemSubmissions
  memoryStats.userAcAt = next.userAcAt
}

function addToSetMap(map: Map<number, Set<number>>, key: number, value: number) {
  const set = map.get(key) ?? new Set<number>()
  const before = set.size
  set.add(value)
  map.set(key, set)
  return set.size !== before
}

function passRate(solved: number, attempted: number) {
  return attempted > 0 ? solved / attempted : 0
}

function validTimeZone(timeZone: string) {
  try {
    new Intl.DateTimeFormat('en-US', { timeZone }).format(new Date())
    return true
  } catch {
    return false
  }
}

function rollingDays(timeZone: string) {
  const today = formatDateInTimeZone(new Date(), timeZone)
  const [year, month, day] = today.split('-').map(Number)
  const todayUtc = Date.UTC(year, month - 1, day)
  return Array.from({ length: 365 }, (_, index) => formatPlainDate(new Date(todayUtc - (364 - index) * 86400000)))
}

function formatDateInTimeZone(date: Date, timeZone: string) {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit'
  }).formatToParts(date)
  const value = Object.fromEntries(parts.map((part) => [part.type, part.value]))
  return `${value.year}-${value.month}-${value.day}`
}

function formatPlainDate(date: Date) {
  return date.toISOString().slice(0, 10)
}

function gravatarUrl(email: string) {
  const hash = createHash('md5').update(email.trim().toLowerCase()).digest('hex')
  return `https://www.gravatar.com/avatar/${hash}?d=identicon&s=80`
}
