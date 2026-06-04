<script setup lang="ts">
import { NCard, NDataTable, NSpin, NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { h, onMounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { apiFetch } from '../api'

interface Contest {
  id: number
  title: string
  description: string
  type: 'OI' | 'ICPC'
  startAt: string
  endAt: string
}

interface ContestProblem {
  id: number
  key: string
  title: string
  score: number
}

interface ContestDetail {
  contest: Contest
  problems: ContestProblem[]
}

const route = useRoute()
const loading = ref(true)
const error = ref('')
const detail = ref<ContestDetail | null>(null)

const columns: DataTableColumns<ContestProblem> = [
  { title: 'Key', key: 'key', width: 90 },
  {
    title: 'Problem',
    key: 'title',
    render(row) {
      return h(
        RouterLink,
        {
          to: `/problems/${row.id}?contestId=${detail.value?.contest.id}`,
          class: 'table-link'
        },
        () => row.title
      )
    }
  },
  { title: 'Score', key: 'score', width: 110 }
]

onMounted(async () => {
  try {
    detail.value = await apiFetch<ContestDetail>(`/api/contests/${route.params.id}`)
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
      <template v-if="detail">
        <section class="page-header">
          <h1>{{ detail.contest.title }}</h1>
          <p>{{ detail.contest.description || 'Contest problem set.' }}</p>
        </section>

        <n-card :bordered="false" class="stacked-card">
          <div class="meta-row">
            <n-tag :bordered="false">{{ detail.contest.type }}</n-tag>
            <span class="muted">{{ new Date(detail.contest.startAt).toLocaleString() }}</span>
            <span class="muted">to {{ new Date(detail.contest.endAt).toLocaleString() }}</span>
          </div>
        </n-card>

        <n-data-table
          :columns="columns"
          :data="detail.problems"
          :bordered="false"
          class="stacked-card"
        />
      </template>
      <p v-else-if="error" class="form-error">{{ error }}</p>
    </n-spin>
  </main>
</template>
