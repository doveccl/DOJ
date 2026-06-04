<script setup lang="ts">
import { NDataTable, NEmpty } from 'naive-ui'
import { onMounted, ref } from 'vue'

interface ProblemRow {
  id: string
  title: string
  tags: string[]
  solvedCount: number
}

const columns = [
  { title: 'Title', key: 'title' },
  { title: 'Tags', key: 'tags' },
  { title: 'Solved', key: 'solvedCount' }
]

const loading = ref(true)
const problems = ref<ProblemRow[]>([])

onMounted(async () => {
  try {
    const response = await fetch('/api/problems')
    const data = (await response.json()) as { list: ProblemRow[] }
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
      <h1>Problems</h1>
    </section>
    <n-data-table :columns="columns" :data="problems" :bordered="false" :loading="loading">
      <template #empty>
        <n-empty description="No problems yet" />
      </template>
    </n-data-table>
  </main>
</template>
