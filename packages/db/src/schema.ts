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
  'SE'
])

export const contestType = pgEnum('contest_type', ['OI', 'ICPC'])
export const taskStatus = pgEnum('task_status', ['WAITING', 'RUNNING', 'DONE', 'FAILED'])

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
    name: varchar('name', { length: 32 }).notNull(),
    email: varchar('email', { length: 320 }).notNull(),
    passwordHash: text('password_hash').notNull(),
    introduction: text('introduction').default('').notNull(),
    admin: boolean('admin').default(false).notNull(),
    mustChangePassword: boolean('must_change_password').default(false).notNull(),
    disabledAt: timestamp('disabled_at', { withTimezone: true }),
    createdAt: createdAt(),
    updatedAt: updatedAt()
  },
  (t) => ({
    nameUidx: uniqueIndex('users_name_uidx').on(sql`lower(${t.name})`),
    emailUidx: uniqueIndex('users_email_uidx').on(sql`lower(${t.email})`),
    createdIdx: index('users_created_idx').on(t.createdAt)
  })
)

export const groups = pgTable('groups', {
  id: id(),
  name: varchar('name', { length: 100 }).notNull(),
  createdAt: createdAt(),
  updatedAt: updatedAt()
})

export const userGroups = pgTable(
  'user_groups',
  {
    userId: integer('user_id')
      .notNull()
      .references(() => users.id, { onDelete: 'cascade' }),
    groupId: integer('group_id')
      .notNull()
      .references(() => groups.id, { onDelete: 'cascade' }),
    createdAt: createdAt()
  },
  (t) => ({
    pk: primaryKey({ columns: [t.userId, t.groupId] }),
    groupIdx: index('user_groups_group_idx').on(t.groupId)
  })
)

export const settings = pgTable('settings', {
  key: varchar('key', { length: 128 }).primaryKey(),
  value: jsonb('value').$type<unknown>().notNull(),
  updatedAt: updatedAt()
})

export const languages = pgTable(
  'languages',
  {
    id: varchar('id', { length: 32 }).primaryKey(),
    name: varchar('name', { length: 100 }).notNull(),
    source: varchar('source', { length: 128 }).notNull(),
    dockerfile: text('dockerfile').notNull(),
    sort: integer('sort').default(0).notNull(),
    createdAt: createdAt(),
    updatedAt: updatedAt()
  },
  (t) => ({
    sortIdx: index('languages_sort_idx').on(t.sort, t.id)
  })
)

export const problems = pgTable(
  'problems',
  {
    id: problemPrimaryId(),
    title: varchar('title', { length: 100 }).notNull(),
    mode: varchar('mode', { length: 16 }).default('default').notNull(),
    timeLimit: integer('time_limit').default(1000).notNull(),
    memoryLimit: bigint('memory_limit', { mode: 'number' }).default(268435456).notNull(),
    tags: text('tags')
      .array()
      .default(sql`ARRAY[]::text[]`)
      .notNull(),
    visible: boolean('visible').default(false).notNull(),
    deletedAt: timestamp('deleted_at', { withTimezone: true }),
    createdAt: createdAt(),
    updatedAt: updatedAt()
  },
  (t) => ({
    visibleIdx: index('problems_visible_idx').on(t.visible, t.deletedAt),
    createdIdx: index('problems_created_idx').on(t.createdAt)
  })
)

export const contests = pgTable(
  'contests',
  {
    id: id(),
    title: varchar('title', { length: 100 }).notNull(),
    description: text('description').default('').notNull(),
    type: contestType('type').default('OI').notNull(),
    startAt: timestamp('start_at', { withTimezone: true }).notNull(),
    endAt: timestamp('end_at', { withTimezone: true }).notNull(),
    freezeAt: timestamp('freeze_at', { withTimezone: true }),
    deletedAt: timestamp('deleted_at', { withTimezone: true }),
    createdAt: createdAt(),
    updatedAt: updatedAt()
  },
  (t) => ({
    startIdx: index('contests_start_idx').on(t.startAt)
  })
)

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
    sort: integer('sort').default(0).notNull()
  },
  (t) => ({
    pk: primaryKey({ columns: [t.contestId, t.problemId] }),
    keyUidx: uniqueIndex('contest_problems_key_uidx').on(t.contestId, t.key)
  })
)

export const assignments = pgTable(
  'assignments',
  {
    id: id(),
    title: varchar('title', { length: 100 }).notNull(),
    description: text('description').default('').notNull(),
    endAt: timestamp('end_at', { withTimezone: true }).notNull(),
    deletedAt: timestamp('deleted_at', { withTimezone: true }),
    createdAt: createdAt(),
    updatedAt: updatedAt()
  },
  (t) => ({
    createdIdx: index('assignments_created_idx').on(t.createdAt)
  })
)

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

