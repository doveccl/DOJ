<script setup lang="ts">
import { NButton, NEllipsis, NIcon, NPopconfirm, NSpace, NTag, NTooltip } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import type { Component } from 'vue'
import {
  AddOutline,
  CloseOutline,
  EyeOffOutline,
  EyeOutline,
  SearchOutline,
  TrashOutline
} from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { apiFetch, DEFAULT_PAGE_SIZE, getItems, PAGE_SIZE_OPTIONS, type Paged } from '../api'
import { useAuthStore } from '../stores/auth'

interface ProblemRow {
  id: number
  title: string
  tags: string[]
  solvedCount: number
  attemptedCount: number
  submissionCount: number
  passRate: number
  solved: boolean
  submitted: boolean
  visible: boolean
  deletedAt: string | null
}

const { t } = useI18n()
const auth = useAuthStore()
const router = useRouter()
const canManage = computed(() => auth.user?.admin ?? false)
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const showCreateModal = ref(false)
const problems = ref<ProblemRow[]>([])
const search = ref('')
const tag = ref<string | null>(null)
const tags = ref<string[]>([])
const createForm = reactive({
  title: '',
  tags: [] as string[],
  mode: 'default' as 'default' | 'strict' | 'custom',
  timeLimit: 1000,
  memoryLimitMb: 256,
  visible: false
})
const pagination = reactive({
  page: 1,
  pageSize: DEFAULT_PAGE_SIZE,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [...PAGE_SIZE_OPTIONS]
})

const columns = computed<DataTableColumns<ProblemRow>>(() => [
  { title: t('common.id'), key: 'id', width: 96 },
  {
    title: t('common.title'),
    key: 'title',
    minWidth: 300,
    render(row) {
      return h(
        NEllipsis,
        { class: 'problem-title-line', tooltip: { placement: 'top-start' } },
        { default: () => h(RouterLink, { to: `/problems/${row.id}`, class: 'table-link problem-title-link' }, () => row.title) }
      )
    }
  },
  {
    title: t('common.tags'),
    key: 'tags',
    minWidth: 180,
    render(row) {
      if (!row.tags.length) return '-'
      const visibleTags = row.tags.slice(0, 2)
      return h(NSpace, { class: 'problem-tags-line', size: 6, wrap: false }, () =>
        [
          ...visibleTags.map((item) =>
            h(NTag, { key: item, bordered: false, size: 'small' }, () =>
              h(NEllipsis, { style: 'max-width: 96px' }, { default: () => item })
            )
          ),
          row.tags.length > visibleTags.length
            ? h(NTag, { key: 'more', bordered: false, size: 'small' }, () => `+${row.tags.length - visibleTags.length}`)
            : null
        ].filter(Boolean)
      )
    }
  },
  {
    title: t('problems.passRate'),
    key: 'passRate',
    width: 180,
    render(row) {
      return h(
        'span',
        { class: 'problem-stats' },
        `${row.solvedCount}/${row.submissionCount} (${((row.passRate ?? 0) * 100).toFixed(0)}%)`
      )
    }
  },
  ...(canManage.value
    ? [
        {
          title: t('admin.actions'),
          key: 'actions',
          width: 120,
          render(row: ProblemRow) {
            return h(NSpace, { size: 8 }, () => [
              tooltipIconButton(
                row.visible ? EyeOutline : EyeOffOutline,
                row.visible ? t('admin.problems.hide') : t('admin.problems.show'),
                () => toggleVisible(row, !row.visible),
                { disabled: !!row.deletedAt, type: row.visible ? 'success' : undefined }
              ),
              h(
                NPopconfirm,
                { onPositiveClick: () => deleteProblem(row) },
                {
                  trigger: () =>
                    tooltipIconButton(
                      TrashOutline,
                      t('admin.delete'),
                      () => {},
                      { type: 'error' }
                    ),
                  default: () => t('problems.deleteConfirm')
                }
              )
            ])
          }
        }
      ]
    : [])
])

