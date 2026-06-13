import { createHash } from 'node:crypto'
import { and, asc, desc, eq, inArray, isNull, sql } from 'drizzle-orm'
import { db, schema } from '@doj/db/client'

interface UserBrief {
  id: number
  name: string
  avatarUrl: string
}

export async function getTopicDetail(id: number) {
  const [topic] = await db
    .select({
      id: schema.topics.id,
      title: schema.topics.title,
      tags: schema.topics.tags,
      pinned: schema.topics.pinned,
      deletedAt: schema.topics.deletedAt,
      createdAt: schema.topics.createdAt,
      updatedAt: schema.topics.updatedAt
    })
    .from(schema.topics)
    .where(eq(schema.topics.id, id))
    .limit(1)

  if (!topic || topic.deletedAt) return null

  const posts = await db
    .select({
      id: schema.posts.id,
      topicId: schema.posts.topicId,
      content: schema.posts.content,
      createdAt: schema.posts.createdAt,
      userId: schema.users.id,
      authorName: schema.users.name,
      authorEmail: schema.users.email
    })
    .from(schema.posts)
    .innerJoin(schema.users, eq(schema.posts.userId, schema.users.id))
    .where(sql`${schema.posts.topicId} = ${id} and ${schema.posts.deletedAt} is null`)
    .orderBy(asc(schema.posts.createdAt), asc(schema.posts.id))
  const postViews = posts.map((post) => ({
    id: post.id,
    topicId: post.topicId,
    user: formatUser(post.userId, post.authorName, post.authorEmail),
    content: post.content,
    createdAt: post.createdAt
  }))

  return {
    id: topic.id,
    title: topic.title,
    tags: topic.tags,
    pinned: topic.pinned,
    author: postViews[0]?.user ?? emptyUser(),
    createdAt: topic.createdAt,
    updatedAt: topic.updatedAt,
    posts: postViews
  }
}

export async function getRecentTopics(limit: number, offset = 0, tags: string[] = []) {
  const rows = await db
    .select({
      id: schema.topics.id,
      title: schema.topics.title,
      tags: schema.topics.tags,
      pinned: schema.topics.pinned,
      createdAt: schema.topics.createdAt,
      updatedAt: schema.topics.updatedAt
    })
    .from(schema.topics)
    .where(topicListFilter(tags))
    .orderBy(desc(schema.topics.pinned), desc(schema.topics.updatedAt), desc(schema.topics.id))
    .limit(limit)
    .offset(offset)

  const authorByTopic = await getTopicAuthors(rows.map((row) => row.id))
  return rows.map((row) => ({
    ...row,
    author: authorByTopic.get(row.id) ?? emptyUser()
  }))
}

export async function countTopics(tags: string[] = []) {
  const [row] = await db
    .select({ total: sql<number>`count(*)::int` })
    .from(schema.topics)
    .where(topicListFilter(tags))
  return row?.total ?? 0
}

function topicListFilter(tags: string[]) {
  return and(
    isNull(schema.topics.deletedAt),
    ...tags.map((tag) => sql`${schema.topics.tags} @> ARRAY[${tag}]::text[]`)
  )
}

async function getTopicAuthors(topicIds: number[]) {
  if (!topicIds.length) return new Map<number, UserBrief>()

  const posts = await db
    .select({
      topicId: schema.posts.topicId,
      userId: schema.users.id,
      authorName: schema.users.name,
      authorEmail: schema.users.email
    })
    .from(schema.posts)
    .innerJoin(schema.users, eq(schema.posts.userId, schema.users.id))
    .where(and(inArray(schema.posts.topicId, topicIds), isNull(schema.posts.deletedAt)))
    .orderBy(asc(schema.posts.topicId), asc(schema.posts.createdAt), asc(schema.posts.id))

  const result = new Map<number, UserBrief>()
  for (const post of posts) {
    if (!result.has(post.topicId)) {
      result.set(post.topicId, formatUser(post.userId, post.authorName, post.authorEmail))
    }
  }
  return result
}

function formatUser(id: number, name: string, email: string) {
  return {
    id,
    name,
    avatarUrl: gravatarUrl(email)
  }
}

function emptyUser() {
  return {
    id: 0,
    name: '-',
    avatarUrl: gravatarUrl('')
  }
}

function gravatarUrl(email: string) {
  const hash = createHash('md5').update(email.trim().toLowerCase()).digest('hex')
  return `https://www.gravatar.com/avatar/${hash}?d=identicon&s=80`
}
