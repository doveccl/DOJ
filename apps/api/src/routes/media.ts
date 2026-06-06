import { Hono } from 'hono'
import { eq } from 'drizzle-orm'
import { db, schema } from '@doj/db/client'
import { getObjectBytes, putObject, storageConfig } from '@doj/shared/storage'
import { authMiddleware, requireAuthUser } from '../auth'
import { numericId } from '../validation'

const maxMediaUploadBytes = 8 * 1024 * 1024
const allowedImageTypes = new Set([
  'image/png',
  'image/jpeg',
  'image/gif',
  'image/webp',
  'image/svg+xml'
])

export function registerMediaRoutes(app: Hono) {
  // Authenticated upload for markdown images / problem attachments.
  app.post('/api/media', authMiddleware, async (c) => {
    const user = await requireAuthUser(c)
    const form = await c.req.formData()
    const upload = form.get('file')
    if (!(upload instanceof File)) {
      return c.json(
        { code: 'MISSING_FILE', message: 'Expected multipart file field named file' },
        400
      )
    }
    if (upload.size > maxMediaUploadBytes) {
      return c.json({ code: 'FILE_TOO_LARGE', message: 'File is too large' }, 413)
    }
    if (!allowedImageTypes.has(upload.type)) {
      return c.json({ code: 'UNSUPPORTED_TYPE', message: 'Unsupported image type' }, 415)
    }

    const bytes = new Uint8Array(await upload.arrayBuffer())
    const objectKey = `media/${user.id}/${crypto.randomUUID()}`
    await putObject({ key: objectKey, body: bytes, contentType: upload.type })

    const [file] = await db
      .insert(schema.files)
      .values({
        bucket: storageConfig.bucket,
        objectKey,
        filename: upload.name || 'upload',
        contentType: upload.type,
        sizeBytes: bytes.byteLength,
        metadata: { kind: 'media', uploadedBy: user.id }
      })
      .returning()

    return c.json({ id: file.id, url: `/api/files/${file.id}`, filename: file.filename }, 201)
  })

  // Public read; objects are addressed by opaque auto-increment id.
  app.get('/api/files/:id', async (c) => {
    const id = numericId.parse(c.req.param('id'))
    const [file] = await db.select().from(schema.files).where(eq(schema.files.id, id)).limit(1)
    if (!file) return c.notFound()

    const bytes = await getObjectBytes(file.objectKey, file.bucket)
    c.header('content-type', file.contentType)
    c.header('cache-control', 'public, max-age=31536000, immutable')
    return c.body(bytes)
  })
}
