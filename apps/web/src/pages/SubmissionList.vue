<script setup lang="ts">
import { NTag } from 'naive-ui'
import { CloseOutline, SearchOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import {
  apiFetch,
  DEFAULT_PAGE_SIZE,
  getItems,
  openApiWebSocket,
  PAGE_SIZE_OPTIONS,
  type Paged
} from '../api'

interface SubmissionRow {
  id: number
  languageId: string
  status: string | null
  displayStatus: string
  timeMs: number | null
  memoryBytes: number | null
  score: number | null
  public: boolean
  contestId: number | null
  assignmentId: number | null
  createdAt: string
  updatedAt: string
  user: { id: number; name: string; avatarUrl?: string }
  problem: { id: number; title: string } | null
}

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

const { t } = useI18n()
const router = useRouter()

const languageNames = ref<Record<string, string>>({})
const languageOptions = computed(() =>
  Object.entries(languageNames.value).map(([value, label]) => ({ label, value }))
)
const statusOptions = [
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
].map((status) => ({ label: status, value: status }))
const filters = ref({
  problemId: '',
  userId: '',
  languageId: null as string | null,
  status: null as string | null,
  contestId: '',
  assignmentId: ''
})

const columns = computed(() => [
  {
    title: t('common.status'),
    key: 'status',
    render(row: SubmissionRow) {
      const displayStatus = row.displayStatus ?? row.status ?? '-'
      return h(NTag, { bordered: false, type: statusType[displayStatus] ?? 'default' }, () => displayStatus)
    }
  },
  {
    title: t('common.id'),
    key: 'id',
    width: 84,
    render(row: SubmissionRow) {
      return h(RouterLink, { to: `/submissions/${row.id}`, class: 'table-link' }, () =>
        String(row.id)
      )
    }
  },
  {
    title: t('common.problem'),
    key: 'problemTitle',
    minWidth: 220,
    render(row: SubmissionRow) {
      if (!row.problem) return '-'
      return h(RouterLink, { to: `/problems/${row.problem.id}`, class: 'table-link' }, () => row.problem?.title ?? '-')
    }
  },
  {
    title: t('common.user'),
    key: 'userName',
    minWidth: 140,
    render(row: SubmissionRow) {
      return row.user.name
    }
  },
  {
    title: t('common.language'),
    key: 'languageId',
    render(row: SubmissionRow) {
      return languageNames.value[row.languageId] ?? row.languageId
    }
  },
  {
    title: t('common.time'),
    key: 'timeMs',
    width: 100,
    render(row: SubmissionRow) {
      return row.timeMs === null ? '-' : `${row.timeMs} ms`
    }
  },
  {
    title: t('common.memory'),
    key: 'memoryBytes',
    render(row: SubmissionRow) {
      return row.memoryBytes === null ? '-' : `${Math.round(row.memoryBytes / 1024)} KB`
    }
  },
  {
    title: t('submissions.score'),
    key: 'score',
    width: 90,
    render(row: SubmissionRow) {
      return row.score ?? '-'
    }
  },
  {
    title: t('submissions.createdAt'),
    key: 'createdAt',
    minWidth: 180,
    render(row: SubmissionRow) {
      return formatDate(row.createdAt)
    }
  }
])

const loading = ref(true)
const submissions = ref<SubmissionRow[]>([])
const page = ref(1)
const pageSize = ref(DEFAULT_PAGE_SIZE)
const total = ref(0)
let socket: WebSocket | null = null

async function loadSubmissions(showLoading = false) {
  if (showLoading) loading.value = true
  try {
    const params = new URLSearchParams({
      page: String(page.value),
      pageSize: String(pageSize.value)
    })
    for (const key of ['problemId', 'userId', 'contestId', 'assignmentId'] as const) {
      const value = filters.value[key].trim()
      if (value) params.set(key, value)
    }
    if (filters.value.languageId) params.set('languageId', filters.value.languageId)
    if (filters.value.status) params.set('status', filters.value.status)
    const data = await apiFetch<Paged<SubmissionRow>>(`/api/submissions?${params.toString()}`)
    submissions.value = getItems(data)
    total.value = data.total
  } finally {
    loading.value = false
  }
}

function changePage(nextPage: number) {
  page.value = nextPage
  loadSubmissions(true).then(subscribeCurrentPage)
}

function changePageSize(nextPageSize: number) {
  pageSize.value = nextPageSize
  page.value = 1
  loadSubmissions(true).then(subscribeCurrentPage)
}

function applyFilters() {
  page.value = 1
  loadSubmissions(true).then(subscribeCurrentPage)
}

function clearFilters() {
  filters.value = {
    problemId: '',
    userId: '',
    languageId: null,
    status: null,
    contestId: '',
    assignmentId: ''
  }
  applyFilters()
}

onMounted(async () => {
  try {
    const languages = await apiFetch<Array<{ id: string; name: string }>>('/api/languages')
    languageNames.value = Object.fromEntries(languages.map((l) => [l.id, l.name]))
  } catch {
    languageNames.value = {}
  }
  await loadSubmissions(true)
  connectFeedSocket()
})

onUnmounted(() => {
  socket?.close()
})

function connectFeedSocket() {
  socket = openApiWebSocket()
  if (!socket) return
  socket.addEventListener('open', subscribeCurrentPage)
  socket.addEventListener('message', (event) => {
    const message = parseWsMessage(event.data)
    if (!message) return
    if (message.type === 'ping') {
      socket?.send(JSON.stringify({ type: 'pong' }))
      return
    }
    if (message.type === 'submission-feed') {
      const index = submissions.value.findIndex((item) => item.id === message.item.id)
      if (index >= 0) submissions.value.splice(index, 1, { ...submissions.value[index], ...message.item })
    }
  })
}

function subscribeCurrentPage() {
  if (socket?.readyState !== WebSocket.OPEN) return
  socket.send(
    JSON.stringify({
      type: 'subscribe-feed',
      scope: 'page',
      submissionIds: submissions.value.map((item) => item.id)
    })
  )
}

function parseWsMessage(raw: string) {
  try {
    return JSON.parse(raw) as
      | { type: 'ping' }
      | { type: 'submission-feed'; item: SubmissionRow }
  } catch {
    return null
  }
}

function formatDate(value: string) {
  return new Date(value).toLocaleString()
}

function submissionRowProps(row: SubmissionRow) {
  return {
    class: 'clickable-table-row',
    onClick: () => router.push(`/submissions/${row.id}`)
  }
}
</script>

<template>
  <main class="page">
    <n-card :bordered="false">
      <div class="table-toolbar submission-toolbar">
        <div class="toolbar-fields submission-fields">
          <n-input v-model:value="filters.problemId" clearable :placeholder="t('submissions.problemId')" />
          <n-input v-model:value="filters.userId" clearable :placeholder="t('submissions.userId')" />
          <n-select
            v-model:value="filters.languageId"
            clearable
            filterable
            :options="languageOptions"
            :placeholder="t('common.language')"
          />
          <n-select
            v-model:value="filters.status"
            clearable
            :options="statusOptions"
            :placeholder="t('common.status')"
          />
          <n-input v-model:value="filters.contestId" clearable :placeholder="t('submissions.contestId')" />
          <n-input v-model:value="filters.assignmentId" clearable :placeholder="t('submissions.assignmentId')" />
        </div>
        <div class="toolbar-actions">
          <n-button secondary @click="clearFilters">
            <template #icon>
              <n-icon :component="CloseOutline" />
            </template>
            {{ t('problems.clear') }}
          </n-button>
          <n-button type="primary" @click="applyFilters">
            <template #icon>
              <n-icon :component="SearchOutline" />
            </template>
            {{ t('submissions.applyFilters') }}
          </n-button>
        </div>
      </div>
      <n-data-table
        :columns="columns"
        :data="submissions"
        :bordered="false"
        :loading="loading"
        :scroll-x="1120"
        :row-props="submissionRowProps"
        :pagination="{
          page,
          pageSize,
          itemCount: total,
          showSizePicker: true,
          pageSizes: [...PAGE_SIZE_OPTIONS],
          onUpdatePage: changePage,
          onUpdatePageSize: changePageSize
        }"
      />
    </n-card>
  </main>
</template>

<style scoped lang="scss">
.submission-fields {
  flex: 0 1 auto;
  display: grid;
  grid-template-columns: repeat(6, minmax(120px, 150px));
}

.submission-toolbar .toolbar-actions {
  margin-left: auto;
}

:deep(.clickable-table-row) {
  cursor: pointer;
}

@media (max-width: 980px) {
  .submission-fields {
    grid-template-columns: repeat(2, minmax(160px, 1fr));
  }
}

@media (max-width: 560px) {
  .submission-fields {
    grid-template-columns: 1fr;
  }
}
</style>

<style scoped lang="scss">
.submission-filters {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 0 12px;
  align-items: end;
}
</style>
