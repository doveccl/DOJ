<script setup lang="ts">
import {
  NAlert,
  NButton,
  NCard,
  NCheckbox,
  NDataTable,
  NDatePicker,
  NForm,
  NFormItem,
  NInput,
  NSelect,
  NSpace,
  NTag
} from 'naive-ui'
import type { DataTableColumns, SelectOption } from 'naive-ui'
import { computed, h, onMounted, reactive, ref, watch } from 'vue'
import { apiFetch } from '../api'
import { useAuthStore } from '../stores/auth'

interface AssignmentRow {
  id: number
  title: string
  description: string
  dueAt: string | null
  allowLate: boolean
  aiCoachingEnabled: boolean
  createdAt: string
}

interface GroupRow {
  id: number
  key: string
  name: string
}

interface ProblemRow {
  id: number
  title: string
}

interface AssignmentReportProblem {
  id: number
  title: string
  score: number
}

interface AssignmentReportRow {
  userId: number
  userName: string
  email: string
  solved: number
  submitted: number
  problems: Record<
    string,
    {
      status: string
      attempts: number
      bestSubmissionId: number | null
      lastSubmissionId: number | null
      updatedAt: string | null
    }
  >
}

interface AssignmentReport {
  assignment: AssignmentRow
  problems: AssignmentReportProblem[]
  rows: AssignmentReportRow[]
}

const auth = useAuthStore()
const canManage = computed(() => auth.user?.groups.includes('admin') ?? false)
const loading = ref(true)
const saving = ref(false)
const reportLoading = ref(false)
const error = ref('')
const assignments = ref<AssignmentRow[]>([])
const report = ref<AssignmentReport | null>(null)
const groupOptions = ref<SelectOption[]>([])
const problemOptions = ref<SelectOption[]>([])
const form = reactive({
  title: '',
  description: '',
  groupIds: [] as number[],
  problemIds: [] as number[],
  dueAt: null as number | null,
  allowLate: false,
  aiCoachingEnabled: true
})

const columns: DataTableColumns<AssignmentRow> = [
  { title: 'Title', key: 'title' },
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
      return h(
        NTag,
        { bordered: false, type: row.aiCoachingEnabled ? 'success' : 'default' },
        () => (row.aiCoachingEnabled ? 'on' : 'off')
      )
    }
  },
  {
    title: 'Action',
    key: 'action',
    width: 120,
    render(row) {
      return h(NButton, { size: 'small', onClick: () => loadReport(row.id) }, () => 'Report')
    }
  }
]

const reportColumns = computed<DataTableColumns<AssignmentReportRow>>(() => [
  { title: 'Student', key: 'userName', minWidth: 140 },
  { title: 'Solved', key: 'solved', width: 96 },
  { title: 'Submitted', key: 'submitted', width: 110 },
  ...(report.value?.problems.map((problem) => ({
    title: String(problem.id),
    key: String(problem.id),
    width: 120,
    render(row: AssignmentReportRow) {
      const cell = row.problems[String(problem.id)]
      if (!cell?.attempts) return '-'
      const type =
        cell.status === 'AC' ? 'success' : cell.status === 'WAITING' ? 'default' : 'warning'
      return h(NTag, { bordered: false, type }, () => `${cell.status} (${cell.attempts})`)
    }
  })) ?? [])
])

async function loadData() {
  loading.value = true
  error.value = ''
  try {
    const [assignmentData, groupData, problemData] = await Promise.all([
      apiFetch<{ list: AssignmentRow[] }>('/api/assignments'),
      apiFetch<{ list: GroupRow[] }>('/api/groups'),
      apiFetch<{ list: ProblemRow[] }>('/api/problems')
    ])
    assignments.value = assignmentData.list
    groupOptions.value = groupData.list.map((group) => ({
      label: `${group.name} (${group.key})`,
      value: group.id
    }))
    problemOptions.value = problemData.list.map((problem) => ({
      label: problem.title,
      value: problem.id
    }))
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

async function createAssignment() {
  saving.value = true
  error.value = ''
  try {
    await apiFetch('/api/assignments', {
      method: 'POST',
      body: JSON.stringify({
        title: form.title,
        description: form.description,
        groupIds: form.groupIds,
        dueAt: form.dueAt ? new Date(form.dueAt).toISOString() : undefined,
        allowLate: form.allowLate,
        aiCoachingEnabled: form.aiCoachingEnabled,
        problems: form.problemIds.map((problemId) => ({ problemId, score: 100 }))
      })
    })
    form.title = ''
    form.description = ''
    form.groupIds = []
    form.problemIds = []
    form.dueAt = null
    form.allowLate = false
    form.aiCoachingEnabled = true
    await loadData()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

async function loadReport(id: number) {
  reportLoading.value = true
  error.value = ''
  try {
    report.value = await apiFetch<AssignmentReport>(`/api/assignments/${id}/report`)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    reportLoading.value = false
  }
}

watch(canManage, (allowed) => {
  if (allowed) loadData()
})

onMounted(() => {
  if (canManage.value) {
    loadData()
  } else {
    loading.value = false
  }
})
</script>

<template>
  <main class="page">
    <section class="page-header">
      <h1>Assignments</h1>
      <p>Publish problem sets to groups with optional deadlines.</p>
    </section>

    <n-alert v-if="!canManage" type="warning" class="page-alert">
      Admin group is required.
    </n-alert>

    <n-alert v-if="error" type="error" class="page-alert">
      {{ error }}
    </n-alert>

    <section v-if="canManage" class="admin-layout">
      <n-card title="Create assignment" :bordered="false">
        <n-form :model="form" label-placement="top">
          <n-form-item label="Title">
            <n-input v-model:value="form.title" placeholder="Week 1 Practice" />
          </n-form-item>
          <n-form-item label="Groups">
            <n-select
              v-model:value="form.groupIds"
              multiple
              filterable
              :options="groupOptions"
              placeholder="Select groups"
            />
          </n-form-item>
          <n-form-item label="Problems">
            <n-select
              v-model:value="form.problemIds"
              multiple
              filterable
              :options="problemOptions"
              placeholder="Select problems"
            />
          </n-form-item>
          <n-form-item label="Due at">
            <n-date-picker
              v-model:value="form.dueAt"
              type="datetime"
              clearable
              class="full-width"
            />
          </n-form-item>
          <n-form-item label="Description">
            <n-input
              v-model:value="form.description"
              type="textarea"
              placeholder="Optional notes"
              :autosize="{ minRows: 3, maxRows: 5 }"
            />
          </n-form-item>
          <n-space vertical>
            <n-checkbox v-model:checked="form.allowLate">Allow late submissions</n-checkbox>
            <n-checkbox v-model:checked="form.aiCoachingEnabled">Enable AI coaching</n-checkbox>
          </n-space>
          <n-space justify="end" class="form-actions">
            <n-button
              type="primary"
              :loading="saving"
              :disabled="!form.title || !form.groupIds.length || !form.problemIds.length"
              @click="createAssignment"
            >
              Create
            </n-button>
          </n-space>
        </n-form>
      </n-card>

      <n-data-table
        :columns="columns"
        :data="assignments"
        :bordered="false"
        :loading="loading"
        class="admin-table"
      />
    </section>

    <n-card
      v-if="canManage && report"
      :title="`Report: ${report.assignment.title}`"
      :bordered="false"
      class="stacked-card"
    >
      <n-data-table
        :columns="reportColumns"
        :data="report.rows"
        :bordered="false"
        :loading="reportLoading"
      />
    </n-card>
  </main>
</template>
