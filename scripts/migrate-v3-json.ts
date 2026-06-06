import { and, eq } from 'drizzle-orm'
import { closeDb, db, schema } from '../packages/db/src/client'

const inputDir = process.env.MIGRATE_V3_EXPORT_DIR
const defaultPassword = process.env.MIGRATE_V3_DEFAULT_PASSWORD
const languageMap = JSON.parse(process.env.MIGRATE_V3_LANGUAGE_MAP ?? '{"2":"py"}') as Record<
  string,
  string
>

if (!inputDir) {
  console.error('Set MIGRATE_V3_EXPORT_DIR to a directory containing v3 mongoexport JSON files.')
  console.error(
    'Expected files: users.json, problems.json, contests.json, submissions.json, posts.json'
  )
  process.exit(1)
}

const maps = {
  users: new Map<string, number>(),
  problems: new Map<string, number>(),
  contests: new Map<string, number>(),
  topics: new Map<string, number>()
}

try {
  await migrateUsers()
  await migrateContests()
  await migrateProblems()
  await migrateSubmissions()
  await migratePosts()
  console.log('v3 JSON migration completed.')
} finally {
  await closeDb()
}

async function migrateUsers() {
  const users = await readCollection('users')
  const [userGroup, adminGroup] = await Promise.all([getGroup('user'), getGroup('admin')])
  if (!userGroup || !adminGroup)
    throw new Error('Run db:seed before migration; builtin groups missing.')

  for (const item of users) {
    const legacyId = legacy(item)
    const [existing] = await db
      .select()
      .from(schema.users)
      .where(eq(schema.users.legacyId, legacyId))
      .limit(1)
    if (existing) {
      maps.users.set(legacyId, existing.id)
      continue
    }

    const [created] = await db
      .insert(schema.users)
      .values({
        legacyId,
        name: stringValue(item.name, `user_${legacyId.slice(-8)}`).slice(0, 32),
        email: stringValue(item.mail, `${legacyId}@legacy.invalid`),
        passwordHash: defaultPassword
          ? await Bun.password.hash(defaultPassword, { algorithm: 'argon2id' })
          : stringValue(
              item.password,
              await Bun.password.hash('change-me', { algorithm: 'argon2id' })
            ),
        introduction: stringValue(item.introduction, '').slice(0, 500),
        solvedCount: numberValue(item.solve),
        submissionCount: numberValue(item.submit),
        createdAt: dateValue(item.createdAt),
        updatedAt: dateValue(item.updatedAt)
      })
      .returning()

    await db.insert(schema.userGroups).values({
      userId: created.id,
      groupId: numberValue(item.group) >= 1 ? adminGroup.id : userGroup.id,
      manager: numberValue(item.group) >= 1
    })

    maps.users.set(legacyId, created.id)
  }

  console.log(`migrated users: ${maps.users.size}`)
}

async function migrateContests() {
  const contests = await readCollection('contests')
  for (const item of contests) {
    const legacyId = legacy(item)
    const [existing] = await db
      .select()
      .from(schema.contests)
      .where(eq(schema.contests.legacyId, legacyId))
      .limit(1)
    if (existing) {
      maps.contests.set(legacyId, existing.id)
      continue
    }

    const [created] = await db
      .insert(schema.contests)
      .values({
        legacyId,
        title: stringValue(item.title, `Legacy Contest ${legacyId.slice(-8)}`).slice(0, 160),
        description: stringValue(item.description, ''),
        type: numberValue(item.type) === 1 ? 'ICPC' : 'OI',
        startAt: dateValue(item.startAt),
        endAt: dateValue(item.endAt),
        freezeAt: dateValue(item.freezeAt, null)
      })
      .returning()

    maps.contests.set(legacyId, created.id)
  }

  console.log(`migrated contests: ${maps.contests.size}`)
}

async function migrateProblems() {
  const problems = await readCollection('problems')
  for (const item of problems) {
    const legacyId = legacy(item)
    const [existing] = await db
      .select()
      .from(schema.problems)
      .where(eq(schema.problems.legacyId, legacyId))
      .limit(1)
    if (existing) {
      maps.problems.set(legacyId, existing.id)
      continue
    }

    const result = await db.transaction(async (tx) => {
      const [problem] = await tx
        .insert(schema.problems)
        .values({
          legacyId,
          title: stringValue(item.title, `Legacy Problem ${legacyId.slice(-8)}`).slice(0, 160),
          tags: Array.isArray(item.tags) ? item.tags.map(String) : [],
          statementMarkdown: stringValue(item.content, 'Legacy statement missing.'),
          timeLimitMs: numberValue(item.timeLimit, 1000),
          memoryLimitBytes: numberValue(item.memoryLimit, 256 * 1024 * 1024),
          solvedCount: numberValue(item.solve),
          submissionCount: numberValue(item.submit),
          createdAt: dateValue(item.createdAt),
          updatedAt: dateValue(item.updatedAt)
        })
        .returning()

      return { problem }
    })

    maps.problems.set(legacyId, result.problem.id)

    const contestLegacyId = legacy(item.contest?.id, '')
    const contestId = contestLegacyId ? maps.contests.get(contestLegacyId) : undefined
    if (contestId) {
      await db
        .insert(schema.contestProblems)
        .values({
          contestId,
          problemId: result.problem.id,
          key: stringValue(item.contest?.key, String(result.problem.id)),
          score: 100
        })
        .onConflictDoNothing()
    }
  }

  console.log(`migrated problems: ${maps.problems.size}`)
}

