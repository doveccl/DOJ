<script setup lang="ts">
import { NAlert, NCard, NDataTable, NSpin, NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { computed, h, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
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
const { t } = useI18n()

const columns = computed<DataTableColumns<AssignmentProblem>>(() => [
  { title: t('common.id'), key: 'id', width: 96 },
  {
    title: t('common.problem'),
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
  { title: t('contests.score'), key: 'score' }
])

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
          <p>{{ detail.assignment.description || t('assignments.fallback') }}</p>
        </section>

        <n-card :bordered="false" class="stacked-card">
          <div class="meta-row">
            <n-tag :bordered="false" :type="detail.assignment.allowLate ? 'warning' : 'default'">
              {{
                detail.assignment.allowLate
                  ? t('assignments.lateAllowed')
                  : t('assignments.lateClosed')
              }}
            </n-tag>
            <span class="muted">
              {{ t('assignments.duePrefix') }}
              {{
                detail.assignment.dueAt ? new Date(detail.assignment.dueAt).toLocaleString() : '-'
              }}
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
        {{ t('assignments.signIn') }}
      </n-alert>
      <n-alert v-else-if="error" type="error" class="page-alert">
        {{ error }}
      </n-alert>
    </n-spin>
  </main>
</template>
