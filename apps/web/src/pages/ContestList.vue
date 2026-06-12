<script setup lang="ts">
import { NButton, NPopconfirm, NSpace, NTag } from 'naive-ui'
import type { DataTableColumns, SelectOption } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { apiFetch, DEFAULT_PAGE_SIZE, getItems, PAGE_SIZE_OPTIONS, type Paged } from '../api'
import { useAuthStore } from '../stores/auth'

interface ContestRow {
  id: number
  title: string
  description: string
  type: 'OI' | 'ICPC'
  startAt: string
  endAt: string
  freezeAt: string | null
  deletedAt?: string | null
}

interface ContestDetail {
  contest: ContestRow
  problems: Array<{ id?: number; problemId?: number; title: string | null; sort: number }>
}

interface ProblemRow {
  id: number
  title: string
}

type ContestStatusFilter = 'all' | 'current' | 'upcoming' | 'past'

const auth = useAuthStore()
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const showFormModal = ref(false)
const editingId = ref<number | null>(null)
const editingLocked = ref(false)
const statusFilter = ref<ContestStatusFilter>('all')
const contests = ref<ContestRow[]>([])
const problemOptions = ref<SelectOption[]>([])
const { t } = useI18n()
const canManage = computed(() => auth.user?.admin ?? false)
const form = reactive({
  title: '',
  description: '',
  type: 'OI' as 'OI' | 'ICPC',
  startAt: Date.now(),
  endAt: Date.now() + 2 * 60 * 60 * 1000,
  freezeAt: null as number | null,
  problemIds: [] as number[]
})
const pagination = reactive({
  page: 1,
  pageSize: DEFAULT_PAGE_SIZE,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [...PAGE_SIZE_OPTIONS]
})

const modalTitle = computed(() => (editingId.value ? t('admin.contests.edit') : t('admin.contests.create')))
const statusOptions = computed(() => [
  { label: t('common.all'), value: 'all' },
  { label: t('contests.current'), value: 'current' },
  { label: t('contests.upcoming'), value: 'upcoming' },
  { label: t('contests.past'), value: 'past' }
])

const columns = computed<DataTableColumns<ContestRow>>(() => [
  {
    title: t('common.title'),
    key: 'title',
    minWidth: 220,
    render(row) {
      const state = contestState(row)
      return h('div', { class: 'contest-title-cell' }, [
        h('div', { class: 'contest-title-row' }, [
          h(RouterLink, { to: `/contests/${row.id}`, class: 'table-link contest-title' }, () => row.title),
          h(NTag, { bordered: false, type: state.type, size: 'small' }, () => state.label)
        ]),
        h('span', { class: 'muted contest-time-left' }, timeLeft(row.endAt))
      ])
    }
  },
  {
    title: t('contests.type'),
    key: 'type',
    width: 110,
    render(row) {
      return h(NTag, { bordered: false }, () => row.type)
    }
  },
  {
    title: t('contests.start'),
    key: 'startAt',
    render(row) {
      return new Date(row.startAt).toLocaleString()
    }
  },
  {
    title: t('contests.end'),
    key: 'endAt',
    render(row) {
      return new Date(row.endAt).toLocaleString()
    }
  },
  ...(canManage.value
    ? [
        {
          title: t('admin.actions'),
          key: 'actions',
          width: 220,
          render(row: ContestRow) {
            return h(NSpace, { size: 8 }, () => [
              h(NButton, { size: 'small', secondary: true, onClick: () => openEdit(row.id) }, () => t('admin.edit')),
              h(
                NPopconfirm,
                { onPositiveClick: () => deleteContest(row) },
                {
                  trigger: () =>
                    h(
                      NButton,
                      { size: 'small', tertiary: true, type: 'error' },
                      () => t('admin.delete')
                    ),
                  default: () => t('admin.contests.deleteConfirm')
                }
              )
            ])
          }
        }
      ]
    : [])
])

onMounted(() => {
  void loadContests()
  void loadProblemOptions()
})