export const assignmentUsers = pgTable(
  'assignment_users',
  {
    assignmentId: integer('assignment_id')
      .notNull()
      .references(() => assignments.id, { onDelete: 'cascade' }),
    userId: integer('user_id')
      .notNull()
      .references(() => users.id, { onDelete: 'cascade' })
  },
  (t) => ({
    pk: primaryKey({ columns: [t.assignmentId, t.userId] })
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
    sort: integer('sort').default(0).notNull()
  },
  (t) => ({
    pk: primaryKey({ columns: [t.assignmentId, t.problemId] })
  })
)

export const submissions = pgTable(
  'submissions',
  {
    id: id(),
    userId: integer('user_id')
      .notNull()
      .references(() => users.id),
    problemId: integer('problem_id')
      .notNull()
      .references(() => problems.id),
    contestId: integer('contest_id').references(() => contests.id),
    assignmentId: integer('assignment_id').references(() => assignments.id),
    languageId: varchar('language_id', { length: 32 })
      .notNull()
      .references(() => languages.id),
    code: text('code').notNull(),
    public: boolean('public').default(false).notNull(),
    status: judgeStatus('status').default('WAITING').notNull(),
    timeMs: integer('time_ms').default(0).notNull(),
    memoryBytes: bigint('memory_bytes', { mode: 'number' }).default(0).notNull(),
    score: integer('score').default(0).notNull(),
    message: text('message').default('').notNull(),
    createdAt: createdAt(),
    updatedAt: updatedAt()
  },
  (t) => ({
    userIdx: index('submissions_user_idx').on(t.userId, t.createdAt),
    problemIdx: index('submissions_problem_idx').on(t.problemId, t.createdAt),
    contestIdx: index('submissions_contest_idx').on(t.contestId, t.createdAt),
    assignmentIdx: index('submissions_assignment_idx').on(t.assignmentId, t.createdAt),
    statusIdx: index('submissions_status_idx').on(t.status),
    createdIdx: index('submissions_created_idx').on(t.createdAt)
  })
)

export const submissionCases = pgTable(
  'submission_cases',
  {
    submissionId: integer('submission_id')
      .notNull()
      .references(() => submissions.id, { onDelete: 'cascade' }),
    caseNo: integer('case_no').notNull(),
    status: judgeStatus('status').notNull(),
    timeMs: integer('time_ms').default(0).notNull(),
    memoryBytes: bigint('memory_bytes', { mode: 'number' }).default(0).notNull(),
    score: integer('score').default(0).notNull(),
    message: text('message').default('').notNull()
  },
  (t) => ({
    pk: primaryKey({ columns: [t.submissionId, t.caseNo] })
  })
)

export const judgeTasks = pgTable(
  'judge_tasks',
  {
    id: id(),
    submissionId: integer('submission_id')
      .notNull()
      .references(() => submissions.id, { onDelete: 'cascade' }),
    status: taskStatus('status').default('WAITING').notNull(),
    lockedUntil: timestamp('locked_until', { withTimezone: true }),
    lastError: text('last_error').default('').notNull(),
    createdAt: createdAt(),
    updatedAt: updatedAt()
  },
  (t) => ({
    submissionOpenUidx: uniqueIndex('judge_tasks_submission_open_uidx')
      .on(t.submissionId)
      .where(sql`${t.status} in ('WAITING', 'RUNNING')`),
    readyIdx: index('judge_tasks_ready_idx').on(t.status, t.createdAt),
    leaseIdx: index('judge_tasks_lease_idx').on(t.lockedUntil)
  })
)

export const topics = pgTable(
  'topics',
  {
    id: id(),
    title: varchar('title', { length: 100 }).notNull(),
    tags: text('tags')
      .array()
      .default(sql`ARRAY[]::text[]`)
      .notNull(),
    pinned: boolean('pinned').default(false).notNull(),
    deletedAt: timestamp('deleted_at', { withTimezone: true }),
    createdAt: createdAt(),
    updatedAt: updatedAt()
  },
  (t) => ({
    listIdx: index('topics_list_idx').on(t.pinned, t.updatedAt, t.id)
  })
)

export const posts = pgTable(
  'posts',
  {
    id: id(),
    topicId: integer('topic_id')
      .notNull()
      .references(() => topics.id, { onDelete: 'cascade' }),
    parentId: integer('parent_id'),
    userId: integer('user_id')
      .notNull()
      .references(() => users.id),
    content: text('content').notNull(),
    deletedAt: timestamp('deleted_at', { withTimezone: true }),
    createdAt: createdAt(),
    updatedAt: updatedAt()
  },
  (t) => ({
    topicIdx: index('posts_topic_idx').on(t.topicId, t.createdAt)
  })
)

export const usersRelations = relations(users, ({ many }) => ({
  groups: many(userGroups),
  posts: many(posts),
  submissions: many(submissions)
}))

export const groupsRelations = relations(groups, ({ many }) => ({
  users: many(userGroups),
  assignments: many(assignmentGroups)
}))

export const problemsRelations = relations(problems, ({ many }) => ({
  submissions: many(submissions),
  contests: many(contestProblems),
  assignments: many(assignmentProblems)
}))

export const topicsRelations = relations(topics, ({ many }) => ({
  posts: many(posts)
}))

export const postsRelations = relations(posts, ({ one }) => ({
  topic: one(topics, { fields: [posts.topicId], references: [topics.id] }),
  user: one(users, { fields: [posts.userId], references: [users.id] })
}))
