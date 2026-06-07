<script setup lang="ts">
import {
  NButton,
  NCard,
  NEmpty,
  NGi,
  NGrid,
  NList,
  NListItem,
  NSpace,
  NSpin,
  NStatistic,
  NTag,
  NThing
} from 'naive-ui'
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../api'
import { useAuthStore } from '../stores/auth'

interface Dashboard {
  stats: {
    problems?: number
    submissions?: number
    contests?: number
    users?: number
    assignments?: number
  }
  recentSubmissions: Array<{
    id: number
    status: string
    languageId: string
    timeMs: number
    memoryBytes: number
    createdAt: string
    userId: number
    userName: string
    problemId: number
    problemTitle: string
  }>
  recentProblems: Array<{
    id: number
    title: string
    tags: string[]
    solvedCount: number
    createdAt: string
  }>
  recentTopics: Array<{
    id: number
    title: string
    tags: string[]
    userName: string
    updatedAt: string
  }>
  recentContests: Array<{
    id: number
    title: string
    type: string
    startAt: string
    endAt: string
  }>
  myAssignments: Array<{
    id: number
    title: string
    dueAt: string | null
    allowLate: boolean
  }>
}

const loading = ref(true)
const error = ref('')
const dashboard = ref<Dashboard | null>(null)
const router = useRouter()
const { t } = useI18n()
const auth = useAuthStore()
const dateTime = new Intl.DateTimeFormat(undefined, {
  dateStyle: 'medium',
  timeStyle: 'short'
})

const STAT_ORDER = ['problems', 'submissions', 'contests', 'assignments', 'users'] as const
const STAT_LINKS: Record<(typeof STAT_ORDER)[number], string> = {
  problems: '/problems',
  submissions: '/submissions',
  contests: '/contests',
  assignments: '/assignments',
  users: '/rank'
}

const statCards = computed(() => {
  const stats = dashboard.value?.stats
  if (!stats) return []
  return STAT_ORDER.filter((key) => typeof stats[key] === 'number').map((key) => ({
    key,
    label: t(`dashboard.${key}`),
    value: stats[key] as number,
    to: STAT_LINKS[key]
  }))
})

const hasCards = computed(() => {
  const data = dashboard.value
  return Boolean(
    data &&
    (data.recentProblems.length ||
      data.myAssignments.length ||
      data.recentContests.length ||
      data.recentTopics.length)
  )
})

const primaryProblem = computed(() => dashboard.value?.recentProblems[0] ?? null)
const primaryContest = computed(() => dashboard.value?.recentContests[0] ?? null)
const primarySubmission = computed(() => dashboard.value?.recentSubmissions[0] ?? null)
const quickActions = computed(
  () =>
    [
      {
        key: 'problems',
        title: t('dashboard.quickProblems'),
        description: t('dashboard.quickProblemsDesc'),
        to: '/problems'
      },
      {
        key: 'contests',
        title: t('dashboard.quickContests'),
        description: primaryContest.value
          ? `${primaryContest.value.type} · ${formatDateTime(primaryContest.value.startAt)}`
          : t('dashboard.quickContestsDesc'),
        to: '/contests'
      },
      auth.signedIn
        ? {
            key: 'assignments',
            title: t('dashboard.quickAssignments'),
            description: dashboard.value?.myAssignments.length
              ? t('dashboard.quickAssignmentsCount', {
                  count: dashboard.value.myAssignments.length
                })
              : t('dashboard.quickAssignmentsDesc'),
            to: '/assignments'
          }
        : {
            key: 'discussion',
            title: t('dashboard.quickDiscussion'),
            description: t('dashboard.quickDiscussionDesc'),
            to: '/discussion'
          }
    ] as const
)
const dashboardSectionCount = computed(() => {
  const data = dashboard.value
  if (!data) return 0
  return [
    data.recentProblems.length,
    data.myAssignments.length,
    data.recentContests.length,
    data.recentTopics.length
  ].filter(Boolean).length
})

