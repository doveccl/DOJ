import type { Server, ServerWebSocket, WebSocketHandler } from 'bun'
import { asc, eq } from 'drizzle-orm'
import { createHash } from 'node:crypto'
import { db, schema } from '@doj/db/client'
import type { JudgeProgress } from '@doj/shared/judge'
import { getAuthUser } from './auth'
import { getJudgeProgress, setProgressBroadcaster } from './progress'
import { getSessionUserId } from './session'

export interface BrowserSocketData {
  kind: 'browser'
  userId: number
  admin: boolean
  subscriptions: Set<number>
  feed: Set<number>
  lastPongAt: number
}

type BrowserMessage =
  | { type: 'subscribe-submission'; submissionId: number }
  | { type: 'unsubscribe-submission'; submissionId: number }
  | { type: 'subscribe-feed'; scope: 'self' | 'page'; submissionIds?: number[] }
  | { type: 'unsubscribe-feed'; scope: 'self' | 'page' }
  | { type: 'pong' }

const browserSockets = new Set<ServerWebSocket<BrowserSocketData>>()
let heartbeatStarted = false

setProgressBroadcaster((submissionId, progress) => {
  broadcastSubmissionProgress(submissionId, progress)
})

export async function handleBrowserUpgrade(request: Request, server: Server<BrowserSocketData>) {
  const protocol = request.headers
    .get('sec-websocket-protocol')
    ?.split(',')
    .map((item) => item.trim())
    .find((item) => item.startsWith('doj-auth.'))
  const token = protocol?.slice('doj-auth.'.length)
  const userId = token ? await getSessionUserId(token) : null
  const user = userId === null ? null : await getAuthUser(userId)
  if (!user) return new Response('Unauthorized', { status: 401 })

  const upgraded = server.upgrade(request, {
    data: {
      kind: 'browser',
      userId: user.id,
      admin: user.admin,
      subscriptions: new Set<number>(),
      feed: new Set<number>(),
      lastPongAt: Date.now()
    } satisfies BrowserSocketData,
    headers: protocol ? { 'Sec-WebSocket-Protocol': protocol } : undefined
  })
  return upgraded ? undefined : new Response('WebSocket upgrade failed', { status: 400 })
}

export const browserWebSocketHandlers: WebSocketHandler<BrowserSocketData> = {
  open(ws) {
    browserSockets.add(ws)
    startHeartbeat()
    ws.send(JSON.stringify({ type: 'ping' }))
  },
  async message(ws, raw) {
    const message = parseBrowserMessage(raw)
    if (!message) return

    if (message.type === 'pong') {
      ws.data.lastPongAt = Date.now()
      return
    }

    if (message.type === 'subscribe-submission') {
      if (!(await canViewSubmission(message.submissionId, ws.data.userId, ws.data.admin))) return
      ws.data.subscriptions.add(message.submissionId)
      const progress = await getJudgeProgress(message.submissionId)
      if (progress) {
        ws.send(JSON.stringify({ type: 'submission-progress', submissionId: message.submissionId, progress }))
      }
      return
    }

    if (message.type === 'unsubscribe-submission') {
      ws.data.subscriptions.delete(message.submissionId)
      return
    }

    if (message.type === 'subscribe-feed') {
      ws.data.feed.clear()
      if (message.scope === 'self') {
        const ids = await visibleSelfSubmissionIds(ws.data.userId)
        for (const id of ids) ws.data.feed.add(id)
        return
      }
      const requestedIds = (message.submissionIds ?? []).filter(Number.isInteger)
      for (const id of requestedIds) {
        if (await canViewSubmission(id, ws.data.userId, ws.data.admin)) ws.data.feed.add(id)
      }
      return
    }

    if (message.type === 'unsubscribe-feed') {
      ws.data.feed.clear()
    }
  },
  close(ws) {
    browserSockets.delete(ws)
  }
}

