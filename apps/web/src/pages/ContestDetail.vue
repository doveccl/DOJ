<script setup lang="ts">
import { NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { apiFetch, DEFAULT_PAGE_SIZE, getItems, PAGE_SIZE_OPTIONS, type Paged } from '../api'
import { useAuthStore } from '../stores/auth'

interface Contest {
  id: number
  title: string
  description: string
  type: 'OI' | 'ICPC'
  startAt: string
  endAt: string
  freezeAt: string | null
  deletedAt?: string | null
}

interface ContestProblem {
  id: number
  problemId?: number
  key: string
  title: string | null
  sort: number
  unavailable?: boolean
}

interface ContestDetail {
  contest: Contest
  problems: ContestProblem[]
}

interface ScoreboardRow {
  user: { id: number; name: string; avatarUrl: string }
  rank: number | null
  totalScore?: number
  effectiveAt?: string | null
  solved?: number
  penalty?: number
  problems: Record<string, {
    submitted: boolean
    pending: boolean
    score?: number | null
    status?: string | null
    accepted?: boolean
    attempts?: number
    wrongAttempts?: number
    penalty?: number
    submissionId?: number | null
  }>
}

interface Scoreboard {
  contest: { frozen: boolean; mode: 'public' | 'full'; type: 'OI' | 'ICPC' }
  problems: { problemId: number; key: string; title: string; sort: number }[]
  rows: ScoreboardRow[]
  page: number
  pageSize: number
  total: number
}

interface SubmissionRow {
  id: number
  languageId: string
  status: string | null
  displayStatus: string
  timeMs: number | null
  memoryBytes: number | null
  score: number | null
  createdAt: string
  user: { id: number; name: string; avatarUrl?: string }
  problem: { id: number; title: string } | null
}

const route = useRoute()
const auth = useAuthStore()
const loading = ref(true)
const scoreboardLoading = ref(false)
const submissionsLoading = ref(false)
const error = ref('')
const scoreError = ref('')
const detail = ref<ContestDetail | null>(null)
const scoreboard = ref<Scoreboard | null>(null)
const submissions = ref<SubmissionRow[]>([])
const showFullScoreboard = ref(false)
const activeTab = ref('problems')
const canViewFullScoreboard = computed(() => auth.user?.admin ?? false)
const canManage = computed(() => auth.user?.admin ?? false)
const { t } = useI18n()
const scoreboardPage = ref(1)
const scoreboardPageSize = ref(DEFAULT_PAGE_SIZE)
const submissionsPage = ref(1)
const submissionsPageSize = ref(DEFAULT_PAGE_SIZE)
const submissionsTotal = ref(0)

const columns = computed<DataTableColumns<ContestProblem>>(() => [
  { title: t('contests.key'), key: 'key', width: 90 },
  {
    title: t('common.problem'),
    key: 'title',
    render(row) {
      const problemId = row.problemId ?? row.id
      if (row.unavailable || !problemId) return row.title ?? t('assignments.unavailable')
      return h(
        RouterLink,
        {
          to: `/problems/${problemId}?contestId=${detail.value?.contest.id}`,
          class: 'table-link'
        },
        () => row.title ?? `P${problemId}`
      )
    }
  }
])

const scoreboardColumns = computed<DataTableColumns<ScoreboardRow>>(() => [
  {
    title: '#',
    key: 'rank',
    width: 72,
    render(row) {
      return row.rank ?? '-'
    }
  },
  {
    title: t('common.user'),
    key: 'user',
    minWidth: 160,
    render(row) {
      return row.user.name
    }
  },
  {
    title: detail.value?.contest.type === 'OI' ? t('submissions.score') : t('common.solved'),
    key: 'summary',
    width: 120,
    render(row) {
      return detail.value?.contest.type === 'OI' ? (row.totalScore ?? '-') : (row.solved ?? 0)
    }
  },
  {
    title: t('contests.penalty'),
    key: 'penalty',
    width: 110,
    render(row) {
      return detail.value?.contest.type === 'ICPC' ? (row.penalty ?? 0) : '-'
    }
  },
  ...(detail.value?.problems.map((problem) => ({
    title: problem.key,
    key: problem.key,
    width: 110,
    render(row: ScoreboardRow) {
      const cell = row.problems[problem.key]
      if (!cell?.submitted) return '-'
      if (cell.pending) return h(NTag, { bordered: false, type: 'warning' }, () => '?')
      if (detail.value?.contest.type === 'OI') return cell.score ?? '-'
      return cell.accepted ? `+${cell.attempts && cell.attempts > 1 ? cell.attempts - 1 : ''}` : `-${cell.attempts ?? 0}`
    }
  })) ?? [])
])

const submissionColumns = computed<DataTableColumns<SubmissionRow>>(() => [
  {
    title: t('common.status'),
    key: 'status',
    width: 110,
    render(row) {
      return h(NTag, { bordered: false }, () => row.displayStatus ?? row.status ?? '-')
    }
  },
  {
    title: t('common.id'),
    key: 'id',
    width: 90,
    render(row) {
      return h(RouterLink, { to: `/submissions/${row.id}`, class: 'table-link' }, () => String(row.id))
    }
  },
  {
    title: t('common.problem'),
    key: 'problem',
    minWidth: 180,
    render(row) {
      if (!row.problem) return '-'
      return h(RouterLink, { to: `/problems/${row.problem.id}?contestId=${detail.value?.contest.id}`, class: 'table-link' }, () => row.problem?.title ?? '-')
    }
  },
  {
    title: t('common.user'),
    key: 'user',
    minWidth: 140,
    render(row) {
      return row.user.name
    }
  },
  { title: t('common.language'), key: 'languageId', width: 120 },
  {
    title: t('submissions.score'),
    key: 'score',
    width: 90,
    render(row) {
      return row.score ?? '-'
    }
  },
  {
    title: t('submissions.createdAt'),
    key: 'createdAt',
    minWidth: 180,
    render(row) {
      return new Date(row.createdAt).toLocaleString()
    }
  }
])

onMounted(() => {
  void loadDetail()
})

async function loadDetail() {
  loading.value = true
  error.value = ''
  try {
    detail.value = await apiFetch<ContestDetail>(`/api/contests/${route.params.id}`)
    if (new Date() >= new Date(detail.value.contest.startAt)) void loadScoreboard(false)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

async function loadScoreboard(full = false) {
  scoreboardLoading.value = true
  scoreError.value = ''
  showFullScoreboard.value = full
  try {
    const params = new URLSearchParams({
      page: String(scoreboardPage.value),
      pageSize: String(scoreboardPageSize.value)
    })
    scoreboard.value = await apiFetch<Scoreboard>(
      full
        ? `/api/admin/contests/${route.params.id}/scoreboard/full?${params}`
        : `/api/contests/${route.params.id}/scoreboard?${params}`
    )
  } catch (cause) {
    scoreError.value = cause instanceof Error ? cause.message : String(cause)
    scoreboard.value = null
  } finally {
    scoreboardLoading.value = false
  }
}

async function loadSubmissions() {
  submissionsLoading.value = true
  error.value = ''
  try {
    const params = new URLSearchParams({
      page: String(submissionsPage.value),
      pageSize: String(submissionsPageSize.value),
      contestId: String(route.params.id)
    })
    const data = await apiFetch<Paged<SubmissionRow>>(`/api/submissions?${params}`)
    submissions.value = getItems(data)
    submissionsTotal.value = data.total
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    submissionsLoading.value = false
  }
}

function handleTabUpdate(tab: string) {
  activeTab.value = tab
  if (tab === 'scoreboard' && !scoreboard.value && !scoreboardLoading.value) void loadScoreboard(showFullScoreboard.value)
  if (tab === 'submissions' && !submissions.value.length && !submissionsLoading.value) void loadSubmissions()
}

function changeScoreboardPage(page: number) {
  scoreboardPage.value = page
  void loadScoreboard(showFullScoreboard.value)
}

function changeScoreboardPageSize(pageSize: number) {
  scoreboardPageSize.value = pageSize
  scoreboardPage.value = 1
  void loadScoreboard(showFullScoreboard.value)
}

function changeSubmissionsPage(page: number) {
  submissionsPage.value = page
  void loadSubmissions()
}

function changeSubmissionsPageSize(pageSize: number) {
  submissionsPageSize.value = pageSize
  submissionsPage.value = 1
  void loadSubmissions()
}

async function toggleDeleted() {
  if (!detail.value) return
  try {
    await apiFetch(
      detail.value.contest.deletedAt
        ? `/api/admin/contests/${detail.value.contest.id}/restore`
        : `/api/admin/contests/${detail.value.contest.id}`,
      { method: detail.value.contest.deletedAt ? 'POST' : 'DELETE' }
    )
    await loadDetail()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  }
}
</script>

<template>
  <main class="page">
    <n-spin :show="loading">
      <template v-if="detail">
        <section class="page-header">
          <h1>{{ detail.contest.title }}</h1>
          <p v-if="detail.contest.description">{{ detail.contest.description }}</p>
        </section>

        <n-card :bordered="false" class="stacked-card">
          <div class="meta-row">
            <n-tag :bordered="false">{{ detail.contest.type }}</n-tag>
            <n-tag v-if="detail.contest.deletedAt" :bordered="false" type="error">
              {{ t('admin.contests.deleted') }}
            </n-tag>
            <span class="muted">{{ new Date(detail.contest.startAt).toLocaleString() }}</span>
            <span class="muted"
              >{{ t('contests.to') }} {{ new Date(detail.contest.endAt).toLocaleString() }}</span
            >
            <n-tag v-if="detail.contest.freezeAt" :bordered="false" type="warning">
              {{ t('contests.freezes') }} {{ new Date(detail.contest.freezeAt).toLocaleString() }}
            </n-tag>
            <n-space v-if="canManage" size="small">
              <n-popconfirm @positive-click="toggleDeleted">
                <template #trigger>
                  <n-button size="small" tertiary :type="detail.contest.deletedAt ? 'success' : 'error'">
                    {{ detail.contest.deletedAt ? t('admin.restore') : t('admin.delete') }}
                  </n-button>
                </template>
                {{ detail.contest.deletedAt ? t('admin.contests.restoreConfirm') : t('admin.contests.deleteConfirm') }}
              </n-popconfirm>
            </n-space>
          </div>
        </n-card>

        <n-card :bordered="false" class="stacked-card">
          <n-tabs v-model:value="activeTab" type="line" animated @update:value="handleTabUpdate">
            <n-tab-pane name="problems" :tab="t('common.problem')">
              <n-data-table :columns="columns" :data="detail.problems" :bordered="false" :scroll-x="520">
                <template #empty>
                  <n-empty :description="t('contests.problemsHidden')" />
                </template>
              </n-data-table>
            </n-tab-pane>
            <n-tab-pane name="scoreboard" :tab="t('contests.scoreboard')">
              <n-space justify="end" class="table-toolbar">
                <n-button
                  size="small"
                  secondary
                  :type="!showFullScoreboard ? 'primary' : 'default'"
                  @click="loadScoreboard(false)"
                >
                  {{ t('contests.publicBoard') }}
                </n-button>
                <n-button
                  v-if="canViewFullScoreboard"
                  size="small"
                  secondary
                  :type="showFullScoreboard ? 'primary' : 'default'"
                  @click="loadScoreboard(true)"
                >
                  {{ t('contests.realBoard') }}
                </n-button>
              </n-space>
              <n-alert v-if="scoreError" type="warning" class="card-alert">
                {{ scoreError }}
              </n-alert>
              <n-alert
                v-else-if="scoreboard?.contest.frozen && scoreboard.contest.mode === 'public'"
                type="warning"
                class="card-alert"
              >
                {{ t('contests.frozen') }}
              </n-alert>
              <n-alert
                v-else-if="scoreboard?.contest.mode === 'full'"
                type="info"
                class="card-alert"
              >
                {{ t('contests.fullBoard') }}
              </n-alert>
              <n-data-table
                remote
                :columns="scoreboardColumns"
                :data="scoreboard?.rows ?? []"
                :bordered="false"
                :loading="scoreboardLoading"
                :scroll-x="Math.max(620, 400 + (detail?.problems.length ?? 0) * 110)"
                :pagination="{
                  page: scoreboardPage,
                  pageSize: scoreboardPageSize,
                  itemCount: scoreboard?.total ?? 0,
                  showSizePicker: true,
                  pageSizes: [...PAGE_SIZE_OPTIONS],
                  onUpdatePage: changeScoreboardPage,
                  onUpdatePageSize: changeScoreboardPageSize
                }"
              />
            </n-tab-pane>
            <n-tab-pane name="submissions" :tab="t('common.submissions')">
              <n-data-table
                remote
                :columns="submissionColumns"
                :data="submissions"
                :bordered="false"
                :loading="submissionsLoading"
                :scroll-x="960"
                :pagination="{
                  page: submissionsPage,
                  pageSize: submissionsPageSize,
                  itemCount: submissionsTotal,
                  showSizePicker: true,
                  pageSizes: [...PAGE_SIZE_OPTIONS],
                  onUpdatePage: changeSubmissionsPage,
                  onUpdatePageSize: changeSubmissionsPageSize
                }"
              />
            </n-tab-pane>
          </n-tabs>
        </n-card>
      </template>
      <p v-else-if="error" class="form-error">{{ error }}</p>
    </n-spin>
  </main>
</template>
