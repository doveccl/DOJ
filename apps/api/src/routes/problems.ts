import { Hono, type Context } from 'hono'
import { and, asc, eq, ilike, sql } from 'drizzle-orm'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { putObject, getObjectBytes, storageConfig } from '@doj/shared/storage'
import { authMiddleware, getOptionalAuthUser, requireGroup } from '../auth'
import { getRuntimeSettings } from '../settings'
import { countRows } from '../services/stats'
import { listQuerySchema, numericId, pageOffset } from '../validation'

const maxPackageFileBytes = 64 * 1024 * 1024
const maxPackageRequestBytes = maxPackageFileBytes + 1024 * 1024

export function registerProblemRoutes(app: Hono) {
  app.get('/api/problems', async (c) => {
    const denied = await denyGuestProblemset(c)
    if (denied) return denied

    const authUser = await getOptionalAuthUser(c)
    const { page, pageSize, search, tag } = problemListQuerySchema.parse(c.req.query())
    const where = and(
      eq(schema.problems.visible, true),
      search ? ilike(schema.problems.title, `%${search}%`) : undefined,
      tag ? sql`${tag} = any(${schema.problems.tags})` : undefined
    )
    const total = await countRows(schema.problems, where)
    const list = await db
      .select({
        id: schema.problems.id,
        title: schema.problems.title,
        tags: schema.problems.tags,
        solvedCount: schema.problems.solvedCount,
        submissionCount: schema.problems.submissionCount,
        createdAt: schema.problems.createdAt,
        solved: authUser
          ? sql<boolean>`exists (
              select 1 from ${schema.solvedProblems}
              where ${schema.solvedProblems.userId} = ${authUser.id}
                and ${schema.solvedProblems.problemId} = ${schema.problems.id}
            )`
          : sql<boolean>`false`
      })
      .from(schema.problems)
      .where(where)
      .orderBy(asc(schema.problems.id))
      .limit(pageSize)
      .offset(pageOffset(page, pageSize))
    const tags = await listVisibleTags()
    return c.json({ total, page, pageSize, list, tags })
  })

  app.get('/api/admin/problems', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const { page, pageSize } = listQuerySchema.parse(c.req.query())
    const total = await countRows(schema.problems)
    const list = await db
      .select()
      .from(schema.problems)
      .orderBy(asc(schema.problems.id))
      .limit(pageSize)
      .offset(pageOffset(page, pageSize))
    const enriched = list.map((problem) => ({
      ...problem,
      summary: summarizeProblem(problem)
    }))
    return c.json({ total, page, pageSize, list: enriched })
  })

  app.get('/api/admin/problems/:id', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const detail = await getProblemDetail(numericId.parse(c.req.param('id')), {
      includeHiddenCases: true,
      includeHiddenProblem: true,
      includePackage: true
    })
    if (!detail) return c.notFound()
    return c.json(detail)
  })

  app.get('/api/problems/:id', async (c) => {
    const denied = await denyGuestProblemset(c)
    if (denied) return denied

    const detail = await getProblemDetail(numericId.parse(c.req.param('id')))
    if (!detail) return c.notFound()
    return c.json(detail)
  })

  app.post('/api/problems', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const body = createProblemSchema.parse(await c.req.json())
    const [problem] = await db
      .insert(schema.problems)
      .values({
        title: body.title,
        tags: body.tags,
        statementMarkdown: body.statementMarkdown,
        timeLimitMs: body.timeLimitMs,
        memoryLimitBytes: body.memoryLimitBytes,
        caseCount: body.caseCount,
        testCases: body.testCases
      })
      .returning()

    return c.json({ problem }, 201)
  })

  app.patch('/api/problems/:id', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const body = updateProblemSchema.parse(await c.req.json())

    const patch: Partial<typeof schema.problems.$inferInsert> = { updatedAt: new Date() }
    if ('title' in body) patch.title = body.title
    if ('tags' in body) patch.tags = body.tags ?? []
    if ('visible' in body) patch.visible = body.visible
    if ('statementMarkdown' in body) patch.statementMarkdown = body.statementMarkdown
    if ('timeLimitMs' in body) patch.timeLimitMs = body.timeLimitMs
    if ('memoryLimitBytes' in body) patch.memoryLimitBytes = body.memoryLimitBytes
    if ('caseCount' in body) patch.caseCount = body.caseCount
    if ('testCases' in body) patch.testCases = body.testCases

    const [updated] = await db
      .update(schema.problems)
      .set(patch)
      .where(eq(schema.problems.id, id))
      .returning()
    if (!updated) return c.notFound()

    const detail = await getProblemDetail(id, {
      includeHiddenCases: true,
      includeHiddenProblem: true,
      includePackage: true
    })
    return c.json(detail)
  })

  // List the package files (path + size, no content) for a problem.
  app.get('/api/problems/:id/package', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const problemId = numericId.parse(c.req.param('id'))
    const files = await listPackageFiles(problemId)
    return c.json({ files })
  })

  // Read a single package file's text content (for the editor).
  app.get('/api/problems/:id/package/content', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const problemId = numericId.parse(c.req.param('id'))
    const path = z.string().min(1).max(255).parse(c.req.query('path'))
    const stored = await getPackageFile(problemId, path)
    if (!stored) return c.notFound()
    const bytes = await getObjectBytes(stored.objectKey, stored.bucket)
    return c.json({ path, content: Buffer.from(bytes).toString('utf8') })
  })

  // Create or overwrite a package file (text content from the editor).
  app.put('/api/problems/:id/package', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const problemId = numericId.parse(c.req.param('id'))
    const body = packageFileSchema.parse(await c.req.json())
    const bytes = new TextEncoder().encode(body.content)
    if (bytes.byteLength > maxPackageFileBytes) {
      return c.json({ code: 'FILE_TOO_LARGE', message: 'Package file is too large' }, 413)
    }
    const file = await upsertPackageFile(problemId, body.path, bytes, 'text/plain')
    return c.json({ file }, 201)
  })

  // Upload package files as raw uploads (data files, binaries, assets).
  app.post('/api/problems/:id/package/upload', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const problemId = numericId.parse(c.req.param('id'))
    const contentLength = Number(c.req.header('content-length') ?? 0)
    if (Number.isFinite(contentLength) && contentLength > maxPackageRequestBytes) {
      return c.json({ code: 'FILE_TOO_LARGE', message: 'Package upload is too large' }, 413)
    }

    const form = await c.req.formData()
    const uploads = form.getAll('file').filter((item): item is File => item instanceof File)
    if (!uploads.length) {
      return c.json(
        { code: 'MISSING_FILE', message: 'Expected multipart file field named file' },
        400
      )
    }
    const prefix = (form.get('prefix') as string | null) ?? ''
    const totalBytes = uploads.reduce((sum, file) => sum + file.size, 0)
    if (totalBytes > maxPackageFileBytes) {
      return c.json({ code: 'FILE_TOO_LARGE', message: 'Package upload is too large' }, 413)
    }

    const saved = []
    for (const upload of uploads) {
      const path = normalizePackagePath(`${prefix}${upload.name}`)
      const bytes = new Uint8Array(await upload.arrayBuffer())
      const file = await upsertPackageFile(
        problemId,
        path,
        bytes,
        upload.type || 'application/octet-stream'
      )
      saved.push(file)
    }
    return c.json({ files: saved }, 201)
  })

  app.delete('/api/problems/:id/package', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const problemId = numericId.parse(c.req.param('id'))
    const path = z.string().min(1).max(255).parse(c.req.query('path'))
    await deletePackageFile(problemId, path)
    return c.json({ ok: true })
  })
}

