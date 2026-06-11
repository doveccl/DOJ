import { drizzle } from 'drizzle-orm/postgres-js'
import postgres from 'postgres'
import * as schema from './schema'

const url = process.env.DATABASE ?? 'postgres://postgres@localhost:5432/postgres'

export const sqlClient = postgres(url)

export const db = drizzle(sqlClient, { schema })
export const closeDb = () => sqlClient.end()
export { schema }
