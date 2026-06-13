import { Hono, type Context } from 'hono'
import { and, asc, desc, eq, ilike, isNull, or, sql } from 'drizzle-orm'
import { createHash } from 'node:crypto'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { deleteObject, getObjectBytes, listObjects, putObject } from '@doj/shared/storage'
import { authMiddleware, denyGuestAccess, getOptionalAuthUser, requireAdmin } from '../auth'
import { ApiHttpError, apiError, notFound } from '../errors'
import { countTopics } from '../services/discussion'
import { countRows, getProblemStats, getUserSolvedIds, getUserSubmittedProblemIds, hasUserSolvedProblem } from '../services/stats'
import { listQuerySchema, numericId, pageOffset } from '../validation'

const maxStatementBytes = 64 * 1024
const maxAssetFileBytes = 64 * 1024 * 1024
const maxAssetRequestBytes = maxAssetFileBytes + 1024 * 1024
const textExtensions = new Set([
  '.txt', '.md', '.json', '.yaml', '.yml', '.xml', '.csv', '.in', '.out', '.ans',
  '.cpp', '.c', '.h', '.hpp', '.py', '.js', '.ts', '.java', '.rs', '.go', 'dockerfile'
])

export function registerProblemRoutes(app: Hono) {
  app.get('/api/problems', async (c) => {
    const denied = await denyGuestProblemset(c)
    if (denied) return denied

    const authUser = await getOptionalAuthUser(c)
    const { page, pageSize, q, tag } = problemListQuerySchema.parse(c.req.query())
    const visibilityFilter = authUser?.admin
      ? isNull(schema.problems.deletedAt)
      : and(eq(schema.problems.visible, true), isNull(schema.problems.deletedAt))
    const where = and(
      visibilityFilter,
      q ? problemSearchFilter(q) : undefined,
      tag ? sql`${tag} = any(${schema.problems.tags})` : undefined
    )

    const total = await countRows(schema.problems, where)
    const rows = await db
      .select({
        id: schema.problems.id,
        title: schema.problems.title,
        tags: schema.problems.tags,
        visible: schema.problems.visible,
        deletedAt: schema.problems.deletedAt
      })
      .from(schema.problems)
      .where(where)
      .orderBy(asc(schema.problems.id))
      .limit(pageSize)
      .offset(pageOffset(page, pageSize))

    const problemIds = rows.map((row) => row.id)
    const stats = await getProblemStats(problemIds)
    const solvedIds = authUser ? await getUserSolvedIds(authUser.id) : new Set<number>()
    const submittedIds = await getUserSubmittedProblemIds(authUser?.id, problemIds)
    const userProgress = new Map(
      rows.map((row) => [
        row.id,
        {
          solved: solvedIds.has(row.id),
          submitted: submittedIds.has(row.id)
        }
      ] as const)
    )
    return c.json({
      total,
      page,
      pageSize,
      items: rows.map((row) => formatProblemListItem(row, stats.get(row.id), userProgress.get(row.id)))
    })
  })

  app.get('/api/problems/:id', async (c) => {
    const denied = await denyGuestProblemset(c)
    if (denied) return denied

    const detail = await getProblemDetail(numericId.parse(c.req.param('id')), await getOptionalAuthUser(c))
    if (!detail) return notFound(c)
    return c.json(detail)
  })

  app.post('/api/admin/problems', authMiddleware, async (c) => {
    const denied = await requireAdmin(c)
    if (denied) return denied

    const body = createProblemSchema.parse(await c.req.json())
    const [problem] = await db
      .insert(schema.problems)
      .values({
        title: body.title,
        tags: body.tags,
        mode: body.mode,
        timeLimit: body.timeLimit,
        memoryLimit: body.memoryLimit,
        visible: body.visible
      })
      .returning()

    await putObject({
      key: problemObjectKey(problem.id, 'statement.md'),
      body: `# ${body.title}\n`,
      contentType: 'text/markdown; charset=utf-8'
    })
    return c.json(await getProblemDetail(problem.id, await getOptionalAuthUser(c)), 201)
  })

  app.get('/api/admin/problems/:id', authMiddleware, async (c) => {
    const denied = await requireAdmin(c)
    if (denied) return denied

    const detail = await getProblemDetail(numericId.parse(c.req.param('id')), await getOptionalAuthUser(c))
    if (!detail) return notFound(c)
    return c.json(detail)
  })

  app.patch('/api/admin/problems/:id', authMiddleware, async (c) => {
    const denied = await requireAdmin(c)
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const body = updateProblemSchema.parse(await c.req.json())
    const patch: Partial<typeof schema.problems.$inferInsert> = { updatedAt: new Date() }
    if (body.title !== undefined) patch.title = body.title
    if (body.tags !== undefined) patch.tags = body.tags
    if (body.visible !== undefined) patch.visible = body.visible
    if (body.mode !== undefined) patch.mode = body.mode
    if (body.timeLimit !== undefined) patch.timeLimit = body.timeLimit
    if (body.memoryLimit !== undefined) patch.memoryLimit = body.memoryLimit

    const [updated] = await db.update(schema.problems).set(patch).where(eq(schema.problems.id, id)).returning()
    if (!updated) return notFound(c)
    return c.json(await getProblemDetail(id, await getOptionalAuthUser(c)))
  })

  app.delete('/api/admin/problems/:id', authMiddleware, async (c) => {
    const denied = await requireAdmin(c)
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const [updated] = await db
      .update(schema.problems)
      .set({ deletedAt: new Date(), updatedAt: new Date() })
      .where(eq(schema.problems.id, id))
      .returning()
    if (!updated) return notFound(c)
    return c.json({ ok: true })
  })

  app.put('/api/admin/problems/:id/statement', authMiddleware, async (c) => {
    const denied = await requireAdmin(c)
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    if (!(await problemExists(id))) return notFound(c)
    const body = statementSchema.parse(await c.req.json())
    if (new TextEncoder().encode(body.markdown).byteLength > maxStatementBytes) {
      return apiError(c, 413, 'STATEMENT_TOO_LARGE', 'Statement markdown is too large')
    }
    await putObject({ key: problemObjectKey(id, 'statement.md'), body: body.markdown, contentType: 'text/markdown; charset=utf-8' })
    await touchProblem(id)
    return c.json({ markdown: body.markdown })
  })

  app.get('/api/admin/problems/:id/assets', authMiddleware, async (c) => {
    const denied = await requireAdmin(c)
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    if (!(await problemExists(id))) return notFound(c)
    const objects = await listObjects(problemObjectKey(id, ''))
    const assets = objects
      .map(assetFromObject)
      .filter((asset): asset is ProblemAssetItem => Boolean(asset))
      .sort(compareAssets)
    return c.json(assets)
  })

  app.get('/api/admin/problems/:id/assets/content', authMiddleware, async (c) => {
    const denied = await requireAdmin(c)
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const path = assetPathSchema.parse(c.req.query('path'))
    const asset = await validateAssetPathForProblem(id, path)
    const bytes = await getObjectBytes(problemObjectKey(id, asset.path))
    const text = isTextAsset(asset.path, asset.contentType)
    return c.json({
      path: asset.path,
      contentType: asset.contentType,
      text,
      content: text ? Buffer.from(bytes).toString('utf8') : Buffer.from(bytes).toString('base64'),
      encoding: text ? 'utf8' : 'base64'
    })
  })

  app.put('/api/admin/problems/:id/assets/content', authMiddleware, async (c) => {
    const denied = await requireAdmin(c)
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const body = assetContentSchema.parse(await c.req.json())
    const asset = await validateAssetPathForProblem(id, body.path)
    const bytes = body.encoding === 'utf8' ? new TextEncoder().encode(body.content) : Buffer.from(body.content, 'base64')
    if (bytes.byteLength > maxAssetFileBytes) return apiError(c, 413, 'FILE_TOO_LARGE', 'Asset file is too large')
    const contentType = body.contentType || inferContentType(asset.path)
    await putObject({ key: problemObjectKey(id, asset.path), body: bytes, contentType })
    await touchProblem(id)
    return c.json({ ...asset, size: bytes.byteLength, contentType, updatedAt: new Date().toISOString(), text: isTextAsset(asset.path, contentType) })
  })

  app.post('/api/admin/problems/:id/assets/upload', authMiddleware, async (c) => {
    const denied = await requireAdmin(c)
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const contentLength = Number(c.req.header('content-length') ?? 0)
    if (Number.isFinite(contentLength) && contentLength > maxAssetRequestBytes) {
      return apiError(c, 413, 'FILE_TOO_LARGE', 'Asset upload is too large')
    }
    const form = await c.req.formData()
    const upload = form.get('file')
    if (!(upload instanceof File)) return apiError(c, 400, 'MISSING_FILE', 'Expected multipart file field named file')
    if (upload.size > maxAssetFileBytes) return apiError(c, 413, 'FILE_TOO_LARGE', 'Asset file is too large')
    const rawPath = String(form.get('path') ?? `assets/${sanitizeFilename(upload.name)}`)
    const path = rawPath.endsWith('/') ? `${rawPath}${sanitizeFilename(upload.name)}` : rawPath
    const asset = await validateAssetPathForProblem(id, path)
    const contentType = upload.type || inferContentType(asset.path)
    await putObject({ key: problemObjectKey(id, asset.path), body: new Uint8Array(await upload.arrayBuffer()), contentType })
    await touchProblem(id)
    return c.json({ path: asset.path, url: asset.path.startsWith('assets/') ? `/api/problems/${id}/assets/${asset.name}` : null }, 201)
  })

  app.delete('/api/admin/problems/:id/assets', authMiddleware, async (c) => {
    const denied = await requireAdmin(c)
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const path = assetPathSchema.parse(c.req.query('path'))
    const asset = await validateAssetPathForProblem(id, path)
    await deleteObject(problemObjectKey(id, asset.path))
    await touchProblem(id)
    return c.json({ ok: true })
  })

  app.get('/api/problems/:id/assets/:filename', async (c) => {
    const denied = await denyGuestProblemset(c)
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const filename = c.req.param('filename')
    if (!isFlatFilename(filename)) return notFound(c)
    const [problem] = await db
      .select({ id: schema.problems.id })
      .from(schema.problems)
      .where(and(eq(schema.problems.id, id), eq(schema.problems.visible, true), isNull(schema.problems.deletedAt)))
      .limit(1)
    if (!problem) return notFound(c)
    const path = `assets/${filename}`
    const bytes = await readObjectOrNull(problemObjectKey(id, path))
    if (!bytes) return notFound(c)
    c.header('cache-control', 'public, max-age=31536000, immutable')
    c.header('content-type', inferContentType(path))
    return c.body(bytes)
  })
}

