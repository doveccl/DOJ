import { Hono, type Context } from 'hono'
import { asc, desc, eq, inArray } from 'drizzle-orm'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { buildStoredZip, parseLooseTestCases, parseZipTestCases } from '@doj/shared/testdata'
import { putObject, getObjectBytes, storageConfig } from '@doj/shared/storage'
import { authMiddleware, getOptionalAuthUser, requireGroup } from '../auth'
import { getRuntimeSettings } from '../settings'
import { countRows } from '../services/stats'
import { listQuerySchema, numericId, pageOffset } from '../validation'

const maxTestdataUploadBytes = 64 * 1024 * 1024

export function registerProblemRoutes(app: Hono) {
  app.get('/api/problems', async (c) => {
    const denied = await denyGuestProblemset(c)
    if (denied) return denied

    const { page, pageSize } = listQuerySchema.parse(c.req.query())
    const visible = eq(schema.problems.visible, true)
    const total = await countRows(schema.problems, visible)
    const list = await db
      .select()
      .from(schema.problems)
      .where(visible)
      .orderBy(asc(schema.problems.id))
      .limit(pageSize)
      .offset(pageOffset(page, pageSize))
    return c.json({ total, page, pageSize, list })
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
    const versions = await getLatestProblemVersions(list.map((problem) => problem.id))
    const enriched = list.map((problem) => ({
      ...problem,
      version: summarizeProblemVersion(versions.get(problem.id) ?? null)
    }))
    return c.json({ total, page, pageSize, list: enriched })
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
    const uploads = form.getAll('file').filter((item): item is File => item instanceof File)
    if (!uploads.length) {
      return c.json(
        { code: 'MISSING_FILE', message: 'Expected multipart file field named file' },
        400
      )
    }

    const totalBytes = uploads.reduce((sum, file) => sum + file.size, 0)
    if (totalBytes > maxTestdataUploadBytes) {
      return c.json({ code: 'FILE_TOO_LARGE', message: 'Testdata is too large' }, 413)
    }

    const isSingleZip =
      uploads.length === 1 &&
      (uploads[0].name.toLowerCase().endsWith('.zip') ||
        uploads[0].type === 'application/zip' ||
        uploads[0].type === 'application/x-zip-compressed')

    let bytes: Uint8Array
    let cases
    let filename: string
    try {
      if (isSingleZip) {
        bytes = new Uint8Array(await uploads[0].arrayBuffer())
        cases = parseZipTestCases(bytes)
        filename = uploads[0].name || `problem-${problemId}-testdata.zip`
      } else {
        // Loose files: classify directly, then repackage into a stored ZIP so the
        // agent's existing ZIP-based fetch/parse path stays unchanged.
        const entries = await Promise.all(
          uploads.map(async (file) => ({
            name: file.name,
            bytes: new Uint8Array(await file.arrayBuffer())
          }))
        )
        cases = parseLooseTestCases(entries)
        bytes = buildStoredZip(entries)
        filename = `problem-${problemId}-testdata.zip`
      }
    } catch (cause) {
      return c.json(
        {
          code: 'INVALID_TESTDATA',
          message: cause instanceof Error ? cause.message : 'Invalid testdata'
        },
        400
      )
    }
    if (!cases.length) {
      return c.json(
        { code: 'EMPTY_TESTDATA', message: 'Testdata must contain .in/.out case pairs' },
        400
      )
    }

    const objectKey = `problems/${problemId}/testdata/${crypto.randomUUID()}.zip`
    await putObject({
      key: objectKey,
      body: bytes,
      contentType: 'application/zip'
    })

    const [file] = await db
      .insert(schema.files)
      .values({
        bucket: storageConfig.bucket,
        objectKey,
        filename,
        contentType: 'application/zip',
        sizeBytes: bytes.byteLength,
        metadata: createTestdataMetadata(problemId, cases)
      })
      .returning()

    const [updated] = await db
      .update(schema.problemVersions)
      .set({ testdataFileId: file.id })
      .where(eq(schema.problemVersions.id, version.id))
      .returning()

    return c.json({ file, version: updated, caseCount: cases.length }, 201)
  })

  app.post('/api/problems/:id/checker', authMiddleware, async (c) => {
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

    const body = checkerSchema.parse(await c.req.json())

    if (body.sourceCode === null) {
      const [updated] = await db
        .update(schema.problemVersions)
        .set({ checkerFileId: null })
        .where(eq(schema.problemVersions.id, version.id))
        .returning()
      return c.json({ version: updated, checkerEnabled: false })
    }

    const bytes = new TextEncoder().encode(body.sourceCode)
    const objectKey = `problems/${problemId}/checker/${crypto.randomUUID()}.cc`
    await putObject({ key: objectKey, body: bytes, contentType: 'text/x-c++src' })

    const [file] = await db
      .insert(schema.files)
      .values({
        bucket: storageConfig.bucket,
        objectKey,
        filename: 'checker.cc',
        contentType: 'text/x-c++src',
        sizeBytes: bytes.byteLength
      })
      .returning()

    const [updated] = await db
      .update(schema.problemVersions)
      .set({ checkerFileId: file.id })
      .where(eq(schema.problemVersions.id, version.id))
      .returning()

    return c.json({ file, version: updated, checkerEnabled: true }, 201)
  })
}

