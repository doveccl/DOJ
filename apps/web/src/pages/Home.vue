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

const heroStats = computed(() => {
  const data = dashboard.value
  if (!data) return []
  return [
    { key: 'problems', label: t('dashboard.recentProblems'), value: data.recentProblems.length },
    { key: 'contests', label: t('dashboard.recentContests'), value: data.recentContests.length },
    {
      key: 'submissions',
      label: t('dashboard.recentSubmissions'),
      value: data.recentSubmissions.length
    }
  ]
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
        <section class="home-hero">
          <div class="hero-copy">
            <n-tag :bordered="false" type="success" size="small">{{ t('dashboard.badge') }}</n-tag>
            <h1>{{ t('dashboard.heroTitle') }}</h1>
            <p>{{ t('dashboard.heroIntro') }}</p>
            <div class="hero-actions">
              <n-button type="primary" @click="go('/problems')">
                {{ t('dashboard.startPractice') }}
              </n-button>
              <n-button secondary @click="go('/contests')">
                {{ t('dashboard.viewContests') }}
              </n-button>
              <n-button v-if="auth.signedIn" tertiary @click="go('/assignments')">
                {{ t('dashboard.myAssignments') }}
              </n-button>
            </div>
          </div>
          <div class="hero-panel">
            <div class="hero-panel-title">{{ t('dashboard.liveNow') }}</div>
            <div class="hero-metrics">
              <div v-for="item in heroStats" :key="item.key" class="hero-metric">
                <strong>{{ item.value }}</strong>
                <span>{{ item.label }}</span>
              </div>
            </div>
            <div
              v-if="primarySubmission"
              class="hero-latest"
              @click="go(`/submissions/${primarySubmission.id}`)"
            >
              <n-tag
                size="small"
                :bordered="false"
                :type="statusType[primarySubmission.status] ?? 'default'"
              >
                {{ primarySubmission.status }}
              </n-tag>
              <span>P{{ primarySubmission.problemId }} {{ primarySubmission.problemTitle }}</span>
            </div>
          </div>
        </section>

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
.home-hero {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1.45fr) minmax(320px, 0.8fr);
  gap: 20px;
  align-items: stretch;
  margin-bottom: 18px;
  overflow: hidden;
  border: 1px solid color-mix(in srgb, var(--brand) 16%, var(--border-color));
  border-radius: 18px;
  background:
    radial-gradient(
      circle at 12% 10%,
      color-mix(in srgb, var(--brand) 18%, transparent),
      transparent 34%
    ),
    linear-gradient(
      135deg,
      color-mix(in srgb, var(--surface-bg) 94%, var(--brand) 6%),
      var(--surface-bg)
    );
  box-shadow: var(--shadow-md);
}

.hero-copy {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  min-height: 260px;
  padding: 34px;

  h1 {
    max-width: 680px;
    margin: 14px 0 10px;
    font-size: clamp(30px, 4.8vw, 56px);
    line-height: 1.02;
    letter-spacing: -0.05em;
    overflow-wrap: anywhere;
    word-break: break-all;
  }

  p {
    max-width: 620px;
    margin: 0;
    color: var(--muted-color);
    font-size: 15px;
    line-height: 1.75;
    overflow-wrap: anywhere;
    word-break: break-all;
  }
}

.hero-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 22px;
}

.hero-panel {
  display: grid;
  align-content: center;
  gap: 18px;
  padding: 26px;
  background: color-mix(in srgb, var(--surface-bg) 78%, transparent);
  border-left: 1px solid color-mix(in srgb, var(--brand) 12%, var(--border-color));
}

.hero-panel-title {
  color: var(--muted-color);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.hero-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.hero-metric {
  display: grid;
  gap: 2px;
  padding: 12px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  background: var(--surface-bg);

  strong {
    font-size: 22px;
    line-height: 1;
  }

  span {
    overflow: hidden;
    color: var(--muted-color);
    font-size: 12px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.hero-latest,
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

.hero-latest {
  display: flex;
  gap: 10px;
  align-items: center;
  min-width: 0;
  padding: 12px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  background: var(--surface-bg);

  span:last-child {
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
  .home-hero {
    grid-template-columns: 1fr;
    border-radius: 14px;
  }

  .hero-copy,
  .hero-panel {
    min-height: auto;
    padding: 22px;
  }

  .hero-panel {
    border-top: 1px solid var(--border-color);
    border-left: 0;
  }

  .hero-metrics {
    grid-template-columns: 1fr;
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