async function denyGuestProblemset(c: Context) {
  return denyGuestAccess(c, 'Sign in to view the problemset')
}

type ProblemAssetSection = 'data' | 'assets' | 'root'

interface ProblemAssetItem {
  path: string
  name: string
  section: ProblemAssetSection
  size: number
  contentType: string
  updatedAt: string
  text: boolean
}

async function getProblemDetail(id: number, authUser: Awaited<ReturnType<typeof getOptionalAuthUser>>) {
  const [problem] = await db.select().from(schema.problems).where(eq(schema.problems.id, id)).limit(1)
  if (!problem) return null
  if (problem.deletedAt) return null
  if (!authUser?.admin && !problem.visible) return null

  const stats = (await getProblemStats([id])).get(id)
  const solved = await hasUserSolvedProblem(authUser?.id, id)
  const [recentSubmission, discussionCount] = await Promise.all([
    getRecentSubmission(id, authUser),
    countTopics([`P${id}`])
  ])

  return {
    id: problem.id,
    title: problem.title,
    statement: await readStatement(id),
    mode: problem.mode,
    timeLimit: problem.timeLimit,
    memoryLimit: problem.memoryLimit,
    tags: problem.tags,
    solvedCount: stats?.solved ?? 0,
    attemptedCount: stats?.attempted ?? 0,
    submissionCount: stats?.submission ?? 0,
    passRate: passRate(stats?.solved ?? 0, stats?.attempted ?? 0),
    solved,
    recentSubmission,
    discussionCount,
    visible: problem.visible,
    deletedAt: null,
    createdAt: problem.createdAt.toISOString(),
    updatedAt: problem.updatedAt.toISOString()
  }
}

