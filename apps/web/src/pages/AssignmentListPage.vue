<script setup lang="ts">
import { NAlert, NDataTable, NEmpty, NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { h, onMounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { apiFetch } from '../api'
import { useAuthStore } from '../stores/auth'

interface AssignmentRow {
  id: string
  title: string
  description: string
  dueAt: string | null
  allowLate: boolean
  aiCoachingEnabled: boolean
}

const auth = useAuthStore()
const loading = ref(true)
const error = ref('')
const assignments = ref<AssignmentRow[]>([])

const columns: DataTableColumns<AssignmentRow> = [
  {
    title: 'Title',
    key: 'title',
    render(row) {
      return h(RouterLink, { to: `/assignments/${row.id}`, class: 'table-link' }, () => row.title)
    }
  },
  {
    title: 'Due',
    key: 'dueAt',
    render(row) {
      return row.dueAt ? new Date(row.dueAt).toLocaleString() : '-'
    }
  },
  {
    title: 'Late',
    key: 'allowLate',
    render(row) {
      return h(NTag, { bordered: false, type: row.allowLate ? 'warning' : 'default' }, () =>
        row.allowLate ? 'allowed' : 'closed'
      )
    }
  },
  {
    title: 'AI',
    key: 'aiCoachingEnabled',
    render(row) {
      return h(NTag, { bordered: false, type: row.aiCoachingEnabled ? 'success' : 'default' }, () =>
        row.aiCoachingEnabled ? 'on' : 'off'
      )
    }
  }
]

async function loadAssignments() {
  loading.value = true
  error.value = ''
  try {
    const data = await apiFetch<{ list: AssignmentRow[] }>('/api/my/assignments')
    assignments.value = data.list
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

watch(
  () => auth.signedIn,
  (signedIn) => {
    if (signedIn) loadAssignments()
  }
)

onMounted(() => {
  if (auth.signedIn) {
    loadAssignments()
  } else {
    loading.value = false
  }
})
</script>

<template>
  <main class="page">
    <section class="page-header">
      <h1>Assignments</h1>
      <p>Your assigned problem sets.</p>
    </section>

    <n-alert v-if="!auth.signedIn" type="warning" class="page-alert">
      Sign in to view assignments.
    </n-alert>

    <n-alert v-if="error" type="error" class="page-alert">
      {{ error }}
    </n-alert>

    <n-data-table :columns="columns" :data="assignments" :bordered="false" :loading="loading">
      <template #empty>
        <n-empty description="No assignments yet" />
      </template>
    </n-data-table>
  </main>
</template>