const statusType: Record<string, 'success' | 'warning' | 'error' | 'info' | 'default'> = {
  AC: 'success',
  WAITING: 'info',
  JUDGING: 'info',
  FROZEN: 'info',
  CE: 'warning',
  PE: 'warning',
  WA: 'error',
  RE: 'error',
  TLE: 'error',
  MLE: 'error',
  OLE: 'error',
  SE: 'error'
}

function formatDateTime(value: string | null) {
  return value ? dateTime.format(new Date(value)) : '-'
}

function go(path: string) {
  void router.push(path)
}

onMounted(async () => {
  try {
    dashboard.value = await apiFetch<Dashboard>('/api/dashboard')
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <main class="page">
    <n-spin :show="loading">
      <p v-if="error" class="form-error">{{ error }}</p>
      <template v-else-if="dashboard">
        <n-card class="home-panel" :bordered="false">
          <div class="home-panel-header">
            <div>
              <h1>{{ t('dashboard.workspaceTitle') }}</h1>
              <p>{{ t('dashboard.workspaceIntro') }}</p>
            </div>
            <n-space>
              <n-button type="primary" @click="go('/problems')">
                {{ t('dashboard.startPractice') }}
              </n-button>
              <n-button secondary @click="go('/submissions')">
                {{ t('dashboard.recentSubmissions') }}
              </n-button>
            </n-space>
          </div>

          <n-grid cols="1 720:3" :x-gap="12" :y-gap="12" class="quick-grid">
            <n-gi v-for="action in quickActions" :key="action.key">
              <n-card
                embedded
                hoverable
                :bordered="false"
                class="quick-card"
                @click="go(action.to)"
              >
                <n-thing :title="action.title" :description="action.description" />
              </n-card>
            </n-gi>
          </n-grid>

          <n-grid cols="1 820:2" :x-gap="12" :y-gap="12" class="current-grid">
            <n-gi>
              <div
                class="current-item"
                @click="primaryProblem && go(`/problems/${primaryProblem.id}`)"
              >
                <span class="current-label">{{ t('dashboard.continueProblem') }}</span>
                <strong v-if="primaryProblem"
                  >P{{ primaryProblem.id }} {{ primaryProblem.title }}</strong
                >
                <span v-else class="muted">{{ t('problems.empty') }}</span>
              </div>
            </n-gi>
            <n-gi>
              <div
                class="current-item"
                @click="primarySubmission && go(`/submissions/${primarySubmission.id}`)"
              >
                <span class="current-label">{{ t('dashboard.latestSubmission') }}</span>
                <div v-if="primarySubmission" class="latest-submission">
                  <n-tag
                    size="small"
                    :bordered="false"
                    :type="statusType[primarySubmission.status] ?? 'default'"
                  >
                    {{ primarySubmission.status }}
                  </n-tag>
                  <strong
                    >P{{ primarySubmission.problemId }} {{ primarySubmission.problemTitle }}</strong
                  >
                </div>
                <span v-else class="muted">{{ t('dashboard.noRecentSubmission') }}</span>
              </div>
            </n-gi>
          </n-grid>
        </n-card>

        <n-grid
          v-if="statCards.length"
          class="admin-stats"
          cols="2 640:3 980:5"
          :x-gap="12"
          :y-gap="12"
        >
          <n-gi v-for="card in statCards" :key="card.key">
            <n-card class="stat-card" hoverable embedded :bordered="false" @click="go(card.to)">
              <n-statistic :label="card.label" :value="card.value" />
            </n-card>
          </n-gi>
        </n-grid>

        <n-grid v-if="hasCards" class="dashboard-main" cols="1 900:2" :x-gap="16" :y-gap="16">
          <n-gi v-if="dashboard.recentProblems.length" :span="dashboardSectionCount === 1 ? 2 : 1">
            <n-card class="section-card" :bordered="false">
              <template #header>
                <div class="section-heading">
                  <span>{{ t('dashboard.recentProblems') }}</span>
                  <n-button text size="small" @click.stop="go('/problems')">
                    {{ t('dashboard.viewAll') }}
                  </n-button>
                </div>
              </template>
              <div
                v-if="primaryProblem"
                class="featured-item"
                @click="go(`/problems/${primaryProblem.id}`)"
              >
                <span class="featured-kicker">P{{ primaryProblem.id }}</span>
                <strong>{{ primaryProblem.title }}</strong>
                <span class="muted">{{ t('common.solved') }} {{ primaryProblem.solvedCount }}</span>
              </div>
              <n-list hoverable clickable>
                <n-list-item
                  v-for="problem in dashboard.recentProblems.slice(1, 5)"
                  :key="problem.id"
                  @click="go(`/problems/${problem.id}`)"
                >
                  <n-thing :title="`P${problem.id} ${problem.title}`">
                    <template #description>
                      <n-space :size="6" align="center">
                        <n-tag
                          v-for="tag in problem.tags"
                          :key="tag"
                          size="small"
                          :bordered="false"
                        >
                          {{ tag }}
                        </n-tag>
                        <span class="muted"
                          >{{ t('common.solved') }} {{ problem.solvedCount }}</span
                        >
                      </n-space>
                    </template>
                  </n-thing>
                </n-list-item>
              </n-list>
            </n-card>
          </n-gi>

          <n-gi v-if="dashboard.myAssignments.length" :span="dashboardSectionCount === 1 ? 2 : 1">
            <n-card class="section-card" :bordered="false">
              <template #header>
                <div class="section-heading">
                  <span>{{ t('dashboard.myAssignments') }}</span>
                  <n-button text size="small" @click.stop="go('/assignments')">
                    {{ t('dashboard.viewAll') }}
                  </n-button>
                </div>
              </template>
              <n-list hoverable clickable>
                <n-list-item
                  v-for="assignment in dashboard.myAssignments.slice(0, 5)"
                  :key="assignment.id"
                  @click="go(`/assignments/${assignment.id}`)"
                >
                  <n-thing :title="assignment.title">
                    <template #description>
                      <span class="muted">
                        {{ t('assignments.duePrefix') }} {{ formatDateTime(assignment.dueAt) }}
                      </span>
                    </template>
                  </n-thing>
                </n-list-item>
              </n-list>
            </n-card>
          </n-gi>

          <n-gi v-if="dashboard.recentContests.length" :span="dashboardSectionCount === 1 ? 2 : 1">
            <n-card class="section-card" :bordered="false">
              <template #header>
                <div class="section-heading">
                  <span>{{ t('dashboard.recentContests') }}</span>
                  <n-button text size="small" @click.stop="go('/contests')">
                    {{ t('dashboard.viewAll') }}
                  </n-button>
                </div>
              </template>
              <div
                v-if="primaryContest"
                class="featured-item"
                @click="go(`/contests/${primaryContest.id}`)"
              >
                <span class="featured-kicker">{{ primaryContest.type }}</span>
                <strong>{{ primaryContest.title }}</strong>
                <span class="muted">{{ formatDateTime(primaryContest.startAt) }}</span>
              </div>
              <n-list hoverable clickable>
                <n-list-item
                  v-for="contest in dashboard.recentContests.slice(1, 5)"
                  :key="contest.id"
                  @click="go(`/contests/${contest.id}`)"
                >
                  <n-thing :title="contest.title">
                    <template #description>
                      <span class="muted">
                        {{ contest.type }} · {{ formatDateTime(contest.startAt) }}
                      </span>
                    </template>
                  </n-thing>
                </n-list-item>
              </n-list>
            </n-card>
          </n-gi>

          <n-gi v-if="dashboard.recentTopics.length" :span="dashboardSectionCount === 1 ? 2 : 1">
            <n-card class="section-card" :bordered="false">
              <template #header>
                <div class="section-heading">
                  <span>{{ t('dashboard.recentTopics') }}</span>
                  <n-button text size="small" @click.stop="go('/discussion')">
                    {{ t('dashboard.viewAll') }}
                  </n-button>
                </div>
              </template>
              <n-list hoverable clickable>
                <n-list-item
                  v-for="topic in dashboard.recentTopics.slice(0, 5)"
                  :key="topic.id"
                  @click="go(`/discussion/${topic.id}`)"
                >
                  <n-thing :title="topic.title">
                    <template #description>
                      <span class="muted">
                        {{ topic.userName }} · {{ formatDateTime(topic.updatedAt) }}
                      </span>
                    </template>
                  </n-thing>
                </n-list-item>
              </n-list>
            </n-card>
          </n-gi>
        </n-grid>

        <n-card
          v-if="dashboard.recentSubmissions.length"
          :bordered="false"
          class="section-card stacked-card"
        >
          <template #header>
            <div class="section-heading">
              <span>{{ t('dashboard.recentSubmissions') }}</span>
              <n-button text size="small" @click.stop="go('/submissions')">
                {{ t('dashboard.viewAll') }}
              </n-button>
            </div>
          </template>
          <n-list hoverable clickable>
            <n-list-item
              v-for="submission in dashboard.recentSubmissions.slice(0, 8)"
              :key="submission.id"
              @click="go(`/submissions/${submission.id}`)"
            >
              <div class="submission-row">
                <n-tag
                  size="small"
                  :bordered="false"
                  :type="statusType[submission.status] ?? 'default'"
                >
                  {{ submission.status }}
                </n-tag>
                <span class="submission-title">
                  P{{ submission.problemId }} {{ submission.problemTitle }}
                </span>
                <span class="muted submission-meta">
                  {{ submission.userName }} · {{ submission.languageId }} ·
                  {{ formatDateTime(submission.createdAt) }}
                </span>
              </div>
            </n-list-item>
          </n-list>
        </n-card>

        <n-empty
          v-if="!statCards.length && !hasCards && !dashboard.recentSubmissions.length"
          :description="t('dashboard.noTopics')"
        />
      </template>
    </n-spin>
  </main>
</template>

<style scoped lang="scss">
.home-panel {
  margin-bottom: 16px;
  box-shadow: var(--shadow-sm);
}

.home-panel-header {
  display: flex;
  gap: 16px;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 16px;

  h1 {
    margin: 0;
    font-size: 24px;
    line-height: 1.2;
    letter-spacing: -0.02em;
  }

  p {
    margin: 6px 0 0;
    color: var(--muted-color);
  }
}

.quick-grid,
.current-grid {
  margin-top: 12px;
}

.quick-card,
.current-item,
.featured-item {
  cursor: pointer;
  transition:
    border-color 0.15s ease,
    transform 0.15s ease;

  &:hover {
    border-color: var(--brand);
    transform: translateY(-1px);
  }
}

.quick-card {
  min-height: 92px;
}

.current-item {
  display: grid;
  gap: 6px;
  min-height: 82px;
  padding: 14px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  background: color-mix(in srgb, var(--surface-bg) 94%, var(--brand) 6%);
}

.current-label {
  color: var(--muted-color);
  font-size: 12px;
  font-weight: 600;
}

.latest-submission {
  display: flex;
  gap: 8px;
  align-items: center;
  min-width: 0;

  strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.admin-stats {
  margin-bottom: 16px;
}

.stat-card {
  cursor: pointer;

  :deep(.n-statistic .n-statistic-value__content) {
    font-weight: 600;
  }
}

.dashboard-main {
  margin-top: 16px;
}

.section-card {
  box-shadow: var(--shadow-sm);
}

.section-heading {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  font-weight: 700;
}

.featured-item {
  display: grid;
  gap: 5px;
  padding: 14px;
  margin-bottom: 6px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  background: color-mix(in srgb, var(--surface-bg) 88%, var(--brand) 12%);

  strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.featured-kicker {
  color: var(--brand);
  font-size: 12px;
  font-weight: 700;
}

.submission-row {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
}

.submission-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.submission-meta {
  text-align: right;
  white-space: nowrap;
}

@media (max-width: 640px) {
  .home-panel-header {
    display: grid;
  }

  .submission-row {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .submission-meta {
    grid-column: 1 / -1;
    text-align: left;
  }
}
</style>