async function getRecentSubmission(
  problemId: number,
  authUser: Awaited<ReturnType<typeof getOptionalAuthUser>>
) {
  if (!authUser) return null

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
      contestType: schema.contests.type,
      contestStartAt: schema.contests.startAt,
      contestEndAt: schema.contests.endAt
    })
    .from(schema.submissions)
    .innerJoin(schema.problems, eq(schema.submissions.problemId, schema.problems.id))
    .innerJoin(schema.users, eq(schema.submissions.userId, schema.users.id))
    .leftJoin(schema.contests, eq(schema.submissions.contestId, schema.contests.id))
    .where(and(eq(schema.submissions.problemId, problemId), eq(schema.submissions.userId, authUser.id)))
    .orderBy(desc(schema.submissions.createdAt), desc(schema.submissions.id))
    .limit(1)

  if (!row) return null
  const cropped = cropRecentSubmission(row, authUser.admin)
  return {
    id: row.id,
    problem: { id: row.problemId, title: row.problemTitle },
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

function cropRecentSubmission(row: {
  contestId: number | null
  contestType: 'OI' | 'ICPC' | null
  contestStartAt: Date | null
  contestEndAt: Date | null
  status: string
}, isAdmin: boolean) {
  if (isAdmin || !row.contestId) return null
  const now = new Date()
  if (row.contestType === 'OI' && row.contestStartAt && row.contestEndAt && now >= row.contestStartAt && now < row.contestEndAt) {
    if (row.status === 'WAITING') return { displayStatus: 'SUBMITTED' }
    if (row.status === 'JUDGING') return { displayStatus: 'JUDGING' }
    return { displayStatus: 'JUDGED' }
  }
  return null
}

function formatProblemListItem(row: {
  id: number
  title: string
  tags: string[]
  visible: boolean
  deletedAt: Date | null
}, stats: { solved: number; attempted: number; submission: number } | undefined, progress?: { solved: boolean; submitted: boolean }) {
  return {
    id: row.id,
    title: row.title,
    tags: row.tags,
    solvedCount: stats?.solved ?? 0,
    attemptedCount: stats?.attempted ?? 0,
    submissionCount: stats?.submission ?? 0,
    passRate: passRate(stats?.solved ?? 0, stats?.attempted ?? 0),
    solved: progress?.solved ?? false,
    submitted: progress?.submitted ?? false,
    visible: row.visible,
    deletedAt: row.deletedAt?.toISOString() ?? null
  }
}

function passRate(solvedUsers: number, attemptedUsers: number) {
  return attemptedUsers > 0 ? solvedUsers / attemptedUsers : 0
}

function gravatarUrl(email: string) {
  const hash = createHash('md5').update(email.trim().toLowerCase()).digest('hex')
  return `https://www.gravatar.com/avatar/${hash}?d=identicon&s=80`
}

async function readStatement(problemId: number) {
  const bytes = await readObjectOrNull(problemObjectKey(problemId, 'statement.md'))
  return bytes ? Buffer.from(bytes).toString('utf8') : ''
}

async function readObjectOrNull(key: string) {
  try {
    return await getObjectBytes(key)
  } catch {
    return null
  }
}

async function problemExists(id: number) {
  const [problem] = await db.select({ id: schema.problems.id }).from(schema.problems).where(eq(schema.problems.id, id)).limit(1)
  return Boolean(problem)
}

async function touchProblem(id: number) {
  await db.update(schema.problems).set({ updatedAt: new Date() }).where(eq(schema.problems.id, id))
}

async function validateAssetPathForProblem(problemId: number, path: string) {
  const normalized = normalizeAssetPath(path)
  const [problem] = await db.select({ mode: schema.problems.mode }).from(schema.problems).where(eq(schema.problems.id, problemId)).limit(1)
  if (!problem) throwAssetPathError('Problem does not exist')
  if (assetSection(normalized) === 'root' && problem.mode !== 'custom') {
    throwAssetPathError('Root judge resources are only allowed for custom problems')
  }
  return assetMetadata(normalized, 0, new Date())
}

function normalizeAssetPath(path: string) {
  const value = path.trim()
  if (
    !value ||
    value.startsWith('/') ||
    value.includes('\\') ||
    hasControlCharacter(value) ||
    value.split('/').some((part) => !part || part === '.' || part === '..') ||
    value === 'data' ||
    value === 'assets' ||
    value === 'statement.md' ||
    value.startsWith('statement.md/')
  ) {
    throwAssetPathError()
  }
  const section = assetSection(value)
  if (section === 'assets' && !isFlatFilename(value.slice('assets/'.length))) throwAssetPathError('assets/ does not allow subdirectories')
  return value
}

function assetSection(path: string): ProblemAssetSection {
  if (path.startsWith('data/')) return 'data'
  if (path.startsWith('assets/')) return 'assets'
  return 'root'
}

function hasControlCharacter(value: string) {
  return Array.from(value).some((char) => char.charCodeAt(0) <= 0x1f)
}

function assetFromObject(object: { key: string; size: number; updatedAt: Date }) {
  const match = object.key.match(/^problems\/\d+\/(.+)$/)
  const path = match?.[1]
  if (!path || path === 'statement.md') return null
  return assetMetadata(path, object.size, object.updatedAt)
}

function assetMetadata(path: string, size: number, updatedAt: Date): ProblemAssetItem {
  const contentType = inferContentType(path)
  return {
    path,
    name: path.split('/').at(-1) ?? path,
    section: assetSection(path),
    size,
    contentType,
    updatedAt: updatedAt.toISOString(),
    text: isTextAsset(path, contentType)
  }
}

function compareAssets(left: ProblemAssetItem, right: ProblemAssetItem) {
  const order: Record<ProblemAssetSection, number> = { data: 0, assets: 1, root: 2 }
  return order[left.section] - order[right.section] || left.path.localeCompare(right.path)
}

function throwAssetPathError(message = 'Invalid asset path'): never {
  throw new ApiHttpError(400, 'INVALID_ASSET_PATH', message, [{ path: 'path', message }])
}

function isFlatFilename(filename: string) {
  return Boolean(filename) && !filename.includes('/') && !filename.includes('\\') && filename !== '.' && filename !== '..'
}

function sanitizeFilename(name: string) {
  return (name || 'upload').replace(/[^a-zA-Z0-9._-]/g, '_')
}

function problemObjectKey(problemId: number, path: string) {
  return `problems/${problemId}/${path}`
}

function inferContentType(path: string) {
  const lower = path.toLowerCase()
  if (lower.endsWith('.png')) return 'image/png'
  if (lower.endsWith('.jpg') || lower.endsWith('.jpeg')) return 'image/jpeg'
  if (lower.endsWith('.gif')) return 'image/gif'
  if (lower.endsWith('.webp')) return 'image/webp'
  if (lower.endsWith('.svg')) return 'image/svg+xml'
  if (lower.endsWith('.md')) return 'text/markdown; charset=utf-8'
  if (isTextAsset(path, '')) return 'text/plain; charset=utf-8'
  return 'application/octet-stream'
}

function isTextAsset(path: string, contentType: string) {
  if (contentType.startsWith('text/') || contentType.includes('json') || contentType.includes('xml')) return true
  const lower = path.toLowerCase()
  return [...textExtensions].some((extension) => lower.endsWith(extension))
}

function problemSearchFilter(q: string) {
  if (/^\d+$/.test(q)) {
    return or(eq(schema.problems.id, Number(q)), ilike(schema.problems.title, `%${q}%`))
  }
  return or(ilike(schema.problems.title, `%${q}%`), sql`${q} = any(${schema.problems.tags})`)
}

const problemModeSchema = z.enum(['default', 'strict', 'custom'])
const tagsSchema = z.array(z.string().min(1).max(32)).max(10).default([])

const createProblemSchema = z.object({
  title: z.string().min(1).max(100),
  mode: problemModeSchema.default('default'),
  timeLimit: z.number().int().min(100).max(60000).default(1000),
  memoryLimit: z.number().int().min(16_777_216).max(1_073_741_824).default(134_217_728),
  tags: tagsSchema,
  visible: z.boolean().default(false)
})

const updateProblemSchema = z
  .object({
    title: z.string().min(1).max(100).optional(),
    mode: problemModeSchema.optional(),
    timeLimit: z.number().int().min(100).max(60000).optional(),
    memoryLimit: z.number().int().min(16_777_216).max(1_073_741_824).optional(),
    tags: z.array(z.string().min(1).max(32)).max(10).optional(),
    visible: z.boolean().optional()
  })
  .refine((value) => Object.keys(value).length > 0, { message: 'At least one field must be updated' })

const statementSchema = z.object({ markdown: z.string() })
const assetPathSchema = z.string().min(1).max(255)
const assetContentSchema = z.object({
  path: assetPathSchema,
  content: z.string(),
  encoding: z.enum(['utf8', 'base64']),
  contentType: z.string().min(1).max(200).optional()
})

const problemListQuerySchema = listQuerySchema.extend({
  q: z.string().trim().max(120).optional().default(''),
  tag: z.string().trim().max(32).optional().default('')
})
