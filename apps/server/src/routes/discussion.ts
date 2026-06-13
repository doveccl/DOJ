import { Hono, type Context } from 'hono'
import { eq } from 'drizzle-orm'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { authMiddleware, denyGuestAccess, requireAuthUser, requireGroup } from '../auth'
import { notFound } from '../errors'
import { checkRateLimit } from '../rate-limit'
import { getRecentTopics, getTopicDetail, countTopics } from '../services/discussion'
import { listQuerySchema, numericId, pageOffset } from '../validation'

export function registerDiscussionRoutes(app: Hono) {
  app.get('/api/discussion/topics', async (c) => {
    const denied = await denyGuestAccess(c, 'Sign in to view discussions')
    if (denied) return denied

    const { page, pageSize, tags: rawTags } = discussionListQuerySchema.parse(c.req.query())
    const tags = parseTags(rawTags)
    const [total, items] = await Promise.all([
      countTopics(tags),
      getRecentTopics(pageSize, pageOffset(page, pageSize), tags)
    ])

    return c.json({ items, page, pageSize, total })
  })

  app.post('/api/discussion/topics', authMiddleware, async (c) => {
    const user = await requireAuthUser(c)
    const rateLimited = await checkRateLimit(
      c,
      'discussion:topic',
      `user:${user.id}`,
      30,
      60 * 60 * 1000
    )
    if (rateLimited) return rateLimited

    const body = createTopicSchema.parse(await c.req.json())

    const topic = await db.transaction(async (tx) => {
      const [created] = await tx
        .insert(schema.topics)
        .values({
          title: body.title,
          tags: body.tags
        })
        .returning()

      await tx.insert(schema.posts).values({
        topicId: created.id,
        userId: user.id,
        content: body.content
      })

      return created
    })

    return c.json(await getTopicDetail(topic.id), 201)
  })

  app.get('/api/discussion/topics/:id', async (c) => {
    const denied = await denyGuestAccess(c, 'Sign in to view discussions')
    if (denied) return denied

    const topic = await getTopicDetail(numericId.parse(c.req.param('id')))
    if (!topic) return notFound(c)
    return c.json(topic)
  })

  app.post('/api/discussion/topics/:id/posts', authMiddleware, async (c: Context) => {
    const user = await requireAuthUser(c)
    const rateLimited = await checkRateLimit(
      c,
      'discussion:reply',
      `user:${user.id}`,
      120,
      60 * 60 * 1000
    )
    if (rateLimited) return rateLimited

    const topicId = numericId.parse(c.req.param('id'))
    const body = createPostSchema.parse(await c.req.json())

    const [topic] = await db
      .select()
      .from(schema.topics)
      .where(eq(schema.topics.id, topicId))
      .limit(1)
    if (!topic) return notFound(c)

    const [reply] = await db
      .insert(schema.posts)
      .values({
        topicId,
        userId: user.id,
        content: body.content
      })
      .returning()

    await db
      .update(schema.topics)
      .set({ updatedAt: new Date() })
      .where(eq(schema.topics.id, topicId))

    return c.json(
      {
        id: reply.id,
        topicId: reply.topicId,
        user: {
          id: user.id,
          name: user.name,
          avatarUrl: user.avatarUrl
        },
        content: reply.content,
        createdAt: reply.createdAt
      },
      201
    )
  })

  app.patch('/api/admin/discussion/topics/:id', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const topicId = numericId.parse(c.req.param('id'))
    const body = updateTopicAdminSchema.parse(await c.req.json())
    const [topic] = await db
      .update(schema.topics)
      .set({ ...body, updatedAt: new Date() })
      .where(eq(schema.topics.id, topicId))
      .returning()
    if (!topic) return notFound(c)
    return c.json(await getTopicDetail(topic.id))
  })

  app.delete('/api/admin/discussion/topics/:id', authMiddleware, async (c) => {
    const denied = await requireGroup(c, 'admin')
    if (denied) return denied

    const topicId = numericId.parse(c.req.param('id'))
    const [topic] = await db
      .update(schema.topics)
      .set({ deletedAt: new Date(), updatedAt: new Date() })
      .where(eq(schema.topics.id, topicId))
      .returning()
    if (!topic) return notFound(c)
    return c.json({ ok: true })
  })
}

const createTopicSchema = z.object({
  title: z.string().min(1).max(100),
  content: z.string().min(1).max(20_000),
  tags: z.array(z.string().min(1).max(32)).max(5).default([])
})

const discussionListQuerySchema = listQuerySchema.extend({
  tags: z.string().optional()
})

function parseTags(raw: string | undefined) {
  return (raw ?? '')
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
    .slice(0, 5)
}

const createPostSchema = z.object({
  content: z.string().min(1).max(20_000)
})

const updateTopicAdminSchema = z.object({
  pinned: z.boolean().optional()
})
