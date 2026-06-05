<script setup lang="ts">
import { NDataTable, NTag } from 'naive-ui'
import { computed, h, onMounted, onUnmounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../api'

interface SubmissionRow {
  id: number
  userId: number
  problemId: number
  languageId: string
  status: string
  timeMs: number
  memoryBytes: number
  message: string
  createdAt: string
}

const statusType: Record<string, 'success' | 'warning' | 'error' | 'info'> = {
  AC: 'success',
  WAITING: 'info',
  JUDGING: 'info',
  CE: 'warning',
  WA: 'error',
  RE: 'error',
  TLE: 'error',
  MLE: 'error',
  OLE: 'error',
  SE: 'error'
}

const { t } = useI18n()

const columns = computed(() => [
  {
    title: t('common.status'),
    key: 'status',
    render(row: SubmissionRow) {
      return h(
        NTag,
        { bordered: false, type: statusType[row.status] ?? 'default' },
        () => row.status
      )
    }
  },
  {
    title: t('common.id'),
    key: 'id',
    render(row: SubmissionRow) {
      return h(RouterLink, { to: `/submissions/${row.id}`, class: 'table-link' }, () =>
        String(row.id)
      )
    }
  },
  { title: t('common.language'), key: 'languageId' },
  { title: t('common.time'), key: 'timeMs' },
  {
    title: t('common.memory'),
    key: 'memoryBytes',
    render(row: SubmissionRow) {
      return `${Math.round(row.memoryBytes / 1024)} KB`
    }
  },
  {
    title: t('common.message'),
    key: 'message',
    ellipsis: {
      tooltip: true
    }
  }
])

const loading = ref(true)
const submissions = ref<SubmissionRow[]>([])
let timer: number | undefined

async function loadSubmissions(showLoading = false) {
  if (showLoading) loading.value = true
  try {
    const data = await apiFetch<{ list: SubmissionRow[] }>('/api/submissions')
    submissions.value = data.list
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await loadSubmissions(true)
  timer = window.setInterval(() => loadSubmissions(), 2500)
})

onUnmounted(() => {
  if (timer) window.clearInterval(timer)
})
</script>

<template>
  <main class="page">
    <section class="page-header">
      <h1>{{ t('submissions.title') }}</h1>
    </section>
    <n-data-table :columns="columns" :data="submissions" :bordered="false" :loading="loading" />
  </main>
</template>