export function broadcastSubmissionProgress(submissionId: number, progress: JudgeProgress) {
  const message = JSON.stringify({ type: 'submission-progress', submissionId, progress })
  for (const ws of browserSockets) {
    if (ws.readyState === WebSocket.OPEN && ws.data.subscriptions.has(submissionId)) ws.send(message)
  }
  void broadcastSubmissionFeed(submissionId)
}

export async function broadcastSubmissionResult(submissionId: number) {
  const detailByUser = new Map<string, unknown>()
  const listItemByUser = new Map<string, unknown>()
  for (const ws of browserSockets) {
    if (ws.readyState !== WebSocket.OPEN) continue
    const key = `${ws.data.userId}:${ws.data.admin ? 1 : 0}`
    if (ws.data.subscriptions.has(submissionId)) {
      let detail = detailByUser.get(key)
      if (!detail) {
        detail = await submissionDetail(submissionId, ws.data.userId, ws.data.admin)
        detailByUser.set(key, detail)
      }
      if (detail) ws.send(JSON.stringify({ type: 'submission-result', submissionId, result: detail }))
    }
    if (ws.data.feed.has(submissionId)) {
      let listItem = listItemByUser.get(key)
      if (!listItem) {
        listItem = await submissionListItem(submissionId, ws.data.userId, ws.data.admin)
        listItemByUser.set(key, listItem)
      }
      if (listItem) ws.send(JSON.stringify({ type: 'submission-feed', item: listItem }))
    }
  }
}

async function broadcastSubmissionFeed(submissionId: number) {
  const listItemByUser = new Map<string, unknown>()
  for (const ws of browserSockets) {
    if (ws.readyState !== WebSocket.OPEN || !ws.data.feed.has(submissionId)) continue
    const key = `${ws.data.userId}:${ws.data.admin ? 1 : 0}`
    let listItem = listItemByUser.get(key)
    if (!listItem) {
      listItem = await submissionListItem(submissionId, ws.data.userId, ws.data.admin)
      listItemByUser.set(key, listItem)
    }
    if (listItem) ws.send(JSON.stringify({ type: 'submission-feed', item: listItem }))
  }
}

function startHeartbeat() {
  if (heartbeatStarted) return
  heartbeatStarted = true
  setInterval(() => {
    const now = Date.now()
    for (const ws of [...browserSockets]) {
      if (now - ws.data.lastPongAt > 60_000) {
        ws.close(1001, 'heartbeat timeout')
        browserSockets.delete(ws)
        continue
      }
      if (ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type: 'ping' }))
    }
  }, 30_000)
}

function parseBrowserMessage(raw: string | Buffer): BrowserMessage | null {
  try {
    const text = typeof raw === 'string' ? raw : Buffer.from(raw).toString('utf8')
    const message = JSON.parse(text) as { type?: unknown }
    if (
      message.type === 'subscribe-submission' ||
      message.type === 'unsubscribe-submission' ||
      message.type === 'subscribe-feed' ||
      message.type === 'unsubscribe-feed' ||
      message.type === 'pong'
    ) {
      return message as BrowserMessage
    }
  } catch {
    return null
  }
  return null
}

async function visibleSelfSubmissionIds(userId: number) {
  const rows = await db
    .select({ id: schema.submissions.id })
    .from(schema.submissions)
    .where(eq(schema.submissions.userId, userId))
    .orderBy(asc(schema.submissions.createdAt))
    .limit(200)
  return rows.map((row) => row.id)
}

async function canViewSubmission(submissionId: number, userId: number, admin: boolean) {
  const [row] = await db
    .select({
      userId: schema.submissions.userId,
      public: schema.submissions.public,
      assignmentId: schema.submissions.assignmentId,
      visible: schema.problems.visible,
      deletedAt: schema.problems.deletedAt
    })
    .from(schema.submissions)
    .innerJoin(schema.problems, eq(schema.submissions.problemId, schema.problems.id))
    .where(eq(schema.submissions.id, submissionId))
    .limit(1)
  if (!row) return false
  if ((!row.visible || row.deletedAt) && !admin && row.userId !== userId) return false
  if (row.assignmentId && !admin && row.userId !== userId) return false
  return admin || row.userId === userId || row.public
}

