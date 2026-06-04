<script setup lang="ts">
import { NDataTable, NTag } from 'naive-ui'
import { h, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'

interface SubmissionRow {
  id: string
  userId: string
  problemId: string
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

const columns = [
  {
    title: 'Status',
    key: 'status',
    render(row: SubmissionRow) {
      return h(NTag, { bordered: false, type: statusType[row.status] ?? 'default' }, () => row.status)
    }
  },
  {
    title: 'ID',
    key: 'id',
    render(row: SubmissionRow) {
      return h(RouterLink, { to: `/submissions/${row.id}`, class: 'table-link' }, () =>
        row.id.slice(0, 8)
      )
    }
  },
  { title: 'Language', key: 'languageId' },
  { title: 'Time', key: 'timeMs' },
  {
    title: 'Memory',
    key: 'memoryBytes',
    render(row: SubmissionRow) {
      return `${Math.round(row.memoryBytes / 1024)} KB`
    }
  },
  {
    title: 'Message',
    key: 'message',
    ellipsis: {
      tooltip: true
    }
  }
]

const loading = ref(true)
const submissions = ref<SubmissionRow[]>([])

onMounted(async () => {
  try {
    const response = await fetch('/api/submissions')
    const data = (await response.json()) as { list: SubmissionRow[] }
    submissions.value = data.list
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <main class="page">
    <section class="page-header">
      <h1>Submissions</h1>
    </section>
    <n-data-table :columns="columns" :data="submissions" :bordered="false" :loading="loading" />
  </main>
</template>