async function denyGuestProblemset(c: Context) {
  const settings = await getRuntimeSettings()
  if (settings.guestProblemsetVisible) return null
  const authUser = await getOptionalAuthUser(c)
  if (authUser) return null
  return c.json({ code: 'UNAUTHORIZED', message: 'Sign in to view the problemset' }, 401)
}

type Problem = typeof schema.problems.$inferSelect

async function getProblemDetail(
  id: number,
  options: {
    includeHiddenCases?: boolean
    includeHiddenProblem?: boolean
    includePackage?: boolean
  } = {}
) {
  const [problem] = await db
    .select()
    .from(schema.problems)
    .where(eq(schema.problems.id, id))
    .limit(1)
  if (!problem || (!problem.visible && !options.includeHiddenProblem)) return null

  const packageFiles = options.includePackage ? await listPackageFiles(id) : undefined

  return {
    problem,
    summary: summarizeProblem(problem),
    package: packageFiles,
    testCases: options.includeHiddenCases
      ? problem.testCases
      : problem.testCases.filter((testCase) => !testCase.hidden)
  }
}

function summarizeProblem(problem: Problem) {
  return {
    timeLimitMs: problem.timeLimitMs,
    memoryLimitBytes: problem.memoryLimitBytes,
    caseCount: problem.caseCount,
    inlineCaseCount: problem.testCases.length
  }
}