async function loadContests() {
  loading.value = true
  error.value = ''
  try {
    const params = new URLSearchParams({
      page: String(pagination.page),
      pageSize: String(pagination.pageSize)
    })
    if (statusFilter.value !== 'all') params.set('status', statusFilter.value)
    const data = await apiFetch<Paged<ContestRow>>(`/api/contests?${params}`)
    contests.value = getItems(data)
    pagination.itemCount = data.total
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

async function loadProblemOptions() {
  if (!canManage.value) return
  try {
    const problems = await apiFetch<Paged<ProblemRow>>('/api/problems?pageSize=100')
    problemOptions.value = getItems(problems).map((problem) => ({
      label: `P${problem.id} ${problem.title}`,
      value: problem.id
    }))
  } catch {
    problemOptions.value = []
  }
}

function resetForm() {
  editingId.value = null
  editingLocked.value = false
  form.title = ''
  form.description = ''
  form.type = 'OI'
  form.startAt = Date.now()
  form.endAt = Date.now() + 2 * 60 * 60 * 1000
  form.freezeAt = null
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
    const detail = await apiFetch<ContestDetail>(`/api/contests/${id}`)
    const now = Date.now()
    editingId.value = id
    editingLocked.value = now >= new Date(detail.contest.startAt).getTime() && now < new Date(detail.contest.endAt).getTime()
    form.title = detail.contest.title
    form.description = detail.contest.description
    form.type = detail.contest.type
    form.startAt = new Date(detail.contest.startAt).getTime()
    form.endAt = new Date(detail.contest.endAt).getTime()
    form.freezeAt = detail.contest.freezeAt ? new Date(detail.contest.freezeAt).getTime() : null
    form.problemIds = detail.problems
      .map((problem) => problem.problemId ?? problem.id)
      .filter((id): id is number => typeof id === 'number')
    showFormModal.value = true
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

async function saveContest() {
  saving.value = true
  error.value = ''
  try {
    const body = editingLocked.value
      ? {
          title: form.title,
          description: form.description
        }
      : {
          title: form.title,
          description: form.description,
          type: form.type,
          startAt: new Date(form.startAt).toISOString(),
          endAt: new Date(form.endAt).toISOString(),
          freezeAt: form.type === 'ICPC' && form.freezeAt ? new Date(form.freezeAt).toISOString() : null,
          problemIds: form.problemIds
        }
    await apiFetch(editingId.value ? `/api/admin/contests/${editingId.value}` : '/api/admin/contests', {
      method: editingId.value ? 'PATCH' : 'POST',
      body: JSON.stringify(body)
    })
    showFormModal.value = false
    resetForm()
    await loadContests()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

async function deleteContest(row: ContestRow) {
  try {
    await apiFetch(`/api/admin/contests/${row.id}`, { method: 'DELETE' })
    await loadContests()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  }
}

function handleStatusChange() {
  pagination.page = 1
  void loadContests()
}

function contestState(row: ContestRow) {
  if (row.deletedAt) return { label: t('admin.contests.deleted'), type: 'error' as const }
  const now = Date.now()
  const start = new Date(row.startAt).getTime()
  const end = new Date(row.endAt).getTime()
  if (now < start) return { label: t('contests.upcoming'), type: 'info' as const }
  if (now >= end) return { label: t('contests.past'), type: 'default' as const }
  return { label: t('contests.current'), type: 'success' as const }
}

function timeLeft(value: string) {
  const diff = new Date(value).getTime() - Date.now()
  if (diff <= 0) return t('assignments.ended')
  const hours = Math.ceil(diff / (60 * 60 * 1000))
  if (hours < 24) return `${hours}h`
  return `${Math.ceil(hours / 24)}d`
}

function handlePageChange(page: number) {
  pagination.page = page
  void loadContests()
}

function handlePageSizeChange(pageSize: number) {
  pagination.pageSize = pageSize
  pagination.page = 1
  void loadContests()
}

watch(
  () => form.type,
  (type) => {
    if (type === 'OI') {
      form.freezeAt = null
    } else if (!form.freezeAt) {
      form.freezeAt = Math.max(form.startAt, form.endAt - 60 * 60 * 1000)
    }
  }
)

watch(
  () => form.endAt,
  (endAt) => {
    if (form.type === 'ICPC' && (!form.freezeAt || form.freezeAt >= endAt)) {
      form.freezeAt = Math.max(form.startAt, endAt - 60 * 60 * 1000)
    }
  }
)
</script>

<template>
  <main class="page">
    <n-alert v-if="error" type="error" class="page-alert">{{ error }}</n-alert>
    <n-card :bordered="false">
      <n-space justify="space-between" align="center" class="table-toolbar">
        <n-radio-group
          v-model:value="statusFilter"
          class="status-filter"
          @update:value="handleStatusChange"
        >
          <n-radio-button
            v-for="option in statusOptions"
            :key="option.value"
            :value="option.value"
          >
            {{ option.label }}
          </n-radio-button>
        </n-radio-group>
        <n-button v-if="canManage" type="primary" @click="openCreate">
          {{ t('admin.contests.create') }}
        </n-button>
      </n-space>
      <n-empty
        v-if="!loading && !contests.length"
        class="empty-state"
        :description="t('contests.empty')"
      >
        <template #extra>
          <n-button secondary size="small" @click="loadContests">
            {{ t('common.refresh') }}
          </n-button>
        </template>
      </n-empty>
      <n-data-table
        v-else
        remote
        :columns="columns"
        :data="contests"
        :bordered="false"
        :loading="loading"
        :pagination="pagination"
        :scroll-x="canManage ? 940 : 660"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>

    <n-modal
      v-if="showFormModal"
      v-model:show="showFormModal"
      preset="card"
      :title="modalTitle"
      class="form-modal"
    >
      <n-form :model="form" label-placement="top">
        <n-alert v-if="editingLocked" type="warning" class="card-alert">
          {{ t('admin.contests.runningEditHint') }}
        </n-alert>
        <n-form-item :label="t('common.title')">
          <n-input v-model:value="form.title" />
        </n-form-item>
        <n-form-item :label="t('admin.description')">
          <n-input v-model:value="form.description" type="textarea" />
        </n-form-item>
        <n-form-item :label="t('admin.contests.type')">
          <n-select
            v-model:value="form.type"
            :disabled="editingLocked"
            :options="[
              { label: 'OI', value: 'OI' },
              { label: 'ICPC', value: 'ICPC' }
            ]"
          />
        </n-form-item>
        <n-form-item :label="t('admin.contests.selectProblems')">
          <n-select
            v-model:value="form.problemIds"
            multiple
            filterable
            :disabled="editingLocked"
            :options="problemOptions"
          />
        </n-form-item>
        <div class="form-grid">
          <n-form-item :label="t('admin.contests.startAt')">
            <n-date-picker v-model:value="form.startAt" type="datetime" :disabled="editingLocked" class="full-width" />
          </n-form-item>
          <n-form-item :label="t('admin.contests.endAt')">
            <n-date-picker v-model:value="form.endAt" type="datetime" :disabled="editingLocked" class="full-width" />
          </n-form-item>
          <n-form-item v-if="form.type === 'ICPC'" :label="t('admin.contests.freezeAt')">
            <n-date-picker
              v-model:value="form.freezeAt"
              type="datetime"
              clearable
              :disabled="editingLocked"
              class="full-width"
            />
          </n-form-item>
        </div>
        <n-space justify="end" class="form-actions">
          <n-button @click="showFormModal = false">{{ t('admin.cancel') }}</n-button>
          <n-button
            type="primary"
            :loading="saving"
            :disabled="!form.title || (!editingLocked && (!form.problemIds.length || form.endAt <= form.startAt))"
            @click="saveContest"
          >
            {{ editingId ? t('admin.save') : t('admin.create') }}
          </n-button>
        </n-space>
      </n-form>
    </n-modal>
  </main>
</template>

<style scoped lang="scss">
.empty-state {
  padding: 48px 0;
}

.status-filter { max-width: 360px; }

.contest-title-cell {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.contest-title-row {
  display: flex;
  gap: 8px;
  align-items: center;
  min-width: 0;
}

.contest-title {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.contest-time-left {
  font-size: 12px;
}
</style>
