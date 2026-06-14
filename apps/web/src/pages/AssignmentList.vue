<script setup lang="ts">
import { NButton, NIcon, NPopconfirm, NProgress, NSpace, NTag, NTooltip } from 'naive-ui'
import type { DataTableColumns, SelectOption } from 'naive-ui'
import type { Component } from 'vue'
import { AddOutline, CreateOutline, TrashOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { apiFetch, DEFAULT_PAGE_SIZE, getItems, PAGE_SIZE_OPTIONS, type Paged } from '../api'
import { useAuthStore } from '../stores/auth'

interface AssignmentRow {
  id: number
  title: string
  description: string
  endAt: string
  deletedAt?: string | null
  completed: number
  total: number
  assigned?: number
  problemCount?: number
}

interface AssignmentDetail {
  assignment: AssignmentRow
  groups: Array<{ id: number; name: string }>
  users: Array<{ id: number; name: string; email: string }>
  problems: Array<{ id: number | null; title: string | null }>
}

interface GroupRow {
  id: number
  name: string
}

interface UserRow {
  id: number
  name: string
  email: string
}

interface ProblemRow {
  id: number
  title: string
}

const auth = useAuthStore()
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const showFormModal = ref(false)
const editingId = ref<number | null>(null)
const editingEnded = ref(false)
const assignments = ref<AssignmentRow[]>([])
const groupOptions = ref<SelectOption[]>([])
const userOptions = ref<SelectOption[]>([])
const problemOptions = ref<SelectOption[]>([])
const { t } = useI18n()
const canManage = computed(() => auth.user?.admin ?? false)
const form = reactive({
  title: '',
  description: '',
  endAt: Date.now() + 7 * 24 * 60 * 60 * 1000,
  groupIds: [] as number[],
  userIds: [] as number[],
  problemIds: [] as number[]
})
const pagination = reactive({
  page: 1,
  pageSize: DEFAULT_PAGE_SIZE,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [...PAGE_SIZE_OPTIONS]
})

const modalTitle = computed(() => (editingId.value ? t('admin.assignments.edit') : t('admin.assignments.create')))

const columns = computed<DataTableColumns<AssignmentRow>>(() => [
  {
    title: t('common.title'),
    key: 'title',
    minWidth: 260,
    render(row) {
      const state = assignmentState(row)
      return h('div', { class: 'assignment-title-row' }, [
        h(RouterLink, { to: `/assignments/${row.id}`, class: 'table-link assignment-title' }, () => row.title),
        state ? h(NTag, { bordered: false, type: state.type, size: 'small' }, () => state.label) : null
      ])
    }
  },
  {
    title: t('assignments.due'),
    key: 'endAt',
    width: 220,
    render(row) {
      return h('span', { class: 'assignment-date' }, new Date(row.endAt).toLocaleString())
    }
  },
  {
    title: t('assignments.progress'),
    key: 'progress',
    width: 220,
    render(row) {
      return h('div', { class: 'assignment-progress' }, [
        h(NProgress, {
          type: 'line',
          percentage: assignmentProgressPercent(row),
          showIndicator: false,
          status: row.total > 0 && row.completed >= row.total ? 'success' : 'default'
        }),
        h('span', { class: 'assignment-progress-text' }, `${row.completed}/${row.total}`)
      ])
    }
  },
  ...(canManage.value
    ? [
        {
          title: t('admin.assignments.problems'),
          key: 'problemCount',
          width: 96,
          render(row: AssignmentRow) {
            return row.problemCount ?? '-'
          }
        },
        {
          title: t('admin.actions'),
          key: 'actions',
          width: 112,
          render(row: AssignmentRow) {
            return h(NSpace, { size: 8 }, () => [
              tooltipIconButton(CreateOutline, t('admin.edit'), () => openEdit(row.id)),
              h(
                NPopconfirm,
                { onPositiveClick: () => deleteAssignment(row) },
                {
                  trigger: () =>
                    tooltipIconButton(TrashOutline, t('admin.delete'), () => {}, { type: 'error' }),
                  default: () => t('admin.assignments.deleteConfirm')
                }
              )
            ])
          }
        }
      ]
    : [])
])

async function loadAssignments() {
  loading.value = true
  error.value = ''
  try {
    const params = new URLSearchParams({
      page: String(pagination.page),
      pageSize: String(pagination.pageSize)
    })
    const data = await apiFetch<Paged<AssignmentRow>>(
      `${canManage.value ? '/api/admin/assignments' : '/api/my/assignments'}?${params}`
    )
    assignments.value = getItems(data)
    pagination.itemCount = data.total
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

async function loadAdminOptions() {
  if (!canManage.value) return
  try {
    const [groups, users, problems] = await Promise.all([
      apiFetch<Paged<GroupRow>>('/api/admin/groups'),
      apiFetch<Paged<UserRow>>('/api/admin/users?pageSize=100'),
      apiFetch<Paged<ProblemRow>>('/api/problems?pageSize=100')
    ])
    groupOptions.value = getItems(groups).map((group) => ({
      label: group.name,
      value: group.id
    }))
    userOptions.value = getItems(users).map((user) => ({
      label: `${user.name} <${user.email}>`,
      value: user.id
    }))
    problemOptions.value = getItems(problems).map((problem) => ({
      label: `P${problem.id} ${problem.title}`,
      value: problem.id
    }))
  } catch {
    groupOptions.value = []
    userOptions.value = []
    problemOptions.value = []
  }
}

function resetForm() {
  editingId.value = null
  editingEnded.value = false
  form.title = ''
  form.description = ''
  form.endAt = Date.now() + 7 * 24 * 60 * 60 * 1000
  form.groupIds = []
  form.userIds = []
  form.problemIds = []
}

function openCreate() {
  resetForm()
  showFormModal.value = true
}

async function openEdit(id: number) {
  saving.value = true
  error.value = ''
  try {
    const detail = await apiFetch<AssignmentDetail>(`/api/admin/assignments/${id}`)
    editingId.value = id
    editingEnded.value = new Date() >= new Date(detail.assignment.endAt)
    form.title = detail.assignment.title
    form.description = detail.assignment.description
    form.endAt = new Date(detail.assignment.endAt).getTime()
    form.groupIds = detail.groups.map((group) => group.id)
    form.userIds = detail.users.map((user) => user.id)
    form.problemIds = detail.problems.map((problem) => problem.id).filter((id): id is number => id !== null)
    showFormModal.value = true
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

async function saveAssignment() {
  saving.value = true
  error.value = ''
  try {
    const body = editingEnded.value
      ? {
          title: form.title,
          description: form.description
        }
      : {
          title: form.title,
          description: form.description,
          endAt: new Date(form.endAt).toISOString(),
          groupIds: form.groupIds,
          userIds: form.userIds,
          problemIds: form.problemIds
        }
    await apiFetch(editingId.value ? `/api/admin/assignments/${editingId.value}` : '/api/admin/assignments', {
      method: editingId.value ? 'PATCH' : 'POST',
      body: JSON.stringify(body)
    })
    showFormModal.value = false
    resetForm()
    await loadAssignments()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

async function deleteAssignment(row: AssignmentRow) {
  try {
    await apiFetch(`/api/admin/assignments/${row.id}`, { method: 'DELETE' })
    await loadAssignments()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  void loadAssignments()
}

function handlePageSizeChange(pageSize: number) {
  pagination.pageSize = pageSize
  pagination.page = 1
  void loadAssignments()
}

function rowProps(row: AssignmentRow) {
  return {
    class: isAssignmentEnded(row) ? 'assignment-row-ended' : ''
  }
}

function renderIcon(icon: Component) {
  return h(NIcon, { component: icon })
}

function tooltipIconButton(
  icon: Component,
  label: string,
  onClick: () => void,
  options: { type?: 'error' } = {}
) {
  return h(
    NTooltip,
    { trigger: 'hover' },
    {
      trigger: () =>
        h(
          NButton,
          {
            size: 'small',
            quaternary: true,
            circle: true,
            type: options.type,
            onClick
          },
          { icon: () => renderIcon(icon) }
        ),
      default: () => label
    }
  )
}

watch(
  () => auth.signedIn,
  (signedIn) => {
    if (signedIn) {
      void loadAssignments()
      void loadAdminOptions()
    }
  }
)

onMounted(() => {
  if (auth.signedIn) {
    loadAssignments()
    void loadAdminOptions()
  } else {
    loading.value = false
  }
})

function assignmentProgressPercent(row: AssignmentRow) {
  if (!row.total) return 0
  return Math.round((row.completed / row.total) * 100)
}

function assignmentState(row: AssignmentRow) {
  if (row.deletedAt) return { label: t('admin.assignments.deleted'), type: 'error' as const }
  if (isAssignmentEnded(row)) return null
  if (row.total > 0 && row.completed >= row.total) return { label: t('assignments.done'), type: 'success' as const }
  const end = new Date(row.endAt).getTime()
  if (end - Date.now() <= 24 * 60 * 60 * 1000) return { label: t('assignments.dueSoon'), type: 'warning' as const }
  return { label: t('assignments.open'), type: 'info' as const }
}

function isAssignmentEnded(row: AssignmentRow) {
  return new Date(row.endAt).getTime() <= Date.now()
}
</script>

<template>
  <main class="page">
    <n-alert v-if="!auth.signedIn" type="warning" class="page-alert">
      {{ t('assignments.signIn') }}
    </n-alert>

    <n-alert v-if="error" type="error" class="page-alert">
      {{ error }}
    </n-alert>

    <n-card :bordered="false">
      <n-space v-if="canManage" justify="end" class="table-toolbar">
        <n-button type="primary" @click="openCreate">
          <template #icon>
            <n-icon :component="AddOutline" />
          </template>
          {{ t('admin.assignments.create') }}
        </n-button>
      </n-space>
      <n-data-table
        remote
        :columns="columns"
        :data="assignments"
        :bordered="false"
        :loading="loading"
        :pagination="pagination"
        :scroll-x="canManage ? 900 : 700"
        :row-props="rowProps"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      >
        <template #empty>
          <n-empty :description="t('assignments.empty')" />
        </template>
      </n-data-table>
    </n-card>

    <n-modal
      v-if="showFormModal"
      v-model:show="showFormModal"
      preset="card"
      :title="modalTitle"
      class="form-modal"
      style="width: min(760px, calc(100vw - 32px))"
    >
      <n-form :model="form" label-placement="top" class="assignment-form">
        <n-form-item :label="t('common.title')">
          <n-input v-model:value="form.title" />
        </n-form-item>
        <n-form-item :label="t('admin.description')">
          <n-input v-model:value="form.description" type="textarea" />
        </n-form-item>
        <n-form-item :label="t('admin.assignments.endAt')">
          <n-date-picker
            v-model:value="form.endAt"
            type="datetime"
            class="full-width"
            :disabled="editingEnded"
          />
        </n-form-item>
        <n-form-item :label="t('admin.assignments.selectGroups')">
          <n-select
            v-model:value="form.groupIds"
            multiple
            filterable
            :disabled="editingEnded"
            :options="groupOptions"
          />
        </n-form-item>
        <n-form-item :label="t('admin.assignments.selectUsers')">
          <n-select
            v-model:value="form.userIds"
            multiple
            filterable
            :disabled="editingEnded"
            :options="userOptions"
          />
        </n-form-item>
        <n-form-item :label="t('admin.assignments.selectProblems')">
          <n-select
            v-model:value="form.problemIds"
            multiple
            filterable
            :disabled="editingEnded"
            :options="problemOptions"
          />
        </n-form-item>
        <n-space justify="end" class="form-actions assignment-form-actions">
          <n-button @click="showFormModal = false">{{ t('admin.cancel') }}</n-button>
          <n-button
            type="primary"
            :loading="saving"
            :disabled="!form.title || (!editingEnded && !form.problemIds.length)"
            @click="saveAssignment"
          >
            {{ t('admin.save') }}
          </n-button>
        </n-space>
      </n-form>
    </n-modal>
  </main>
</template>

<style scoped lang="scss">
.assignment-title-cell {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.assignment-title-row {
  display: flex;
  gap: 8px;
  align-items: center;
  min-width: 0;
}

.assignment-title {
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.assignment-date {
  white-space: nowrap;
}

.assignment-progress {
  display: grid;
  grid-template-columns: minmax(90px, 1fr) auto;
  gap: 10px;
  align-items: center;
}

.assignment-progress-text {
  color: var(--muted-color);
  font-size: 13px;
  white-space: nowrap;
}

:deep(.assignment-row-ended .n-data-table-td) {
  background-color: color-mix(in srgb, var(--surface-bg) 90%, var(--text-color) 10%);
}

:deep(.assignment-row-ended:hover .n-data-table-td) {
  background-color: color-mix(in srgb, var(--surface-bg) 86%, var(--text-color) 14%);
}

.assignment-form {
  max-height: calc(100vh - 190px);
  overflow-y: auto;
  padding-right: 4px;
}

.assignment-form-actions {
  position: sticky;
  bottom: 0;
  z-index: 1;
  padding-top: 12px;
  padding-bottom: 2px;
  background: var(--surface-bg);
}
</style>
