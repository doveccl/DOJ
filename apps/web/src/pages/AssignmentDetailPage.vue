<script setup lang="ts">
import { NAlert, NCard, NDataTable, NSpin, NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { h, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { apiFetch } from '../api'
import { useAuthStore } from '../stores/auth'

interface Assignment {
  id: number
  title: string
  description: string
  dueAt: string | null
  allowLate: boolean
  aiCoachingEnabled: boolean
}

interface AssignmentProblem {
  id: number
  title: string
  score: number
}

interface AssignmentDetail {
  assignment: Assignment
  problems: AssignmentProblem[]
}

const route = useRoute()
const auth = useAuthStore()
const loading = ref(true)
const error = ref('')
const detail = ref<AssignmentDetail | null>(null)

const columns: DataTableColumns<AssignmentProblem> = [
  { title: 'ID', key: 'id', width: 96 },
  {
    title: 'Problem',
    key: 'title',
    render(row) {
      return h(
        RouterLink,
        {
          to: `/problems/${row.id}?assignmentId=${detail.value?.assignment.id}`,
          class: 'table-link'
        },
        () => row.title
      )
    }
  },
  { title: 'Score', key: 'score' }
]

async function loadDetail() {
  loading.value = true
  error.value = ''
  try {
    detail.value = await apiFetch<AssignmentDetail>(`/api/my/assignments/${route.params.id}`)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

watch(
  () => auth.signedIn,
  (signedIn) => {
    if (signedIn) loadDetail()
  }
)

onMounted(() => {
  if (auth.signedIn) {
    loadDetail()
  } else {
    loading.value = false
  }
})
</script>

<template>
  <main class="page">
    <n-spin :show="loading">
      <template v-if="detail">
        <section class="page-header">
          <h1>{{ detail.assignment.title }}</h1>
          <p>{{ detail.assignment.description || 'Assigned problem set.' }}</p>
        </section>

        <n-card :bordered="false" class="stacked-card">
          <div class="meta-row">
            <n-tag :bordered="false" :type="detail.assignment.allowLate ? 'warning' : 'default'">
              {{ detail.assignment.allowLate ? 'late allowed' : 'late closed' }}
            </n-tag>
            <n-tag :bordered="false" :type="detail.assignment.aiCoachingEnabled ? 'success' : 'default'">
              {{ detail.assignment.aiCoachingEnabled ? 'AI on' : 'AI off' }}
            </n-tag>
            <span class="muted">
              Due {{ detail.assignment.dueAt ? new Date(detail.assignment.dueAt).toLocaleString() : '-' }}
            </span>
          </div>
        </n-card>

        <n-data-table
          :columns="columns"
          :data="detail.problems"
          :bordered="false"
          class="stacked-card"
        />
      </template>

      <n-alert v-else-if="!auth.signedIn" type="warning" class="page-alert">
        Sign in to view this assignment.
      </n-alert>
      <n-alert v-else-if="error" type="error" class="page-alert">
        {{ error }}
      </n-alert>
    </n-spin>
  </main>
</template>
