import { relations, sql } from 'drizzle-orm'
import {
  bigint,
  boolean,
  index,
  integer,
  jsonb,
  pgEnum,
  pgTable,
  primaryKey,
  text,
  timestamp,
  uniqueIndex,
  varchar
} from 'drizzle-orm/pg-core'
import type { ProblemTestCase } from '@doj/shared/judge'

export const judgeStatus = pgEnum('judge_status', [
  'WAITING',
  'JUDGING',
  'AC',
  'WA',
  'PE',
  'TLE',
  'MLE',
  'OLE',
  'RE',
  'CE',
  'SE',
  'FROZEN'
])

export const contestType = pgEnum('contest_type', ['OI', 'ICPC'])
export const taskStatus = pgEnum('task_status', [
  'WAITING',
  'RUNNING',
  'DONE',
  'FAILED',
  'CANCELLED'
])

const id = () => integer('id').primaryKey().generatedByDefaultAsIdentity()
const problemPrimaryId = () =>
  integer('id').primaryKey().generatedByDefaultAsIdentity({
    startWith: 1000,
    name: 'problems_id_seq'
  })
const createdAt = () => timestamp('created_at', { withTimezone: true }).defaultNow().notNull()
const updatedAt = () => timestamp('updated_at', { withTimezone: true }).defaultNow().notNull()

export const users = pgTable(
  'users',
  {
    id: id(),
    legacyId: varchar('legacy_id', { length: 64 }),
    name: varchar('name', { length: 32 }).notNull(),
    email: varchar('email', { length: 255 }).notNull(),
    passwordHash: text('password_hash').notNull(),
    introduction: varchar('introduction', { length: 500 }).default('').notNull(),
    solvedCount: integer('solved_count').default(0).notNull(),
    submissionCount: integer('submission_count').default(0).notNull(),
    disabledAt: timestamp('disabled_at', { withTimezone: true }),
    createdAt: createdAt(),
    updatedAt: updatedAt()
  },
  (t) => ({
    nameUidx: uniqueIndex('users_name_uidx').on(sql`lower(${t.name})`),
    emailUidx: uniqueIndex('users_email_uidx').on(sql`lower(${t.email})`),
    rankIdx: index('users_rank_idx').on(t.solvedCount, t.submissionCount)
  })
)

export const groups = pgTable(
  'groups',
  {
    id: id(),
    key: varchar('key', { length: 64 }).notNull(),
    name: varchar('name', { length: 128 }).notNull(),
    description: text('description').default('').notNull(),
    builtin: boolean('builtin').default(false).notNull(),
    createdAt: createdAt(),
    updatedAt: updatedAt()
  },
  (t) => ({
    keyUidx: uniqueIndex('groups_key_uidx').on(t.key)
  })
)

export const systemSettings = pgTable('system_settings', {
  key: varchar('key', { length: 128 }).primaryKey(),
  value: jsonb('value').$type<unknown>().notNull(),
  updatedAt: updatedAt()
})

export const judgeLanguages = pgTable(
  'judge_languages',
  {
    id: varchar('id', { length: 64 }).primaryKey(),
    name: varchar('name', { length: 128 }).notNull(),
    enabled: boolean('enabled').default(true).notNull(),
    sourceFile: varchar('source_file', { length: 128 }).notNull(),
    dockerfile: text('dockerfile').notNull(),
    command: text('command')
      .array()
      .default(sql`ARRAY[]::text[]`)
      .notNull(),
    sortOrder: integer('sort_order').default(0).notNull(),
    createdAt: createdAt(),
    updatedAt: updatedAt()
  },
  (t) => ({
    enabledIdx: index('judge_languages_enabled_idx').on(t.enabled, t.sortOrder)
  })
)

