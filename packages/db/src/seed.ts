import { closeDb, db, schema } from './client'
import { eq, sql } from 'drizzle-orm'
import { defaultLanguageConfigs } from '@doj/shared/languages'
import { runtimeSettingsDefaults } from '@doj/shared/settings'
import { putObject } from '@doj/shared/storage'

await db
  .insert(schema.settings)
  .values(
    Object.entries(runtimeSettingsDefaults).map(([key, value]) => ({
      key,
      value
    }))
  )
  .onConflictDoUpdate({
    target: schema.settings.key,
    set: {
      value: sql`excluded.value`,
      updatedAt: new Date()
    }
  })

for (const language of defaultLanguageConfigs) {
  await db
    .insert(schema.languages)
    .values({
      id: language.id,
      name: language.name,
      source: language.source,
      dockerfile: language.dockerfile,
      sort: language.sort
    })
    .onConflictDoUpdate({
      target: schema.languages.id,
      set: {
        name: language.name,
        source: language.source,
        dockerfile: language.dockerfile,
        sort: language.sort,
        updatedAt: new Date()
      }
    })
}

const adminName = process.env.ADMIN_NAME ?? 'admin'
const adminEmail = process.env.ADMIN_EMAIL ?? 'admin@example.test'
const adminPassword = process.env.ADMIN_PASSWORD ?? 'admin12345'

const [existingAdmin] = await db
  .select()
  .from(schema.users)
  .where(eq(schema.users.name, adminName))
  .limit(1)
if (!existingAdmin) {
  await db
    .insert(schema.users)
    .values({
      name: adminName,
      email: adminEmail,
      admin: true,
      mustChangePassword: true,
      passwordHash: await Bun.password.hash(adminPassword, {
        algorithm: 'argon2id',
        memoryCost: 19456,
        timeCost: 2
      })
    })
    .returning()
} else if (!existingAdmin.admin || !existingAdmin.mustChangePassword) {
  await db
    .update(schema.users)
    .set({ admin: true, mustChangePassword: true, updatedAt: new Date() })
    .where(eq(schema.users.id, existingAdmin.id))
}

const starterProblems = [
  {
    title: 'A+B Problem',
    tags: ['beginner'],
    statement:
      '# A+B Problem\n\nRead two integers a and b, then print their sum.\n\nInput: two integers separated by whitespace.\n\nOutput: one integer.',
    mode: 'default',
    timeLimit: 1000,
    memoryLimit: 128 * 1024 * 1024
  }
]

for (const starter of starterProblems) {
  const [existing] = await db
    .select()
    .from(schema.problems)
    .where(eq(schema.problems.id, 1000))
    .limit(1)
  if (existing) {
    await db
      .update(schema.problems)
      .set({
        title: starter.title,
        tags: starter.tags,
        timeLimit: starter.timeLimit,
        memoryLimit: starter.memoryLimit,
        mode: starter.mode,
        visible: true,
        updatedAt: new Date()
      })
      .where(eq(schema.problems.id, existing.id))
    continue
  }

  await db.insert(schema.problems).values({
    id: 1000,
    title: starter.title,
    tags: starter.tags,
    timeLimit: starter.timeLimit,
    memoryLimit: starter.memoryLimit,
    mode: starter.mode,
    visible: true
  })
}

await db.execute(sql`
  select setval(pg_get_serial_sequence('problems', 'id'), (select max(id) from problems))
`)

await Promise.all([
  putObject({
    key: 'problems/1000/statement.md',
    body: starterProblems[0].statement,
    contentType: 'text/markdown; charset=utf-8'
  }),
  putObject({ key: 'problems/1000/data/in1.txt', body: '1 2\n', contentType: 'text/plain' }),
  putObject({ key: 'problems/1000/data/ans1.txt', body: '3\n', contentType: 'text/plain' }),
  putObject({ key: 'problems/1000/data/in2.txt', body: '-5 8\n', contentType: 'text/plain' }),
  putObject({ key: 'problems/1000/data/ans2.txt', body: '3\n', contentType: 'text/plain' })
])

console.log('Seeded settings, cpp language, admin user, P1000 metadata, and P1000 S3 assets.')

await closeDb()
