import { drizzle } from 'drizzle-orm/postgres-js'
import postgres from 'postgres'
import * as schema from './schema'

const url = process.env.DATABASE_URL ?? 'postgres://postgres@localhost:5432/postgres'

const client = postgres(url)

export const db = drizzle(client, { schema })
export const closeDb = () => client.end()
export { schema }