async function migrateSubmissions() {
  const submissions = await readCollection('submissions')
  let imported = 0
  for (const item of submissions) {
    const userId = maps.users.get(legacy(item.uid, ''))
    const problemLegacyId = legacy(item.pid, '')
    const problemId = maps.problems.get(problemLegacyId)
    const languageId = languageMap[String(item.language)]
    if (!userId || !problemId || !languageId) continue

    const legacyId = legacy(item)
    const [existing] = await db
      .select()
      .from(schema.submissions)
      .where(eq(schema.submissions.legacyId, legacyId))
      .limit(1)
    if (existing) continue

    const contestId = maps.contests.get(legacy(item.cid, '')) ?? null
    const [created] = await db
      .insert(schema.submissions)
      .values({
        legacyId,
        userId,
        problemId,
        contestId,
        languageId,
        sourceCode: stringValue(item.code, ''),
        open: Boolean(item.open),
        status: statusValue(item.result?.status),
        timeMs: numberValue(item.result?.time),
        memoryBytes: numberValue(item.result?.memory),
        message: stringValue(item.result?.extra, ''),
        createdAt: dateValue(item.createdAt),
        updatedAt: dateValue(item.updatedAt)
      })
      .returning()

    const cases = Array.isArray(item.cases) ? item.cases : []
    if (cases.length) {
      await db.insert(schema.submissionCases).values(
        cases.map((testCase, index) => ({
          submissionId: created.id,
          caseIndex: index + 1,
          status: statusValue(testCase.status),
          timeMs: numberValue(testCase.time),
          memoryBytes: numberValue(testCase.memory),
          message: stringValue(testCase.extra, '')
        }))
      )
    }

    imported += 1
  }

  console.log(`migrated submissions: ${imported}`)
}

async function migratePosts() {
  const posts = await readCollection('posts')
  let imported = 0
  for (const item of posts) {
    const userId = maps.users.get(legacy(item.uid, ''))
    if (!userId) continue

    const topicKey = legacy(item.topic, stringValue(item.topic, 'Legacy Topic'))
    let topicId = maps.topics.get(topicKey)
    if (!topicId) {
      const [topic] = await db
        .insert(schema.discussionTopics)
        .values({
          userId,
          title:
            topicKey.length === 24 ? `Legacy topic ${topicKey.slice(-8)}` : topicKey.slice(0, 160),
          tags: ['legacy'],
          createdAt: dateValue(item.createdAt),
          updatedAt: dateValue(item.updatedAt)
        })
        .returning()
      topicId = topic.id
      maps.topics.set(topicKey, topic.id)
      maps.topics.set(legacy(item), topic.id)
    }

    const [existing] = await db
      .select()
      .from(schema.bbsReplies)
      .where(
        and(
          eq(schema.bbsReplies.topicId, topicId),
          eq(schema.bbsReplies.contentMarkdown, stringValue(item.content, ''))
        )
      )
      .limit(1)
    if (existing) continue

    await db.insert(schema.discussionReplies).values({
      topicId,
      userId,
      contentMarkdown: stringValue(item.content, ''),
      createdAt: dateValue(item.createdAt),
      updatedAt: dateValue(item.updatedAt)
    })
    imported += 1
  }

  console.log(`migrated discussion replies: ${imported}`)
}

async function readCollection(name: string) {
  const file = Bun.file(`${inputDir}/${name}.json`)
  if (!(await file.exists())) return []
  const text = await file.text()
  const trimmed = text.trim()
  if (!trimmed) return []
  if (trimmed.startsWith('[')) return JSON.parse(trimmed) as Array<Record<string, unknown>>
  return trimmed.split('\n').map((line) => JSON.parse(line)) as Array<Record<string, unknown>>
}

async function getGroup(key: string) {
  const [group] = await db.select().from(schema.groups).where(eq(schema.groups.key, key)).limit(1)
  return group
}

function legacy(value: unknown, fallback?: string) {
  if (!value) {
    if (fallback !== undefined) return fallback
    throw new Error('missing legacy id')
  }
  if (typeof value === 'string') return value
  if (typeof value === 'object') {
    const record = value as Record<string, unknown>
    if (typeof record._id === 'string') return record._id
    if (typeof record.$oid === 'string') return record.$oid
    if (record._id) return legacy(record._id)
  }
  return String(value)
}

function stringValue(value: unknown, fallback: string) {
  return typeof value === 'string' ? value : fallback
}

function numberValue(value: unknown, fallback = 0) {
  const number = Number(value)
  return Number.isFinite(number) ? number : fallback
}

function dateValue(value: unknown, fallback: Date | null = new Date()) {
  if (!value) return fallback
  if (value instanceof Date) return value
  if (typeof value === 'object' && value && '$date' in value) {
    return dateValue((value as { $date: unknown }).$date, fallback)
  }
  const date = new Date(String(value))
  return Number.isNaN(date.getTime()) ? fallback : date
}

function statusValue(status: unknown) {
  const map = ['WAITING', 'AC', 'WA', 'TLE', 'MLE', 'RE', 'CE', 'SE', 'FROZEN'] as const
  return map[numberValue(status)] ?? 'SE'
}
