<script setup lang="ts">
import { NDataTable, NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { h, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { apiFetch } from '../api'

interface ContestRow {
  id: number
  title: string
  type: 'OI' | 'ICPC'
  startAt: string
  endAt: string
}

const loading = ref(true)
const contests = ref<ContestRow[]>([])

const columns: DataTableColumns<ContestRow> = [
  {
    title: 'Title',
    key: 'title',
    render(row) {
      return h(RouterLink, { to: `/contests/${row.id}`, class: 'table-link' }, () => row.title)
    }
  },
  {
    title: 'Type',
    key: 'type',
    width: 110,
    render(row) {
      return h(NTag, { bordered: false }, () => row.type)
    }
  },
  {
    title: 'Start',
    key: 'startAt',
    render(row) {
      return new Date(row.startAt).toLocaleString()
    }
  },
  {
    title: 'End',
    key: 'endAt',
    render(row) {
      return new Date(row.endAt).toLocaleString()
    }
  }
]

onMounted(async () => {
  try {
    const data = await apiFetch<{ list: ContestRow[] }>('/api/contests')
    contests.value = data.list
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <main class="page">
    <section class="page-header">
      <h1>Contests</h1>
    </section>
    <n-data-table :columns="columns" :data="contests" :bordered="false" :loading="loading" />
  </main>
</template>
