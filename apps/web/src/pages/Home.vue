<script setup lang="ts">
import {
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
        <n-grid v-if="statCards.length" cols="2 640:3 980:5" :x-gap="16" :y-gap="16">
          <n-gi v-for="card in statCards" :key="card.key">
            <n-card class="stat-card" hoverable embedded :bordered="false" @click="go(card.to)">
              <n-statistic :label="card.label" :value="card.value" />
            </n-card>
          </n-gi>
        </n-grid>

        <n-grid v-if="hasCards" class="dashboard-main" cols="1 900:2" :x-gap="16" :y-gap="16">
          <n-gi v-if="dashboard.recentProblems.length">
            <n-card :title="t('dashboard.recentProblems')" size="small">
              <n-list hoverable clickable>
                <n-list-item
                  v-for="problem in dashboard.recentProblems.slice(0, 5)"
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
                        <span class="muted">{{ t('common.solved') }} {{ problem.solvedCount }}</span>
                      </n-space>
                    </template>
                  </n-thing>
                </n-list-item>
              </n-list>
            </n-card>
          </n-gi>

          <n-gi v-if="dashboard.myAssignments.length">
            <n-card :title="t('dashboard.myAssignments')" size="small">
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

          <n-gi v-if="dashboard.recentContests.length">
            <n-card :title="t('dashboard.recentContests')" size="small">
              <n-list hoverable clickable>
                <n-list-item
                  v-for="contest in dashboard.recentContests.slice(0, 5)"
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

          <n-gi v-if="dashboard.recentTopics.length">
            <n-card :title="t('dashboard.recentTopics')" size="small">
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
          :title="t('dashboard.recentSubmissions')"
          size="small"
          class="stacked-card"
        >
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
.stat-card {
  cursor: pointer;

  :deep(.n-statistic .n-statistic-value__content) {
    font-weight: 600;
  }
}

.dashboard-main {
  margin-top: 16px;
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
  .submission-row {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .submission-meta {
    grid-column: 1 / -1;
    text-align: left;
  }
}
</style>