async function listPackageFiles(problemId: number) {
  const rows = await db
    .select({
      path: schema.problemFiles.path,
      filename: schema.files.filename,
      contentType: schema.files.contentType,
      sizeBytes: schema.files.sizeBytes,
      updatedAt: schema.problemFiles.updatedAt
    })
    .from(schema.problemFiles)
    .innerJoin(schema.files, eq(schema.problemFiles.fileId, schema.files.id))
    .where(eq(schema.problemFiles.problemId, problemId))
  return rows.sort((a, b) => a.path.localeCompare(b.path))
}

async function getPackageFile(problemId: number, path: string) {
  const rows = await db
    .select({
      path: schema.problemFiles.path,
      bucket: schema.files.bucket,
      objectKey: schema.files.objectKey
    })
    .from(schema.problemFiles)
    .innerJoin(schema.files, eq(schema.problemFiles.fileId, schema.files.id))
    .where(eq(schema.problemFiles.problemId, problemId))
  return rows.find((row) => row.path === path) ?? null
}

async function upsertPackageFile(
  problemId: number,
  path: string,
  bytes: Uint8Array,
  contentType: string
) {
  const normalized = normalizePackagePath(path)
  const objectKey = `problems/${problemId}/package/${crypto.randomUUID()}`
  await putObject({ key: objectKey, body: bytes, contentType })

  const [file] = await db
    .insert(schema.files)
    .values({
      bucket: storageConfig.bucket,
      objectKey,
      filename: normalized.split('/').at(-1) ?? normalized,
      contentType,
      sizeBytes: bytes.byteLength
    })
    .returning()

  await db
    .insert(schema.problemFiles)
    .values({ problemId, path: normalized, fileId: file.id })
    .onConflictDoUpdate({
      target: [schema.problemFiles.problemId, schema.problemFiles.path],
      set: { fileId: file.id, updatedAt: new Date() }
    })

  return { path: normalized, filename: file.filename, sizeBytes: file.sizeBytes }
}

async function deletePackageFile(problemId: number, path: string) {
  const normalized = normalizePackagePath(path)
  const existing = await db
    .select()
    .from(schema.problemFiles)
    .where(eq(schema.problemFiles.problemId, problemId))
  const match = existing.find((row) => row.path === normalized)
  if (!match) return
  await db.delete(schema.problemFiles).where(eq(schema.problemFiles.fileId, match.fileId))
}

// Restrict to a safe relative path inside the build context.
function normalizePackagePath(path: string) {
  const cleaned = path.replace(/\\/g, '/').replace(/^\/+/, '').replace(/\.\.+/g, '.').trim()
  if (!cleaned) throw new Error('invalid package path')
  return cleaned
}

const problemTestCaseSchema = z.object({
  name: z.string().max(120).optional(),
  input: z.string().max(256 * 1024),
  output: z.string().max(256 * 1024),
  hidden: z.boolean().default(false),
  points: z.number().int().min(0).max(100).optional()
})

const createProblemSchema = z.object({
  title: z.string().min(1).max(160),
  tags: z.array(z.string()).default([]),
  statementMarkdown: z.string().min(1),
  timeLimitMs: z.number().int().positive().default(1000),
  memoryLimitBytes: z
    .number()
    .int()
    .positive()
    .default(256 * 1024 * 1024),
  caseCount: z.number().int().min(0).max(1000).default(0),
  testCases: z.array(problemTestCaseSchema).max(100).default([])
})

const updateProblemSchema = z
  .object({
    title: z.string().min(1).max(160).optional(),
    tags: z.array(z.string()).optional(),
    visible: z.boolean().optional(),
    statementMarkdown: z.string().min(1).optional(),
    timeLimitMs: z.number().int().positive().optional(),
    memoryLimitBytes: z.number().int().positive().optional(),
    caseCount: z.number().int().min(0).max(1000).optional(),
    testCases: z.array(problemTestCaseSchema).max(100).optional()
  })
  .refine((value) => Object.keys(value).length > 0, {
    message: 'At least one field must be updated'
  })

const packageFileSchema = z.object({
  path: z.string().min(1).max(255),
  content: z.string().max(2 * 1024 * 1024)
})

const problemListQuerySchema = listQuerySchema.extend({
  search: z.string().trim().max(120).optional().default(''),
  tag: z.string().trim().max(80).optional().default('')
})

async function listVisibleTags() {
  const rows = await db
    .select({ tag: sql<string>`unnest(${schema.problems.tags})` })
    .from(schema.problems)
    .where(eq(schema.problems.visible, true))
  return [...new Set(rows.map((row) => row.tag).filter(Boolean))].sort((a, b) => a.localeCompare(b))
}
