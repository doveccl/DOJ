import { Hono, type Context } from 'hono'
import { asc, desc, eq } from 'drizzle-orm'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { parseZipTestCases } from '@doj/shared/testdata'
import { putObject, storageConfig } from '@doj/shared/storage'
import { authMiddleware, getOptionalAuthUser, requireGroup } from '../auth'
import { getRuntimeSettings } from '../settings'
import { numericId } from '../validation'

const maxTestdataUploadBytes = 64 * 1024 * 1024

export function registerProblemRoutes(app: Hono) {
  app.get('/api/problems', async (c) => {
    const denied = await denyGuestProblemset(c)
    if (denied) return denied

    const list = await db
      .select()
      .from(schema.problems)
      .where(eq(schema.problems.visible, true))
      .orderBy(asc(schema.problems.id))
      .limit(50)
    return c.json({ total: list.length, list })
  })

  app.get('/api/admin/problems', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const list = await db.select().from(schema.problems).orderBy(asc(schema.problems.id)).limit(100)
    return c.json({ total: list.length, list })
  })

  app.get('/api/admin/problems/:id', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const detail = await getProblemDetail(numericId.parse(c.req.param('id')), {
      includeHiddenCases: true,
      includeHiddenProblem: true
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

  app.patch('/api/problems/:id', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const id = numericId.parse(c.req.param('id'))
    const body = updateProblemSchema.parse(await c.req.json())
    const detail = await getProblemDetail(id, {
      includeHiddenCases: true,
      includeHiddenProblem: true
    })
    if (!detail) return c.notFound()

    const versionFieldsChanged = [
      'statementMarkdown',
      'timeLimitMs',
      'memoryLimitBytes',
      'testCases'
    ].some((key) => key in body)

    await db.transaction(async (tx) => {
      const problemPatch: Partial<typeof schema.problems.$inferInsert> = {}
      if ('title' in body) problemPatch.title = body.title
      if ('tags' in body) problemPatch.tags = body.tags ?? []
      if ('visible' in body) problemPatch.visible = body.visible
      if (Object.keys(problemPatch).length) {
        problemPatch.updatedAt = new Date()
        await tx.update(schema.problems).set(problemPatch).where(eq(schema.problems.id, id))
      }

      if (versionFieldsChanged && detail.version) {
        await tx.insert(schema.problemVersions).values({
          problemId: id,
          version: detail.version.version + 1,
          statementMarkdown: body.statementMarkdown ?? detail.version.statementMarkdown,
          timeLimitMs: body.timeLimitMs ?? detail.version.timeLimitMs,
          memoryLimitBytes: body.memoryLimitBytes ?? detail.version.memoryLimitBytes,
          outputLimitBytes: detail.version.outputLimitBytes,
          testCases: body.testCases ?? detail.version.testCases,
          testdataFileId: 'testCases' in body ? null : detail.version.testdataFileId,
          checkerFileId: detail.version.checkerFileId,
          interactorFileId: detail.version.interactorFileId
        })
      }
    })

    const updated = await getProblemDetail(id, {
      includeHiddenCases: true,
      includeHiddenProblem: true
    })
    return c.json(updated)
  })

  app.post('/api/problems', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const settings = await getRuntimeSettings()
    const body = createProblemSchema.parse(await c.req.json())

    const result = await db.transaction(async (tx) => {
      const [problem] = await tx
        .insert(schema.problems)
        .values({
          title: body.title,
          tags: body.tags
        })
        .returning()

      const [version] = await tx
        .insert(schema.problemVersions)
        .values({
          problemId: problem.id,
          version: 1,
          statementMarkdown: body.statementMarkdown,
          timeLimitMs: body.timeLimitMs,
          memoryLimitBytes: body.memoryLimitBytes,
          outputLimitBytes: settings.outputLimitBytes,
          testCases: body.testCases
        })
        .returning()

      return { problem, version }
    })

    return c.json(result, 201)
  })

  app.post('/api/problems/:id/testdata', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const problemId = numericId.parse(c.req.param('id'))
    const [version] = await db
      .select()
      .from(schema.problemVersions)
      .where(eq(schema.problemVersions.problemId, problemId))
      .orderBy(desc(schema.problemVersions.version))
      .limit(1)
    if (!version) return c.notFound()

    const form = await c.req.formData()
    const upload = form.get('file')
    if (!(upload instanceof File)) {
      return c.json(
        { code: 'MISSING_FILE', message: 'Expected multipart file field named file' },
        400
      )
    }

    const bytes = new Uint8Array(await upload.arrayBuffer())
    if (bytes.byteLength > maxTestdataUploadBytes) {
      return c.json({ code: 'FILE_TOO_LARGE', message: 'Testdata ZIP is too large' }, 413)
    }

    let cases
    try {
      cases = parseZipTestCases(bytes)
    } catch (cause) {
      return c.json(
        {
          code: 'INVALID_TESTDATA_ZIP',
          message: cause instanceof Error ? cause.message : 'Invalid ZIP testdata'
        },
        400
      )
    }
    if (!cases.length) {
      return c.json(
        { code: 'EMPTY_TESTDATA', message: 'ZIP must contain .in/.out case pairs' },
        400
      )
    }

    const objectKey = `problems/${problemId}/testdata/${crypto.randomUUID()}.zip`
    await putObject({
      key: objectKey,
      body: bytes,
      contentType: upload.type || 'application/zip'
    })

    const [file] = await db
      .insert(schema.files)
      .values({
        bucket: storageConfig.bucket,
        objectKey,
        filename: upload.name || `problem-${problemId}-testdata.zip`,
        contentType: upload.type || 'application/zip',
        sizeBytes: bytes.byteLength,
        metadata: {
          problemId,
          caseCount: cases.length,
          kind: 'problem-testdata'
        }
      })
      .returning()

    const [updated] = await db
      .update(schema.problemVersions)
      .set({ testdataFileId: file.id })
      .where(eq(schema.problemVersions.id, version.id))
      .returning()

    return c.json({ file, version: updated, caseCount: cases.length }, 201)
  })
}

async function denyGuestProblemset(c: Context) {
  const settings = await getRuntimeSettings()
  if (settings.guestProblemsetVisible) return null
  const authUser = await getOptionalAuthUser(c)
  if (authUser) return null
  return c.json({ code: 'UNAUTHORIZED', message: 'Sign in to view the problemset' }, 401)
}

async function getProblemDetail(
  id: number,
  options: { includeHiddenCases?: boolean; includeHiddenProblem?: boolean } = {}
) {
  const [problem] = await db
    .select()
    .from(schema.problems)
    .where(eq(schema.problems.id, id))
    .limit(1)
  if (!problem || (!problem.visible && !options.includeHiddenProblem)) return null

  const [version] = await db
    .select()
    .from(schema.problemVersions)
    .where(eq(schema.problemVersions.problemId, problem.id))
    .orderBy(desc(schema.problemVersions.version))
    .limit(1)

  return {
    problem,
    version: version
      ? {
          ...version,
          testCases: options.includeHiddenCases
            ? version.testCases
            : version.testCases.filter((testCase) => !testCase.hidden)
        }
      : null
  }
}

const problemTestCaseSchema = z.object({
  name: z.string().max(120).optional(),
  input: z.string().max(256 * 1024),
  output: z.string().max(256 * 1024),
  hidden: z.boolean().default(false)
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
    testCases: z.array(problemTestCaseSchema).max(100).optional()
  })
  .refine((value) => Object.keys(value).length > 0, {
    message: 'At least one field must be updated'
  })
