import { and, eq } from 'drizzle-orm'
import { db, schema } from '@doj/db/client'

export interface LanguageRuntime {
  id: string
  name: string
  sourceFile: string
  dockerfile: string
  command: string[]
}

export async function getLanguage(id: string): Promise<LanguageRuntime> {
  const [language] = await db
    .select({
      id: schema.judgeLanguages.id,
      name: schema.judgeLanguages.name,
      sourceFile: schema.judgeLanguages.sourceFile,
      dockerfile: schema.judgeLanguages.dockerfile,
      command: schema.judgeLanguages.command
    })
    .from(schema.judgeLanguages)
    .where(and(eq(schema.judgeLanguages.id, id), eq(schema.judgeLanguages.enabled, true)))
    .limit(1)

  if (!language) throw new Error(`unsupported or disabled language: ${id}`)
  return language
}
