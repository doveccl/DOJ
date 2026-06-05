<script setup lang="ts">
import { NDataTable, NEmpty } from 'naive-ui'
import { computed, h, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../api'

interface ProblemRow {
  id: number
  title: string
  tags: string[]
  solvedCount: number
}

const { t } = useI18n()

const columns = computed(() => [
  { title: t('common.id'), key: 'id', width: 96 },
  {
    title: t('common.title'),
    key: 'title',
    render(row: ProblemRow) {
      return h(RouterLink, { to: `/problems/${row.id}`, class: 'table-link' }, () => row.title)
    }
  },
  { title: t('common.tags'), key: 'tags' },
  { title: t('common.solved'), key: 'solvedCount' }
])

const loading = ref(true)
const problems = ref<ProblemRow[]>([])

onMounted(async () => {
  try {
    const data = await apiFetch<{ list: ProblemRow[] }>('/api/problems')
    problems.value = data.list.map((item) => ({
      ...item,
      tags: item.tags.length ? item.tags : ['-']
    }))
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <main class="page">
    <section class="page-header">
      <h1>{{ t('problems.title') }}</h1>
    </section>
    <n-data-table :columns="columns" :data="problems" :bordered="false" :loading="loading">
      <template #empty>
        <n-empty :description="t('problems.empty')" />
      </template>
    </n-data-table>
  </main>
</template>