const checkerSchema = z.object({
  sourceCode: z.string().min(1).max(200_000).nullable()
})

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

  const version = await getLatestProblemVersion(problem.id)

  const checkerSource =
    options.includeHiddenProblem && version?.checkerFileId
      ? await readFileText(version.checkerFileId)
      : undefined

  return {
    problem,
    version: version
      ? {
          ...version,
          testdata: summarizeTestdata(version),
          checkerEnabled: Boolean(version.checkerFileId),
          checkerSource,
          interactorEnabled: Boolean(version.interactorFileId),
          testCases: options.includeHiddenCases
            ? version.testCases
            : version.testCases.filter((testCase) => !testCase.hidden)
        }
      : null
  }
}

async function readFileText(fileId: number) {
  const [file] = await db.select().from(schema.files).where(eq(schema.files.id, fileId)).limit(1)
  if (!file) return undefined
  const bytes = await getObjectBytes(file.objectKey, file.bucket)
  return Buffer.from(bytes).toString('utf8')
}

type ProblemVersion = typeof schema.problemVersions.$inferSelect
type StoredFile = typeof schema.files.$inferSelect
type ProblemVersionWithFile = ProblemVersion & { testdataFile: StoredFile | null }

async function getLatestProblemVersion(problemId: number): Promise<ProblemVersionWithFile | null> {
  const [version] = await db
    .select()
    .from(schema.problemVersions)
    .where(eq(schema.problemVersions.problemId, problemId))
    .orderBy(desc(schema.problemVersions.version))
    .limit(1)

  if (!version) return null

  const [testdataFile] = version.testdataFileId
    ? await db
        .select()
        .from(schema.files)
        .where(eq(schema.files.id, version.testdataFileId))
        .limit(1)
    : []

  return { ...version, testdataFile: testdataFile ?? null }
}

// Batched variant for list endpoints: one query for the latest version per
// problem (DISTINCT ON), one query for the referenced testdata files.
async function getLatestProblemVersions(
  problemIds: number[]
): Promise<Map<number, ProblemVersionWithFile>> {
  const result = new Map<number, ProblemVersionWithFile>()
  if (!problemIds.length) return result

  const versions = await db
    .selectDistinctOn([schema.problemVersions.problemId])
    .from(schema.problemVersions)
    .where(inArray(schema.problemVersions.problemId, problemIds))
    .orderBy(desc(schema.problemVersions.problemId), desc(schema.problemVersions.version))

  const fileIds = [
    ...new Set(versions.map((version) => version.testdataFileId).filter((id): id is number => !!id))
  ]
  const files = fileIds.length
    ? await db.select().from(schema.files).where(inArray(schema.files.id, fileIds))
    : []
  const fileById = new Map(files.map((file) => [file.id, file]))

  for (const version of versions) {
    result.set(version.problemId, {
      ...version,
      testdataFile: version.testdataFileId ? (fileById.get(version.testdataFileId) ?? null) : null
    })
  }
  return result
}