export const judgeAgents = pgTable(
  'judge_agents',
  {
    id: id(),
    key: varchar('key', { length: 64 }).notNull(),
    name: varchar('name', { length: 128 }).notNull(),
    enabled: boolean('enabled').default(true).notNull(),
    tokenHash: text('token_hash').notNull(),
    labels: text('labels')
      .array()
      .default(sql`ARRAY[]::text[]`)
      .notNull(),
    concurrency: integer('concurrency').default(2).notNull(),
    sortOrder: integer('sort_order').default(0).notNull(),
    lastSeenAt: timestamp('last_seen_at', { withTimezone: true }),
    createdAt: createdAt(),
    updatedAt: updatedAt()
  },
  (t) => ({
    keyUidx: uniqueIndex('judge_agents_key_uidx').on(t.key),
    enabledIdx: index('judge_agents_enabled_idx').on(t.enabled, t.sortOrder),
    lastSeenIdx: index('judge_agents_last_seen_idx').on(t.lastSeenAt)
  })
)

export const userGroups = pgTable(
  'user_groups',
  {
    userId: integer('user_id')
      .notNull()
      .references(() => users.id, { onDelete: 'cascade' }),
    groupId: integer('group_id')
      .notNull()
      .references(() => groups.id, { onDelete: 'cascade' }),
    manager: boolean('manager').default(false).notNull(),
    createdAt: createdAt()
  },
  (t) => ({
    pk: primaryKey({ columns: [t.userId, t.groupId] }),
    groupIdx: index('user_groups_group_idx').on(t.groupId)
  })
)

export const files = pgTable(
  'files',
  {
    id: id(),
    legacyId: varchar('legacy_id', { length: 64 }),
    bucket: varchar('bucket', { length: 128 }).notNull(),
    objectKey: text('object_key').notNull(),
    filename: text('filename').notNull(),
    contentType: varchar('content_type', { length: 255 }).notNull(),
    sizeBytes: bigint('size_bytes', { mode: 'number' }).notNull(),
    metadata: jsonb('metadata').$type<Record<string, unknown>>().default({}).notNull(),
    createdAt: createdAt()
  },
  (t) => ({
    objectUidx: uniqueIndex('files_object_uidx').on(t.bucket, t.objectKey)
  })
)

export const problems = pgTable(
  'problems',
  {
    id: problemPrimaryId(),
    legacyId: varchar('legacy_id', { length: 64 }),
    title: varchar('title', { length: 160 }).notNull(),
    tags: text('tags')
      .array()
      .default(sql`ARRAY[]::text[]`)
      .notNull(),
    visible: boolean('visible').default(true).notNull(),
    solvedCount: integer('solved_count').default(0).notNull(),
    submissionCount: integer('submission_count').default(0).notNull(),
    createdAt: createdAt(),
    updatedAt: updatedAt(),
    deletedAt: timestamp('deleted_at', { withTimezone: true })
  },
  (t) => ({
    visibleIdx: index('problems_visible_idx').on(t.visible)
  })
)

export const problemVersions = pgTable(
  'problem_versions',
  {
    id: id(),
    problemId: integer('problem_id')
      .notNull()
      .references(() => problems.id),
    version: integer('version').notNull(),
    statementMarkdown: text('statement_markdown').notNull(),
    timeLimitMs: integer('time_limit_ms').default(1000).notNull(),
    memoryLimitBytes: bigint('memory_limit_bytes', { mode: 'number' }).default(268435456).notNull(),
    outputLimitBytes: integer('output_limit_bytes').default(67108864).notNull(),
    testCases: jsonb('test_cases').$type<ProblemTestCase[]>().default([]).notNull(),
    testdataFileId: integer('testdata_file_id').references(() => files.id),
    checkerFileId: integer('checker_file_id').references(() => files.id),
    interactorFileId: integer('interactor_file_id').references(() => files.id),
    createdAt: createdAt()
  },
  (t) => ({
    versionUidx: uniqueIndex('problem_versions_problem_version_uidx').on(t.problemId, t.version)
  })
)

