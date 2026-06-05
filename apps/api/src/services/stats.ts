import { eq, sql } from 'drizzle-orm'
import { db, schema } from '@doj/db/client'

export async function countRows(table: any, where?: any) {
  const query = db.select({ total: sql<number>`count(*)::int` }).from(table)
  const [row] = await (where ? query.where(where) : query)
  return row?.total ?? 0
}

export async function countVisibleSubmissions() {
  const [row] = await db
    .select({ total: sql<number>`count(*)::int` })
    .from(schema.submissions)
    .innerJoin(schema.problems, eq(schema.submissions.problemId, schema.problems.id))
    .where(eq(schema.problems.visible, true))
  return row?.total ?? 0
}
