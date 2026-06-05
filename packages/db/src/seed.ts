import { closeDb, db, schema } from './client'
import { eq, sql } from 'drizzle-orm'
import { defaultLanguageConfigs } from '@doj/shared/languages'
import { runtimeSettingsDefaults } from '@doj/shared/settings'

const builtinGroups = [
  { key: 'admin', name: 'Admin', description: 'System administrators', builtin: true },
  { key: 'user', name: 'User', description: 'Registered users', builtin: true },
  { key: 'guest', name: 'Guest', description: 'Anonymous or low-trust users', builtin: true }
]

await db
  .insert(schema.groups)
  .values(builtinGroups)
  .onConflictDoNothing({ target: schema.groups.key })

await db
  .insert(schema.systemSettings)
  .values(
    Object.entries(runtimeSettingsDefaults).map(([key, value]) => ({
      key,
      value
    }))
  )
  .onConflictDoUpdate({
    target: schema.systemSettings.key,
    set: {
      value: sql`excluded.value`,
      updatedAt: new Date()
    }
  })

for (const language of defaultLanguageConfigs) {
  await db
    .insert(schema.judgeLanguages)
    .values({
      id: language.id,
      name: language.name,
      enabled: language.enabled,
      sourceFile: language.sourceFile,
      dockerfile: language.dockerfile(language.sourceFile),
      command: language.command,
      sortOrder: language.sortOrder
    })
    .onConflictDoUpdate({
      target: schema.judgeLanguages.id,
      set: {
        name: language.name,
        sourceFile: language.sourceFile,
        dockerfile: language.dockerfile(language.sourceFile),
        command: language.command,
        sortOrder: language.sortOrder,
        updatedAt: new Date()
      }
    })
}

await db
  .insert(schema.judgeRunners)
  .values({
    key: 'local-docker',
    name: 'Local Docker',
    enabled: true,
    kind: 'docker',
    endpoint: process.env.DOCKER_HOST ?? null,
    authHeader: null,
    concurrency: Number(process.env.DOJ_JUDGE_CONCURRENCY ?? 2),
    sortOrder: 10
  })
  .onConflictDoUpdate({
    target: schema.judgeRunners.key,
    set: {
      name: 'Local Docker',
      enabled: true,
      kind: 'docker',
      endpoint: process.env.DOCKER_HOST ?? null,
      authHeader: null,
      concurrency: Number(process.env.DOJ_JUDGE_CONCURRENCY ?? 2),
      sortOrder: 10,
      updatedAt: new Date()
    }
  })

const adminName = process.env.DOJ_ADMIN_NAME ?? 'admin'
const adminEmail = process.env.DOJ_ADMIN_EMAIL ?? 'admin@example.test'
const adminPassword = process.env.DOJ_ADMIN_PASSWORD ?? 'admin12345'

const [adminGroup] = await db
  .select()
  .from(schema.groups)
  .where(eq(schema.groups.key, 'admin'))
  .limit(1)
if (!adminGroup) throw new Error('admin group missing after seed')

const [existingAdmin] = await db
  .select()
  .from(schema.users)
  .where(eq(schema.users.name, adminName))
  .limit(1)
if (!existingAdmin) {
  const [admin] = await db
    .insert(schema.users)
    .values({
      name: adminName,
      email: adminEmail,
      passwordHash: await Bun.password.hash(adminPassword, {
        algorithm: 'argon2id',
        memoryCost: 19456,
        timeCost: 2
      })
    })
    .returning()

  await db
    .insert(schema.userGroups)
    .values({ userId: admin.id, groupId: adminGroup.id, manager: true })
}

const starterProblems = [
  {
    title: 'A+B Problem',
    tags: ['beginner'],
    statementMarkdown:
      '# A+B Problem\n\nRead two integers `a` and `b`, then print their sum.\n\nInput: two integers separated by whitespace.\n\nOutput: one integer.',
    timeLimitMs: 1000,
    memoryLimitBytes: 128 * 1024 * 1024,
    testCases: [
      {
        name: 'sample',
        input: '1 2\n',
        output: '3\n'
      },
      {
        name: 'negative',
        input: '-5 8\n',
        output: '3\n',
        hidden: true
      }
    ]
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
      .update(schema.problemVersions)
      .set({
        statementMarkdown: starter.statementMarkdown,
        timeLimitMs: starter.timeLimitMs,
        memoryLimitBytes: starter.memoryLimitBytes,
        outputLimitBytes: runtimeSettingsDefaults.outputLimitBytes,
        testCases: starter.testCases
      })
      .where(eq(schema.problemVersions.problemId, existing.id))
    continue
  }

  const [problem] = await db
    .insert(schema.problems)
    .values({
      title: starter.title,
      tags: starter.tags
    })
    .returning()

  await db.insert(schema.problemVersions).values({
    problemId: problem.id,
    version: 1,
    statementMarkdown: starter.statementMarkdown,
    timeLimitMs: starter.timeLimitMs,
    memoryLimitBytes: starter.memoryLimitBytes,
    outputLimitBytes: runtimeSettingsDefaults.outputLimitBytes,
    testCases: starter.testCases
  })
}

console.log('Seeded builtin groups, judge language, admin user, and P1000 A+B Problem.')

await closeDb()