export const contests = pgTable('contests', {
  id: id(),
  legacyId: varchar('legacy_id', { length: 64 }),
  title: varchar('title', { length: 160 }).notNull(),
  description: text('description').default('').notNull(),
  type: contestType('type').default('OI').notNull(),
  startAt: timestamp('start_at', { withTimezone: true }).notNull(),
  endAt: timestamp('end_at', { withTimezone: true }).notNull(),
  freezeAt: timestamp('freeze_at', { withTimezone: true }),
  createdAt: createdAt(),
  updatedAt: updatedAt()
})

export const contestProblems = pgTable(
  'contest_problems',
  {
    contestId: integer('contest_id')
      .notNull()
      .references(() => contests.id, { onDelete: 'cascade' }),
    problemId: integer('problem_id')
      .notNull()
      .references(() => problems.id),
    key: varchar('key', { length: 32 }).notNull(),
    score: integer('score').default(100).notNull(),
    sortOrder: integer('sort_order').default(0).notNull()
  },
  (t) => ({
    pk: primaryKey({ columns: [t.contestId, t.problemId] }),
    keyUidx: uniqueIndex('contest_problems_key_uidx').on(t.contestId, t.key)
  })
)

export const assignments = pgTable('assignments', {
  id: id(),
  title: varchar('title', { length: 160 }).notNull(),
  description: text('description').default('').notNull(),
  startAt: timestamp('start_at', { withTimezone: true }),
  dueAt: timestamp('due_at', { withTimezone: true }),
  allowLate: boolean('allow_late').default(false).notNull(),
  aiCoachingEnabled: boolean('ai_coaching_enabled').default(true).notNull(),
  createdAt: createdAt(),
  updatedAt: updatedAt()
})

export const assignmentGroups = pgTable(
  'assignment_groups',
  {
    assignmentId: integer('assignment_id')
      .notNull()
      .references(() => assignments.id, { onDelete: 'cascade' }),
    groupId: integer('group_id')
      .notNull()
      .references(() => groups.id, { onDelete: 'cascade' })
  },
  (t) => ({
    pk: primaryKey({ columns: [t.assignmentId, t.groupId] })
  })
)

export const assignmentProblems = pgTable(
  'assignment_problems',
  {
    assignmentId: integer('assignment_id')
      .notNull()
      .references(() => assignments.id, { onDelete: 'cascade' }),
    problemId: integer('problem_id')
      .notNull()
      .references(() => problems.id),
    score: integer('score').default(100).notNull(),
    sortOrder: integer('sort_order').default(0).notNull()
  },
  (t) => ({
    pk: primaryKey({ columns: [t.assignmentId, t.problemId] })
  })
)

export const submissions = pgTable(
  'submissions',
  {
    id: id(),
    legacyId: varchar('legacy_id', { length: 64 }),
    userId: integer('user_id').notNull(),
    problemId: integer('problem_id').notNull(),
    problemVersionId: integer('problem_version_id').notNull(),
    contestId: integer('contest_id'),
    assignmentId: integer('assignment_id'),
    languageId: varchar('language_id', { length: 64 }).notNull(),
    sourceCode: text('source_code').notNull(),
    open: boolean('open').default(false).notNull(),
    status: judgeStatus('status').default('WAITING').notNull(),
    timeMs: integer('time_ms').default(0).notNull(),
    memoryBytes: bigint('memory_bytes', { mode: 'number' }).default(0).notNull(),
    message: text('message').default('').notNull(),
    createdAt: createdAt(),
    updatedAt: updatedAt()
  },
  (t) => ({
    userIdx: index('submissions_user_idx').on(t.userId, t.createdAt),
    problemIdx: index('submissions_problem_idx').on(t.problemId, t.createdAt),
    contestIdx: index('submissions_contest_idx').on(t.contestId, t.createdAt),
    assignmentIdx: index('submissions_assignment_idx').on(t.assignmentId, t.createdAt),
    statusIdx: index('submissions_status_idx').on(t.status)
  })
)