async function submissionListItem(submissionId: number, viewerUserId: number, admin: boolean) {
  const [row] = await db
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
    .where(eq(schema.submissions.id, submissionId))
    .limit(1)
  if (!row) return null
  const cropped = cropSubmissionResult(row, { viewerUserId, admin })
  const canViewProblem = admin || (row.problemVisible && !row.problemDeletedAt)
  return {
    id: row.id,
    problem: canViewProblem ? { id: row.problemId, title: row.problemTitle } : null,
    user: { id: row.userId, name: row.userName, avatarUrl: gravatarUrl(row.userEmail) },
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

async function submissionDetail(submissionId: number, userId: number, admin: boolean) {
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
      .leftJoin(schema.contests, eq(schema.submissions.contestId, schema.contests.id))
    .innerJoin(schema.users, eq(schema.submissions.userId, schema.users.id))
    .where(eq(schema.submissions.id, submissionId))
    .limit(1)
  if (!submission) return null
  const isOwnerOrAdmin = submission.userId === userId || admin
  const cropped = cropSubmissionResult(submission, { viewerUserId: userId, admin })
  const canInspect = !cropped
  const canInspectSource = (!submission.contestId && submission.public) || isOwnerOrAdmin
  const cases = canInspect
    ? await db
        .select()
        .from(schema.submissionCases)
        .where(eq(schema.submissionCases.submissionId, submission.id))
        .orderBy(asc(schema.submissionCases.caseNo))
    : []
  return {
    ...(await submissionListItem(submissionId, userId, admin)),
    code: canInspectSource ? submission.code : null,
    message: canInspect ? submission.message : null,
    cases: canInspect
      ? cases.map((item) => ({
          caseNo: item.caseNo,
          status: item.status,
          timeMs: item.timeMs,
          memoryBytes: item.memoryBytes,
          score: item.score,
          message: item.message || null
        }))
      : [],
    canCoach: canInspect && isOwnerOrAdmin && !['WAITING', 'JUDGING'].includes(submission.status),
    judgeProgress: canInspect ? await getJudgeProgress(submissionId) : null
  }
}

type CroppedSubmission = { displayStatus: string }

function cropSubmissionResult(
  row: {
    userId: number
    contestId: number | null
    contestType?: 'OI' | 'ICPC' | null
    contestStartAt?: Date | null
    contestEndAt?: Date | null
    contestFreezeAt?: Date | null
    status: string
  },
  viewer: { viewerUserId: number; admin: boolean }
): CroppedSubmission | null {
  if (viewer.admin || !row.contestId) return null
  const now = new Date()
  if (row.contestType === 'OI' && row.contestStartAt && row.contestEndAt && now >= row.contestStartAt && now < row.contestEndAt) {
    return { displayStatus: oiDisplayStatus(row.status) }
  }
  if (
    row.contestType === 'ICPC' &&
    row.userId !== viewer.viewerUserId &&
    row.contestFreezeAt &&
    row.contestEndAt &&
    now >= row.contestFreezeAt &&
    now < row.contestEndAt
  ) {
    return { displayStatus: row.status === 'WAITING' ? 'SUBMITTED' : row.status === 'JUDGING' ? 'JUDGING' : 'JUDGED' }
  }
  return null
}

function oiDisplayStatus(status: string) {
  if (status === 'WAITING') return 'SUBMITTED'
  if (status === 'JUDGING') return 'JUDGING'
  return 'JUDGED'
}

function gravatarUrl(email: string) {
  const hash = createHash('md5').update(email.trim().toLowerCase()).digest('hex')
  return `https://www.gravatar.com/avatar/${hash}?d=identicon&s=80`
}