async function loadProblems() {
  loading.value = true
  try {
    const params = new URLSearchParams({
      page: String(pagination.page),
      pageSize: String(pagination.pageSize)
    })
    if (search.value.trim()) params.set('q', search.value.trim())
    if (tag.value) params.set('tag', tag.value)
    const data = await apiFetch<Paged<ProblemRow>>(`/api/problems?${params}`)
    problems.value = getItems(data)
    pagination.itemCount = data.total
  } finally {
    loading.value = false
  }
}

async function loadTags() {
  try {
    const items = await apiFetch<Array<{ name: string; count: number }>>('/api/tags')
    tags.value = items.map((item) => item.name)
  } catch {
    tags.value = []
  }
}

async function createProblem() {
  saving.value = true
  error.value = ''
  try {
    const created = await apiFetch<ProblemRow>('/api/admin/problems', {
      method: 'POST',
      body: JSON.stringify({
        title: createForm.title,
        tags: createForm.tags,
        mode: createForm.mode,
        timeLimit: createForm.timeLimit,
        memoryLimit: createForm.memoryLimitMb * 1024 * 1024,
        visible: createForm.visible
      })
    })
    showCreateModal.value = false
    createForm.title = ''
    createForm.tags = []
    createForm.mode = 'default'
    createForm.timeLimit = 1000
    createForm.memoryLimitMb = 256
    createForm.visible = false
    await router.push(`/problems/${created.id}`)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

async function toggleVisible(row: ProblemRow, visible: boolean) {
  const previous = row.visible
  row.visible = visible
  try {
    await apiFetch(`/api/admin/problems/${row.id}`, {
      method: 'PATCH',
      body: JSON.stringify({ visible })
    })
  } catch (cause) {
    row.visible = previous
    error.value = cause instanceof Error ? cause.message : String(cause)
  }
}

async function deleteProblem(row: ProblemRow) {
  try {
    await apiFetch(`/api/admin/problems/${row.id}`, { method: 'DELETE' })
    await loadProblems()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  void loadProblems()
}

function handlePageSizeChange(pageSize: number) {
  pagination.pageSize = pageSize
  pagination.page = 1
  void loadProblems()
}

function applyFilters() {
  pagination.page = 1
  void loadProblems()
}

function clearFilters() {
  search.value = ''
  tag.value = null
  applyFilters()
}

function rowProps(row: ProblemRow) {
  const progressClass = row.solved ? 'problem-row-solved' : row.submitted ? 'problem-row-attempted' : ''
  return {
    class: [progressClass, row.visible ? '' : 'problem-row-hidden']
      .filter(Boolean)
      .join(' ')
  }
}

function renderIcon(icon: Component) {
  return h(NIcon, { component: icon })
}

function tooltipIconButton(
  icon: Component,
  label: string,
  onClick: () => void,
  options: { type?: 'success' | 'warning' | 'error'; disabled?: boolean } = {}
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
            disabled: options.disabled,
            onClick
          },
          { icon: () => renderIcon(icon) }
        ),
      default: () => label
    }
  )
}

onMounted(() => {
  void loadProblems()
  void loadTags()
})
</script>

