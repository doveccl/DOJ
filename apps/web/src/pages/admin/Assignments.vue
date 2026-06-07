<script setup lang="ts">
import {
  NAlert,
  NButton,
  NCard,
  NCheckbox,
  NDataTable,
  NDatePicker,
  NEmpty,
  NForm,
  NFormItem,
  NInput,
  NModal,
  NSelect,
  NSpace,
  NTag
} from 'naive-ui'
import type { DataTableColumns, SelectOption } from 'naive-ui'
import { computed, h, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../../api'
import { useAuthStore } from '../../stores/auth'

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
const { t } = useI18n()
const canManage = computed(() => auth.user?.groups.includes('admin') ?? false)
const loading = ref(true)
const saving = ref(false)
const reportLoading = ref(false)
const error = ref('')
const showCreateModal = ref(false)
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
const pagination = reactive({
  page: 1,
  pageSize: 50,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [20, 50, 100]
})

const columns = computed<DataTableColumns<AssignmentRow>>(() => [
  { title: t('common.title'), key: 'title' },
  {
    title: t('admin.assignments.due'),
    key: 'dueAt',
    render(row) {
      return row.dueAt ? new Date(row.dueAt).toLocaleString() : '-'
    }
  },
  {
    title: t('admin.assignments.late'),
    key: 'allowLate',
    render(row) {
      return h(NTag, { bordered: false, type: row.allowLate ? 'warning' : 'default' }, () =>
        row.allowLate ? t('admin.assignments.allowed') : t('admin.assignments.closed')
      )
    }
  },
  {
    title: t('admin.assignments.ai'),
    key: 'aiCoachingEnabled',
    render(row) {
      return h(
        NTag,
        { bordered: false, type: row.aiCoachingEnabled ? 'success' : 'default' },
        () => (row.aiCoachingEnabled ? t('admin.assignments.on') : t('admin.assignments.off'))
      )
    }
  },
  {
    title: t('admin.actions'),
    key: 'action',
    width: 120,
    render(row) {
      return h(NButton, { size: 'small', onClick: () => loadReport(row.id) }, () =>
        t('admin.assignments.report')
      )
    }
  }
])

const reportColumns = computed<DataTableColumns<AssignmentReportRow>>(() => [
  { title: t('admin.assignments.student'), key: 'userName', minWidth: 140 },
  { title: t('common.solved'), key: 'solved', width: 96 },
  { title: t('admin.assignments.submitted'), key: 'submitted', width: 110 },
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
    const [assignmentData, groupData, problems] = await Promise.all([
      apiFetch<{ list: AssignmentRow[]; total: number }>(
        `/api/assignments?page=${pagination.page}&pageSize=${pagination.pageSize}`
      ),
      apiFetch<{ list: GroupRow[] }>('/api/groups'),
      loadAllProblems()
    ])
    assignments.value = assignmentData.list
    pagination.itemCount = assignmentData.total
    groupOptions.value = groupData.list.map((group) => ({
      label: `${group.name} (${group.key})`,
      value: group.id
    }))
    problemOptions.value = problems.map((problem) => ({
      label: problem.title,
      value: problem.id
    }))
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

async function loadAllProblems() {
  const result: ProblemRow[] = []
  for (let page = 1; ; page += 1) {
    const data = await apiFetch<{ list: ProblemRow[]; total: number; pageSize: number }>(
      `/api/admin/problems?page=${page}&pageSize=100`
    )
    result.push(...data.list)
    if (result.length >= data.total || data.list.length < data.pageSize) return result
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  void loadData()
}

function handlePageSizeChange(pageSize: number) {
  pagination.pageSize = pageSize
  pagination.page = 1
  void loadData()
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
    showCreateModal.value = false
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
    <n-alert v-if="!canManage" type="warning" class="page-alert">
      {{ t('admin.requireAdmin') }}
    </n-alert>

    <n-alert v-if="error" type="error" class="page-alert">
      {{ error }}
    </n-alert>

    <n-card v-if="canManage" :bordered="false">
      <n-space justify="end" class="table-toolbar">
        <n-button type="primary" @click="showCreateModal = true">
          {{ t('admin.assignments.create') }}
        </n-button>
      </n-space>
      <n-empty
        v-if="!loading && !assignments.length"
        class="empty-state"
        :description="t('admin.assignments.empty')"
      >
        <template #extra>
          <n-button secondary size="small" @click="showCreateModal = true">
            {{ t('admin.assignments.create') }}
          </n-button>
        </template>
      </n-empty>
      <n-data-table
        v-else
        remote
        :columns="columns"
        :data="assignments"
        :bordered="false"
        :loading="loading"
        :pagination="pagination"
        class="admin-table"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>

    <n-modal
      v-model:show="showCreateModal"
      preset="card"
      :title="t('admin.assignments.create')"
      class="form-modal"
    >
      <n-form :model="form" label-placement="top">
        <n-form-item :label="t('common.title')">
          <n-input v-model:value="form.title" placeholder="Week 1 Practice" />
        </n-form-item>
        <n-form-item :label="t('nav.groups')">
          <n-select
            v-model:value="form.groupIds"
            multiple
            filterable
            :options="groupOptions"
            :placeholder="t('admin.assignments.selectGroups')"
          />
        </n-form-item>
        <n-form-item :label="t('nav.problems')">
          <n-select
            v-model:value="form.problemIds"
            multiple
            filterable
            :options="problemOptions"
            :placeholder="t('admin.assignments.selectProblems')"
          />
        </n-form-item>
        <n-form-item :label="t('admin.assignments.dueAt')">
          <n-date-picker v-model:value="form.dueAt" type="datetime" clearable class="full-width" />
        </n-form-item>
        <n-form-item :label="t('admin.description')">
          <n-input
            v-model:value="form.description"
            type="textarea"
            :placeholder="t('admin.optionalNotes')"
            :autosize="{ minRows: 3, maxRows: 5 }"
          />
        </n-form-item>
        <n-space vertical>
          <n-checkbox v-model:checked="form.allowLate">
            {{ t('admin.assignments.allowLate') }}
          </n-checkbox>
          <n-checkbox v-model:checked="form.aiCoachingEnabled">
            {{ t('admin.assignments.aiCoaching') }}
          </n-checkbox>
        </n-space>
        <n-space justify="end" class="form-actions">
          <n-button @click="showCreateModal = false">{{ t('admin.cancel') }}</n-button>
          <n-button
            type="primary"
            :loading="saving"
            :disabled="!form.title || !form.groupIds.length || !form.problemIds.length"
            @click="createAssignment"
          >
            {{ t('admin.create') }}
          </n-button>
        </n-space>
      </n-form>
    </n-modal>

    <n-card
      v-if="canManage && report"
      :title="`${t('admin.assignments.report')}: ${report.assignment.title}`"
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

<style scoped lang="scss">
.empty-state {
  padding: 48px 0;
}
</style>
