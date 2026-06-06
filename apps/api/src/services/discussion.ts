import { asc, desc, eq } from 'drizzle-orm'
import { db, schema } from '@doj/db/client'

export async function getTopicDetail(id: number) {
  const [topic] = await db
    .select({
      id: schema.discussionTopics.id,
      title: schema.discussionTopics.title,
      tags: schema.discussionTopics.tags,
      linkedProblemId: schema.discussionTopics.linkedProblemId,
      linkedContestId: schema.discussionTopics.linkedContestId,
      createdAt: schema.discussionTopics.createdAt,
      updatedAt: schema.discussionTopics.updatedAt,
      problemVisible: schema.problems.visible,
      userId: schema.users.id,
      userName: schema.users.name
    })
    .from(schema.discussionTopics)
    .innerJoin(schema.users, eq(schema.discussionTopics.userId, schema.users.id))
    .leftJoin(schema.problems, eq(schema.discussionTopics.linkedProblemId, schema.problems.id))
    .where(eq(schema.discussionTopics.id, id))
    .limit(1)

  if (!topic) return null
  const { problemVisible: _problemVisible, ...topicPayload } = topic

  const replies = await db
    .select({
      id: schema.discussionReplies.id,
      contentMarkdown: schema.discussionReplies.contentMarkdown,
      createdAt: schema.discussionReplies.createdAt,
      userId: schema.users.id,
      userName: schema.users.name
    })
    .from(schema.discussionReplies)
    .innerJoin(schema.users, eq(schema.discussionReplies.userId, schema.users.id))
    .where(eq(schema.discussionReplies.topicId, id))
    .orderBy(asc(schema.discussionReplies.createdAt))

  return {
    topic: {
      ...topicPayload,
      linkedProblemId: topic.linkedProblemId && topic.problemVisible ? topic.linkedProblemId : null
    },
    replies
  }
}

export async function getRecentTopics(limit: number) {
  const rows = await db
    .select({
      id: schema.discussionTopics.id,
      title: schema.discussionTopics.title,
      tags: schema.discussionTopics.tags,
      linkedProblemId: schema.discussionTopics.linkedProblemId,
      linkedContestId: schema.discussionTopics.linkedContestId,
      createdAt: schema.discussionTopics.createdAt,
      updatedAt: schema.discussionTopics.updatedAt,
      problemVisible: schema.problems.visible,
      userId: schema.users.id,
      userName: schema.users.name
    })
    .from(schema.discussionTopics)
    .innerJoin(schema.users, eq(schema.discussionTopics.userId, schema.users.id))
    .leftJoin(schema.problems, eq(schema.discussionTopics.linkedProblemId, schema.problems.id))
    .orderBy(desc(schema.discussionTopics.updatedAt))
    .limit(limit)

  return rows.map(({ problemVisible, ...row }) => ({
    ...row,
    linkedProblemId: row.linkedProblemId && problemVisible ? row.linkedProblemId : null
  }))
}