<template>
  <main class="page">
    <p v-if="error" class="form-error page-alert">{{ error }}</p>
    <n-card :bordered="false">
      <div class="table-toolbar problem-toolbar">
        <div class="toolbar-fields">
          <n-input
            v-model:value="search"
            clearable
            class="toolbar-field problem-search"
            :placeholder="t('problems.search')"
            @keyup.enter="applyFilters"
            @clear="applyFilters"
          />
          <n-select
            v-model:value="tag"
            clearable
            filterable
            class="toolbar-field tag-filter"
            :options="tags.map((item) => ({ label: item, value: item }))"
            :placeholder="t('problems.tag')"
            @update:value="applyFilters"
          />
          <n-button secondary @click="clearFilters">
            <template #icon>
              <n-icon :component="CloseOutline" />
            </template>
            {{ t('problems.clear') }}
          </n-button>
        </div>
        <div class="toolbar-actions">
          <n-button type="primary" @click="applyFilters">
            <template #icon>
              <n-icon :component="SearchOutline" />
            </template>
            {{ t('problems.searchAction') }}
          </n-button>
          <n-button v-if="canManage" type="primary" secondary @click="showCreateModal = true">
            <template #icon>
              <n-icon :component="AddOutline" />
            </template>
            {{ t('admin.problems.create') }}
          </n-button>
        </div>
      </div>

      <n-data-table
        remote
        :columns="columns"
        :data="problems"
        :bordered="false"
        :loading="loading"
        :pagination="pagination"
        :row-props="rowProps"
        :scroll-x="canManage ? 840 : 660"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      >
        <template #empty>
          <n-empty :description="t('problems.empty')" />
        </template>
      </n-data-table>
    </n-card>

    <n-modal
      v-if="showCreateModal"
      v-model:show="showCreateModal"
      preset="card"
      :title="t('admin.problems.create')"
      class="form-modal"
    >
      <n-form :model="createForm" label-placement="top">
        <n-form-item :label="t('common.title')">
          <n-input v-model:value="createForm.title" placeholder="A+B Problem" />
        </n-form-item>
        <n-form-item :label="t('common.tags')">
          <n-dynamic-tags v-model:value="createForm.tags" />
        </n-form-item>
        <div class="form-grid two">
          <n-form-item :label="t('admin.problems.mode')">
            <n-select
              v-model:value="createForm.mode"
              :options="[
                { label: 'default', value: 'default' },
                { label: 'strict', value: 'strict' },
                { label: 'custom', value: 'custom' }
              ]"
            />
          </n-form-item>
          <n-form-item :label="t('admin.problems.visible')">
            <n-switch v-model:value="createForm.visible" />
          </n-form-item>
          <n-form-item :label="t('admin.problems.timeMs')">
            <n-input-number v-model:value="createForm.timeLimit" :min="100" class="full-width" />
          </n-form-item>
          <n-form-item :label="t('admin.problems.memoryMb')">
            <n-input-number v-model:value="createForm.memoryLimitMb" :min="16" class="full-width" />
          </n-form-item>
        </div>
        <n-space justify="end" class="form-actions">
          <n-button @click="showCreateModal = false">{{ t('admin.cancel') }}</n-button>
          <n-button
            type="primary"
            :loading="saving"
            :disabled="!createForm.title"
            @click="createProblem"
          >
            {{ t('admin.create') }}
          </n-button>
        </n-space>
      </n-form>
    </n-modal>
  </main>
</template>

<style scoped lang="scss">
.problem-toolbar {
  align-items: center;
}

.problem-toolbar .toolbar-fields {
  flex: 0 1 auto;
  display: grid;
  grid-template-columns: minmax(220px, 320px) minmax(160px, 220px) auto;
}

.problem-toolbar .toolbar-actions {
  flex: 0 0 auto;
  margin-left: auto;
}

.problem-search,
.tag-filter {
  width: 100%;
}

.problem-title-line {
  min-width: 0;
  max-width: 100%;
}

.problem-title-link {
  display: inline;
  min-width: 0;
}

.problem-tags-line {
  max-width: 100%;
  overflow: hidden;
}

.problem-stats {
  white-space: nowrap;
}

:deep(.problem-row-solved td) {
  background: rgba(16, 185, 129, 0.12);
}

:deep(.problem-row-solved:hover td) {
  background: rgba(16, 185, 129, 0.16);
}

:deep(.problem-row-attempted td) {
  background: rgba(245, 158, 11, 0.12);
}

:deep(.problem-row-attempted:hover td) {
  background: rgba(245, 158, 11, 0.16);
}

:deep(.problem-row-hidden td:nth-child(1)),
:deep(.problem-row-hidden td:nth-child(2)),
:deep(.problem-row-hidden td:nth-child(2) .table-link) {
  color: var(--muted-color);
}

@media (max-width: 760px) {
  .problem-toolbar .toolbar-fields {
    grid-template-columns: 1fr;
  }

  .problem-toolbar .toolbar-actions {
    width: 100%;
  }
}
</style>
