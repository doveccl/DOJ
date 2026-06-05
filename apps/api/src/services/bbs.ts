import { asc, desc, eq } from 'drizzle-orm'
import { db, schema } from '@doj/db/client'

export async function getTopicDetail(id: number) {
  const [topic] = await db
    .select({
      id: schema.bbsTopics.id,
      title: schema.bbsTopics.title,
      tags: schema.bbsTopics.tags,
      linkedProblemId: schema.bbsTopics.linkedProblemId,
      linkedContestId: schema.bbsTopics.linkedContestId,
      createdAt: schema.bbsTopics.createdAt,
      updatedAt: schema.bbsTopics.updatedAt,
      problemVisible: schema.problems.visible,
      userId: schema.users.id,
      userName: schema.users.name
    })
    .from(schema.bbsTopics)
    .innerJoin(schema.users, eq(schema.bbsTopics.userId, schema.users.id))
    .leftJoin(schema.problems, eq(schema.bbsTopics.linkedProblemId, schema.problems.id))
    .where(eq(schema.bbsTopics.id, id))
    .limit(1)

  if (!topic) return null
  const { problemVisible: _problemVisible, ...topicPayload } = topic

  const replies = await db
    .select({
      id: schema.bbsReplies.id,
      contentMarkdown: schema.bbsReplies.contentMarkdown,
      createdAt: schema.bbsReplies.createdAt,
      userId: schema.users.id,
      userName: schema.users.name
    })
    .from(schema.bbsReplies)
    .innerJoin(schema.users, eq(schema.bbsReplies.userId, schema.users.id))
    .where(eq(schema.bbsReplies.topicId, id))
    .orderBy(asc(schema.bbsReplies.createdAt))

  return {
    topic: {
      ...topicPayload,
      linkedProblemId: topic.linkedProblemId && topic.problemVisible ? topic.linkedProblemId : null
    },
    replies
  }
}

export async function getRecentBbsTopics(limit: number) {
  const rows = await db
    .select({
      id: schema.bbsTopics.id,
      title: schema.bbsTopics.title,
      tags: schema.bbsTopics.tags,
      linkedProblemId: schema.bbsTopics.linkedProblemId,
      linkedContestId: schema.bbsTopics.linkedContestId,
      createdAt: schema.bbsTopics.createdAt,
      updatedAt: schema.bbsTopics.updatedAt,
      problemVisible: schema.problems.visible,
      userId: schema.users.id,
      userName: schema.users.name
    })
    .from(schema.bbsTopics)
    .innerJoin(schema.users, eq(schema.bbsTopics.userId, schema.users.id))
    .leftJoin(schema.problems, eq(schema.bbsTopics.linkedProblemId, schema.problems.id))
    .orderBy(desc(schema.bbsTopics.updatedAt))
    .limit(limit)

  return rows.map(({ problemVisible, ...row }) => ({
    ...row,
    linkedProblemId: row.linkedProblemId && problemVisible ? row.linkedProblemId : null
  }))
}