export const submissionCases = pgTable(
  'submission_cases',
  {
    submissionId: integer('submission_id').notNull(),
    caseIndex: integer('case_index').notNull(),
    status: judgeStatus('status').notNull(),
    timeMs: integer('time_ms').default(0).notNull(),
    memoryBytes: bigint('memory_bytes', { mode: 'number' }).default(0).notNull(),
    message: text('message').default('').notNull()
  },
  (t) => ({
    pk: primaryKey({ columns: [t.submissionId, t.caseIndex] })
  })
)

export const solvedProblems = pgTable(
  'solved_problems',
  {
    userId: integer('user_id').notNull(),
    problemId: integer('problem_id').notNull(),
    firstSubmissionId: integer('first_submission_id').notNull(),
    createdAt: createdAt()
  },
  (t) => ({
    pk: primaryKey({ columns: [t.userId, t.problemId] }),
    problemIdx: index('solved_problems_problem_idx').on(t.problemId)
  })
)

export const judgeTasks = pgTable(
  'judge_tasks',
  {
    id: id(),
    submissionId: integer('submission_id').notNull(),
    status: taskStatus('status').default('WAITING').notNull(),
    priority: integer('priority').default(0).notNull(),
    attempts: integer('attempts').default(0).notNull(),
    maxAttempts: integer('max_attempts').default(3).notNull(),
    lockedBy: varchar('locked_by', { length: 128 }),
    lockedUntil: timestamp('locked_until', { withTimezone: true }),
    lastError: text('last_error').default('').notNull(),
    createdAt: createdAt(),
    updatedAt: updatedAt()
  },
  (t) => ({
    readyIdx: index('judge_tasks_ready_idx').on(t.status, t.priority, t.createdAt),
    leaseIdx: index('judge_tasks_lease_idx').on(t.lockedUntil)
  })
)

export const bbsTopics = pgTable(
  'bbs_topics',
  {
    id: id(),
    userId: integer('user_id').notNull(),
    title: varchar('title', { length: 160 }).notNull(),
    tags: text('tags')
      .array()
      .default(sql`ARRAY[]::text[]`)
      .notNull(),
    linkedProblemId: integer('linked_problem_id'),
    linkedContestId: integer('linked_contest_id'),
    createdAt: createdAt(),
    updatedAt: updatedAt()
  },
  (t) => ({
    tagIdx: index('bbs_topics_tags_idx').on(t.tags)
  })
)

export const bbsReplies = pgTable(
  'bbs_replies',
  {
    id: id(),
    topicId: integer('topic_id')
      .notNull()
      .references(() => bbsTopics.id, { onDelete: 'cascade' }),
    userId: integer('user_id').notNull(),
    contentMarkdown: text('content_markdown').notNull(),
    createdAt: createdAt(),
    updatedAt: updatedAt()
  },
  (t) => ({
    topicIdx: index('bbs_replies_topic_idx').on(t.topicId, t.createdAt)
  })
)

export const aiCoachingSessions = pgTable(
  'ai_coaching_sessions',
  {
    id: id(),
    userId: integer('user_id').notNull(),
    submissionId: integer('submission_id').notNull(),
    model: varchar('model', { length: 128 }).notNull(),
    promptVersion: varchar('prompt_version', { length: 64 }).notNull(),
    responseMarkdown: text('response_markdown').notNull(),
    metadata: jsonb('metadata').$type<Record<string, unknown>>().default({}).notNull(),
    createdAt: createdAt()
  },
  (t) => ({
    submissionIdx: index('ai_coaching_submission_idx').on(t.submissionId, t.createdAt)
  })
)

export const usersRelations = relations(users, ({ many }) => ({
  groups: many(userGroups)
}))

export const groupsRelations = relations(groups, ({ many }) => ({
  users: many(userGroups)
}))

export const problemsRelations = relations(problems, ({ many }) => ({
  versions: many(problemVersions)
}))