function summarizeProblemVersion(version: ProblemVersionWithFile | null) {
  if (!version) return null
  return {
    id: version.id,
    version: version.version,
    timeLimitMs: version.timeLimitMs,
    memoryLimitBytes: version.memoryLimitBytes,
    testdata: summarizeTestdata(version),
    checkerEnabled: Boolean(version.checkerFileId),
    interactorEnabled: Boolean(version.interactorFileId),
    createdAt: version.createdAt
  }
}

function summarizeTestdata(version: ProblemVersionWithFile) {
  const metadata = readTestdataMetadata(version.testdataFile?.metadata)
  const inlineCases = version.testCases.map((testCase, index) => ({
    name: testCase.name ?? String(index + 1),
    inputBytes: byteLength(testCase.input),
    outputBytes: byteLength(testCase.output)
  }))
  const cases = metadata.cases.length ? metadata.cases : inlineCases
  const totalInputBytes =
    metadata.totalInputBytes ?? cases.reduce((total, item) => total + item.inputBytes, 0)
  const totalOutputBytes =
    metadata.totalOutputBytes ?? cases.reduce((total, item) => total + item.outputBytes, 0)

  return {
    mode: version.testdataFile ? 'zip' : cases.length ? 'inline' : 'none',
    caseCount: metadata.caseCount ?? cases.length,
    cases,
    totalInputBytes,
    totalOutputBytes,
    file: version.testdataFile
      ? {
          id: version.testdataFile.id,
          filename: version.testdataFile.filename,
          contentType: version.testdataFile.contentType,
          sizeBytes: version.testdataFile.sizeBytes,
          createdAt: version.testdataFile.createdAt
        }
      : null
  }
}

function createTestdataMetadata(
  problemId: number,
  cases: Array<{ name?: string; input: string; output: string }>
) {
  const caseSummaries = cases.map((testCase, index) => ({
    name: testCase.name ?? String(index + 1),
    inputBytes: byteLength(testCase.input),
    outputBytes: byteLength(testCase.output)
  }))

  return {
    problemId,
    kind: 'problem-testdata',
    caseCount: cases.length,
    cases: caseSummaries,
    totalInputBytes: caseSummaries.reduce((total, item) => total + item.inputBytes, 0),
    totalOutputBytes: caseSummaries.reduce((total, item) => total + item.outputBytes, 0)
  }
}

function readTestdataMetadata(metadata: Record<string, unknown> | undefined) {
  const caseCount = typeof metadata?.caseCount === 'number' ? metadata.caseCount : undefined
  const totalInputBytes =
    typeof metadata?.totalInputBytes === 'number' ? metadata.totalInputBytes : undefined
  const totalOutputBytes =
    typeof metadata?.totalOutputBytes === 'number' ? metadata.totalOutputBytes : undefined
  const cases = Array.isArray(metadata?.cases)
    ? metadata.cases.flatMap((item) => {
        if (!item || typeof item !== 'object') return []
        const candidate = item as Record<string, unknown>
        if (
          typeof candidate.name !== 'string' ||
          typeof candidate.inputBytes !== 'number' ||
          typeof candidate.outputBytes !== 'number'
        ) {
          return []
        }
        return [
          {
            name: candidate.name,
            inputBytes: candidate.inputBytes,
            outputBytes: candidate.outputBytes
          }
        ]
      })
    : []

  return { caseCount, cases, totalInputBytes, totalOutputBytes }
}

function byteLength(value: string) {
  return new TextEncoder().encode(value).byteLength
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
