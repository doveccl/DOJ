<script setup lang="ts">
import { NDataTable } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../api'

interface RankRow {
  id: number
  name: string
  solvedCount: number
  submissionCount: number
  introduction: string
}

const loading = ref(true)
const rows = ref<RankRow[]>([])
const { t } = useI18n()

const columns = computed<DataTableColumns<RankRow>>(() => [
  {
    title: '#',
    key: 'rank',
    width: 72,
    render(_row, index) {
      return String(index + 1)
    }
  },
  { title: t('common.user'), key: 'name' },
  { title: t('common.solved'), key: 'solvedCount', width: 120 },
  { title: t('dashboard.submissions'), key: 'submissionCount', width: 140 },
  {
    title: t('rank.intro'),
    key: 'introduction',
    ellipsis: {
      tooltip: true
    }
  }
])

onMounted(async () => {
  try {
    const data = await apiFetch<{ list: RankRow[] }>('/api/rank')
    rows.value = data.list
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <main class="page">
    <section class="page-header">
      <h1>{{ t('rank.title') }}</h1>
    </section>
    <n-data-table :columns="columns" :data="rows" :bordered="false" :loading="loading" />
  </main>
</template>
