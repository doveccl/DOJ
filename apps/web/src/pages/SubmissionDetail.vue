<script setup lang="ts">
import { NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { apiFetch, isUnauthorized, openApiWebSocket } from '../api'
import MarkdownView from '../components/MarkdownView.vue'
import { useAuthStore } from '../stores/auth'

interface Submission {
  id: number
  languageId: string
  code?: string | null
  status: string | null
  displayStatus?: string
  timeMs: number | null
  memoryBytes: number | null
  score: number | null
  message: string | null
  contestId: number | null
  assignmentId: number | null
  public: boolean
  createdAt: string
  updatedAt: string
  canCoach: boolean
  judgeProgress: JudgeProgress | null
  cases: SubmissionCase[]
  problem: { id: number; title: string } | null
  user: { id: number; name: string }
}

interface JudgeProgress {
  phase: string
  message: string
  completedCases: number
  totalCases: number
  currentCase?: number
  caseNo?: number
  status?: string
  timeMs?: number
  memoryBytes?: number
  score?: number
}

interface SubmissionCase {
  caseIndex?: number
  caseNo?: number
  status: string
  timeMs: number
  memoryBytes: number
  score: number
  message: string | null
}

interface CoachingSession {
  summary: string
  hints: string[]
  nextSteps: string[]
}

const route = useRoute()
const loading = ref(true)
const coachingLoading = ref(false)
const error = ref('')
const requireSignIn = ref(false)
const coaching = ref('')
const submission = ref<Submission | null>(null)
const casePage = ref(1)
const casePageSize = 50
const { t } = useI18n()
const auth = useAuthStore()
let socket: WebSocket | null = null

const statusType: Record<string, 'success' | 'warning' | 'error' | 'info'> = {
  AC: 'success',
  WAITING: 'info',
  JUDGING: 'info',
  CE: 'warning',
  PE: 'warning',
  WA: 'error',
  RE: 'error',
  TLE: 'error',
  MLE: 'error',
  OLE: 'error',
  SE: 'error'
}

const canCoach = computed(() => {
  return submission.value?.canCoach ?? false
})
const progressPercent = computed(() => {
  const progress = submission.value?.judgeProgress
  if (!progress?.totalCases) return 0
  const percent = Math.round(((progress.completedCases ?? 0) / progress.totalCases) * 100)
  return progress.phase === 'finished' ? 100 : Math.min(99, percent)
})
const sourceMarkdown = computed(() => {
  const code = submission.value?.code ?? ''
  if (!code) return ''
  const fence = code.includes('```') ? '~~~~' : '```'
  return `${fence}${submission.value?.languageId}\n${code}\n${fence}`
})
const visibleCases = computed(() => {
  const start = (casePage.value - 1) * casePageSize
  return submission.value?.cases.slice(start, start + casePageSize) ?? []
})
const hasCroppedResult = computed(() => {
  return submission.value?.status === null && submission.value.displayStatus !== null
})

const caseColumns = computed<DataTableColumns<SubmissionCase>>(() => [
  {
    title: '#',
    key: 'caseNo',
    width: 72,
    render(row) {
      return row.caseNo ?? row.caseIndex ?? '-'
    }
  },
  {
    title: t('common.status'),
    key: 'status',
    width: 120,
    render(row) {
      return h(
        NTag,
        { bordered: false, type: statusType[row.status] ?? 'default' },
        () => row.status
      )
    }
  },
  { title: t('common.time'), key: 'timeMs', width: 120 },
  {
    title: t('common.memory'),
    key: 'memoryBytes',
    width: 140,
    render(row) {
      return `${Math.round(row.memoryBytes / 1024)} KB`
    }
  },
  { title: t('submissions.score'), key: 'score', width: 90 },
  {
    title: t('common.message'),
    key: 'message',
    ellipsis: {
      tooltip: true
    }
  }
])

onMounted(async () => {
  await load()
  connectProgressSocket()
})

onUnmounted(() => {
  socket?.close()
})

watch(
  () => auth.signedIn,
  async () => {
    socket?.close()
    socket = null
    await load()
    connectProgressSocket()
  }
)

async function load(showLoading = true) {
  if (showLoading) loading.value = true
  try {
    submission.value = await apiFetch<Submission>(`/api/submissions/${route.params.id}`)
    requireSignIn.value = false
    error.value = ''
  } catch (caught) {
    if (isUnauthorized(caught)) {
      requireSignIn.value = true
      submission.value = null
    } else {
      error.value = caught instanceof Error ? caught.message : String(caught)
    }
  } finally {
    loading.value = false
  }
}

function connectProgressSocket() {
  socket = openApiWebSocket()
  if (!socket) return
  socket.addEventListener('open', () => {
    socket?.send(
      JSON.stringify({
        type: 'subscribe-submission',
        submissionId: Number(route.params.id)
      })
    )
  })
  socket.addEventListener('message', (event) => {
    const message = parseWsMessage(event.data)
    if (!message) return
    if (message.type === 'ping') {
      socket?.send(JSON.stringify({ type: 'pong' }))
      return
    }
    if (message.type === 'submission-progress' && submission.value) {
      submission.value.judgeProgress = message.progress
      if (submission.value.status === 'WAITING') submission.value.status = 'JUDGING'
      if (message.progress.status && message.progress.caseNo) {
        upsertProgressCase(message.progress)
      }
      return
    }
    if (message.type === 'submission-result') {
      submission.value = message.result
    }
  })
}

function parseWsMessage(raw: string) {
  try {
    return JSON.parse(raw) as
      | { type: 'ping' }
      | { type: 'submission-progress'; submissionId: number; progress: JudgeProgress }
      | { type: 'submission-result'; submissionId: number; result: Submission }
  } catch {
    return null
  }
}

function upsertProgressCase(progress: JudgeProgress) {
  if (!submission.value || !progress.caseNo || !progress.status) return
  const existing = submission.value.cases.find((item) => item.caseNo === progress.caseNo)
  const patch = {
    caseNo: progress.caseNo,
    status: progress.status,
    timeMs: progress.timeMs ?? 0,
    memoryBytes: progress.memoryBytes ?? 0,
    score: progress.score ?? 0,
    message: progress.message ?? null
  }
  if (existing) Object.assign(existing, patch)
  else {
    submission.value.cases.push(patch)
    submission.value.cases.sort((left, right) => (left.caseNo ?? 0) - (right.caseNo ?? 0))
  }
}

async function getCoaching() {
  if (!submission.value) return

  coachingLoading.value = true
  error.value = ''
  try {
    const session = await apiFetch<CoachingSession>(
      `/api/submissions/${submission.value.id}/coach`,
      {
        method: 'POST'
      }
    )
    coaching.value = coachingToMarkdown(session)
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : String(caught)
  } finally {
    coachingLoading.value = false
  }
}

function coachingToMarkdown(session: CoachingSession) {
  const hints = session.hints.length
    ? session.hints.map((item) => `- ${item}`).join('\n')
    : '- 暂无额外提示。'
  const nextSteps = session.nextSteps.length
    ? session.nextSteps.map((item) => `- ${item}`).join('\n')
    : '- 结合样例和边界条件继续排查。'
  return [
    `### ${t('submissions.coachingSummary')}`,
    session.summary,
    `### ${t('submissions.coachingHints')}`,
    hints,
    `### ${t('submissions.coachingNextSteps')}`,
    nextSteps
  ].join('\n\n')
}

function formatDate(value: string) {
  return new Date(value).toLocaleString()
}
</script>

<template>
  <main class="page">
    <n-spin :show="loading">
      <section v-if="submission" class="submission-layout">
        <div>
          <n-card :bordered="false" class="submission-main-card">
            <template #header>
              <div class="submission-title-row">
                <span>{{ t('submissions.detailTitle') }} #{{ submission.id }}</span>
                <n-tag
                  :bordered="false"
                  :type="
                    statusType[submission.displayStatus ?? submission.status ?? ''] ?? 'default'
                  "
                >
                  {{ submission.displayStatus ?? submission.status ?? '-' }}
                </n-tag>
              </div>
            </template>
            <div class="submission-summary-grid">
              <div class="summary-item">
                <span>{{ t('submissions.score') }}</span>
                <strong>{{ submission.score ?? '-' }}</strong>
              </div>
              <div class="summary-item">
                <span>{{ t('common.language') }}</span>
                <strong>{{ submission.languageId }}</strong>
              </div>
              <div class="summary-item">
                <span>{{ t('common.time') }}</span>
                <strong>{{ submission.timeMs === null ? '-' : `${submission.timeMs} ms` }}</strong>
              </div>
              <div class="summary-item">
                <span>{{ t('common.memory') }}</span>
                <strong>{{
                  submission.memoryBytes === null
                    ? '-'
                    : `${Math.round(submission.memoryBytes / 1024)} KB`
                }}</strong>
              </div>
              <div class="summary-item wide">
                <span>{{ t('common.problem') }}</span>
                <strong>{{
                  submission.problem?.title ?? t('submissions.problemRestricted')
                }}</strong>
              </div>
              <div class="summary-item">
                <span>{{ t('common.user') }}</span>
                <strong>{{ submission.user.name }}</strong>
              </div>
              <div class="summary-item">
                <span>{{ t('submissions.createdAt') }}</span>
                <strong>{{ formatDate(submission.createdAt) }}</strong>
              </div>
            </div>
            <p v-if="hasCroppedResult" class="muted cropped-hint">
              {{ t('submissions.resultRestricted') }}
            </p>
          </n-card>
          <n-card
            v-if="submission.judgeProgress"
            :title="t('submissions.progress')"
            :bordered="false"
            class="stacked-card"
          >
            <div class="progress-stack">
              <n-progress
                type="line"
                :percentage="progressPercent"
                :indicator-placement="'inside'"
                processing
              />
              <p class="muted">
                {{ submission.judgeProgress.message }}
                <span v-if="submission.judgeProgress.totalCases">
                  · {{ submission.judgeProgress.completedCases }}/{{
                    submission.judgeProgress.totalCases
                  }}
                </span>
                <span v-if="submission.judgeProgress.caseNo">
                  · {{ t('submissions.currentCase') }} #{{ submission.judgeProgress.caseNo }}
                </span>
                <span v-if="submission.judgeProgress.status">
                  · {{ submission.judgeProgress.status }}
                </span>
              </p>
            </div>
          </n-card>
          <n-card
            v-if="submission.code"
            :title="t('submissions.source')"
            :bordered="false"
            class="stacked-card"
          >
            <markdown-view :source="sourceMarkdown" class="source-markdown" />
          </n-card>
          <n-card v-else :title="t('submissions.source')" :bordered="false" class="stacked-card">
            <p class="muted">
              {{ t('submissions.sourcePrivate') }}
            </p>
          </n-card>
          <n-card
            v-if="submission.message"
            :title="t('submissions.judgeMessage')"
            :bordered="false"
            class="stacked-card"
          >
            <pre class="code-block">{{ submission.message }}</pre>
          </n-card>
          <n-card
            v-if="submission.cases?.length"
            :title="t('submissions.testCases')"
            :bordered="false"
            class="stacked-card"
          >
            <n-data-table
              :columns="caseColumns"
              :data="visibleCases"
              :bordered="false"
              :scroll-x="720"
            />
            <n-pagination
              v-if="submission.cases.length > casePageSize"
              v-model:page="casePage"
              :page-size="casePageSize"
              :item-count="submission.cases.length"
              class="case-pagination"
            />
          </n-card>
        </div>
        <aside class="submission-side">
          <n-card :bordered="false">
            <div class="submission-meta-list">
              <div>
                <span>{{ t('submissions.contest') }}</span>
                <strong>{{
                  submission.contestId ? `#${submission.contestId}` : t('submissions.no')
                }}</strong>
              </div>
              <div>
                <span>{{ t('submissions.assignment') }}</span>
                <strong>{{
                  submission.assignmentId ? `#${submission.assignmentId}` : t('submissions.no')
                }}</strong>
              </div>
              <div>
                <span>{{ t('submissions.publicCode') }}</span>
                <strong>{{
                  submission.public ? t('submissions.yes') : t('submissions.no')
                }}</strong>
              </div>
            </div>
          </n-card>
          <n-card :title="t('submissions.aiCoaching')" :bordered="false" class="stacked-card">
            <p v-if="!canCoach" class="muted">
              {{ t('submissions.coachingUnavailable') }}
            </p>
            <p v-if="error" class="form-error">{{ error }}</p>
            <n-button
              type="primary"
              :disabled="!canCoach"
              :loading="coachingLoading"
              @click="getCoaching"
            >
              {{ t('submissions.getCoaching') }}
            </n-button>
            <markdown-view v-if="coaching" :source="coaching" class="coaching-output" />
          </n-card>
        </aside>
      </section>
      <sign-in-required v-else-if="requireSignIn" />
      <p v-else-if="error" class="form-error">{{ error }}</p>
    </n-spin>
  </main>
</template>

<style scoped lang="scss">
.submission-main-card,
.submission-side > .n-card {
  min-width: 0;
}

.submission-title-row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  font-size: 18px;
  font-weight: 700;
}

.submission-summary-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.summary-item,
.submission-meta-list > div {
  display: grid;
  gap: 4px;
  padding: 10px 12px;
  border-radius: var(--radius-md);
  background: color-mix(in srgb, var(--surface-bg) 94%, var(--text-color) 6%);
}

.summary-item.wide {
  grid-column: 1 / -1;
}

.summary-item span,
.submission-meta-list span {
  color: var(--muted-color);
  font-size: 12px;
}

.summary-item strong,
.submission-meta-list strong {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.submission-side {
  display: grid;
  gap: 16px;
}

.submission-meta-list {
  display: grid;
  gap: 10px;
}

.progress-stack {
  display: grid;
  gap: 10px;
}

.cropped-hint,
.case-pagination {
  margin-top: 12px;
}

@media (max-width: 640px) {
  .submission-summary-grid {
    grid-template-columns: 1fr;
  }
}
</style>
