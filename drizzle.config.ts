import { defineConfig } from 'drizzle-kit'

const databaseUrl = process.env.DATABASE ?? 'postgres://postgres@localhost:5432/postgres'

export default defineConfig({
  dialect: 'postgresql',
  schema: './packages/db/src/schema.ts',
  out: './drizzle',
  dbCredentials: {
    url: databaseUrl
  }
})
