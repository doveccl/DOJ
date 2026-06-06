import { Hono } from 'hono'
import { and, eq } from 'drizzle-orm'
import { z } from 'zod'
import { db, schema } from '@doj/db/client'
import { authMiddleware, requireAuthUser } from '../auth'
import { checkRateLimit } from '../rate-limit'
import { getRecentTopics, getTopicDetail } from '../services/discussion'
import { numericId } from '../validation'

export function registerDiscussionRoutes(app: Hono) {
  app.get('/api/discussion/topics', async (c) => {
    const list = await getRecentTopics(50)

    return c.json({ total: list.length, list })
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

    if (body.linkedProblemId) {
      const [problem] = await db
        .select({ id: schema.problems.id })
        .from(schema.problems)
        .where(and(eq(schema.problems.id, body.linkedProblemId), eq(schema.problems.visible, true)))
        .limit(1)
      if (!problem) {
        return c.json({ code: 'PROBLEM_NOT_FOUND', message: 'Linked problem is not visible' }, 404)
      }
    }

    if (body.linkedContestId) {
      const [contest] = await db
        .select({ id: schema.contests.id })
        .from(schema.contests)
        .where(eq(schema.contests.id, body.linkedContestId))
        .limit(1)
      if (!contest) {
        return c.json({ code: 'CONTEST_NOT_FOUND', message: 'Linked contest does not exist' }, 404)
      }
    }

    const topic = await db.transaction(async (tx) => {
      const [created] = await tx
        .insert(schema.discussionTopics)
        .values({
          userId: user.id,
          title: body.title,
          tags: body.tags,
          linkedProblemId: body.linkedProblemId ?? null,
          linkedContestId: body.linkedContestId ?? null
        })
        .returning()

      await tx.insert(schema.discussionReplies).values({
        topicId: created.id,
        userId: user.id,
        contentMarkdown: body.contentMarkdown
      })

      return created
    })

    return c.json(await getTopicDetail(topic.id), 201)
  })

  app.get('/api/discussion/topics/:id', async (c) => {
    const topic = await getTopicDetail(numericId.parse(c.req.param('id')))
    if (!topic) return c.notFound()
    return c.json(topic)
  })

  app.post('/api/discussion/topics/:id/replies', authMiddleware, async (c) => {
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
    const body = createReplySchema.parse(await c.req.json())

    const [topic] = await db
      .select()
      .from(schema.discussionTopics)
      .where(eq(schema.discussionTopics.id, topicId))
      .limit(1)
    if (!topic) return c.notFound()

    const [reply] = await db
      .insert(schema.discussionReplies)
      .values({
        topicId,
        userId: user.id,
        contentMarkdown: body.contentMarkdown
      })
      .returning()

    await db
      .update(schema.discussionTopics)
      .set({ updatedAt: new Date() })
      .where(eq(schema.discussionTopics.id, topicId))

    return c.json(reply, 201)
  })
}

const createTopicSchema = z.object({
  title: z.string().min(1).max(160),
  contentMarkdown: z.string().min(1).max(20_000),
  tags: z.array(z.string().min(1).max(40)).max(12).default([]),
  linkedProblemId: numericId.optional(),
  linkedContestId: numericId.optional()
})

const createReplySchema = z.object({
  contentMarkdown: z.string().min(1).max(20_000)
})
