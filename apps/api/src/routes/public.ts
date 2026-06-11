import { Hono } from 'hono'
import { and, asc, isNull, sql } from 'drizzle-orm'
import { db, schema } from '@doj/db/client'
import { denyGuestAccess, getOptionalAuthUser } from '../auth'
import { apiError } from '../errors'
import {
  getHeatmap,
  getRanking,
  getRecommendedProblems
} from '../services/stats'
import { listQuerySchema } from '../validation'

export function registerPublicRoutes(app: Hono) {
  app.get('/api/languages', async (c) => {
    const list = await db
      .select({
        id: schema.languages.id,
        name: schema.languages.name,
        source: schema.languages.source,
        sort: schema.languages.sort
      })
      .from(schema.languages)
      .orderBy(asc(schema.languages.sort), asc(schema.languages.id))

    return c.json(list)
  })

  app.get('/api/ranking', async (c) => {
    const denied = await denyGuestAccess(c, 'Sign in to view ranking')
    if (denied) return denied

    const query = listQuerySchema.parse(c.req.query())
    return c.json(await getRanking(query.page, query.pageSize))
  })

  app.get('/api/tags', async (c) => {
    const denied = await denyGuestAccess(c, 'Sign in to view problem tags')
    if (denied) return denied

    const rows = await db
      .select({
        name: sql<string>`unnest(${schema.problems.tags})`,
        count: sql<number>`count(*)::int`
      })
      .from(schema.problems)
      .where(sql`${schema.problems.visible} = true and ${schema.problems.deletedAt} is null`)
      .groupBy(sql`1`)
    const items = rows
      .filter((row) => row.name)
      .sort((left, right) => right.count - left.count || left.name.localeCompare(right.name))
    return c.json(items)
  })

  app.get('/api/home/heatmap', async (c) => {
    const user = await getOptionalAuthUser(c)
    if (!user) return apiError(c, 401, 'UNAUTHORIZED', 'Sign in to view heatmap')
    return c.json(await getHeatmap(user.id, c.req.query('tz') ?? 'UTC'))
  })

  app.get('/api/home/recommended-problems', async (c) => {
    const denied = await denyGuestAccess(c, 'Sign in to view recommended problems')
    if (denied) return denied

    const user = await getOptionalAuthUser(c)
    return c.json(await getRecommendedProblems(user?.id))
  })

  app.get('/api/home/contests', async (c) => {
    const denied = await denyGuestAccess(c, 'Sign in to view contests')
    if (denied) return denied

    const items = await db
      .select({
        id: schema.contests.id,
        title: schema.contests.title,
        type: schema.contests.type,
        startAt: schema.contests.startAt,
        endAt: schema.contests.endAt
      })
      .from(schema.contests)
      .where(and(isNull(schema.contests.deletedAt), sql`${schema.contests.endAt} > now()`))
      .orderBy(asc(schema.contests.startAt), asc(schema.contests.id))
      .limit(10)

    return c.json(items)
  })
}
