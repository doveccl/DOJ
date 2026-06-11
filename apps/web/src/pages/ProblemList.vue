<script setup lang="ts">
import { NButton, NIcon, NSpace, NTag, NTooltip } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import type { Component } from 'vue'
import {
  AddOutline,
  CloseOutline,
  EyeOffOutline,
  EyeOutline,
  SearchOutline,
  TrashOutline,
  RefreshOutline
} from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { apiFetch, DEFAULT_PAGE_SIZE, getItems, PAGE_SIZE_OPTIONS, type Paged } from '../api'
import { useAuthStore } from '../stores/auth'

interface ProblemRow {
  id: number
  title: string
  tags: string[]
  passRate: number
  solved: boolean
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
    minWidth: 240,
    render(row) {
      return h('div', { class: 'problem-title-cell' }, [
        h(RouterLink, { to: `/problems/${row.id}`, class: 'table-link problem-title-link' }, () => row.title),
        h(NSpace, { size: 6, align: 'center', wrap: false, class: 'problem-badges' }, () => [
          row.solved
            ? h(NTag, { bordered: false, size: 'small', type: 'success' }, () => t('problems.solved'))
            : null,
          !row.visible
            ? h(NTag, { bordered: false, size: 'small', type: 'warning' }, () => t('admin.disabled'))
            : null,
          row.deletedAt
            ? h(NTag, { bordered: false, size: 'small', type: 'error' }, () => t('admin.problems.deleted'))
            : null,
          h('span', { class: 'muted problem-rate' }, `${t('problems.passRate')} ${((row.passRate ?? 0) * 100).toFixed(0)}%`)
        ])
      ])
    }
  },
  {
    title: t('common.tags'),
    key: 'tags',
    minWidth: 180,
    render(row) {
      if (!row.tags.length) return '-'
      return h(NSpace, { size: 6 }, () =>
        row.tags.map((item) => h(NTag, { key: item, bordered: false, size: 'small' }, () => item))
      )
    }
  },
  ...(canManage.value
    ? [
        {
          title: t('admin.actions'),
          key: 'actions',
          width: 120,
          align: 'right' as const,
          render(row: ProblemRow) {
            return h(NSpace, { size: 8 }, () => [
              tooltipIconButton(
                row.visible ? EyeOutline : EyeOffOutline,
                row.visible ? t('admin.enabled') : t('admin.disabled'),
                () => toggleVisible(row, !row.visible),
                { disabled: !!row.deletedAt }
              ),
              tooltipIconButton(
                row.deletedAt ? RefreshOutline : TrashOutline,
                row.deletedAt ? t('admin.restore') : t('admin.delete'),
                () => toggleDeleted(row),
                { type: row.deletedAt ? 'success' : 'error' }
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

async function toggleDeleted(row: ProblemRow) {
  try {
    await apiFetch(
      row.deletedAt ? `/api/admin/problems/${row.id}/restore` : `/api/admin/problems/${row.id}`,
      { method: row.deletedAt ? 'POST' : 'DELETE' }
    )
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

function renderIcon(icon: Component) {
  return h(NIcon, { component: icon })
}

function tooltipIconButton(
  icon: Component,
  label: string,
  onClick: () => void,
  options: { type?: 'success' | 'error'; disabled?: boolean } = {}
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
        </div>
        <div class="toolbar-actions">
          <n-button secondary @click="clearFilters">
            <template #icon>
              <n-icon :component="CloseOutline" />
            </template>
            {{ t('problems.clear') }}
          </n-button>
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
        :scroll-x="canManage ? 820 : 640"
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
  align-items: stretch;
}

.problem-toolbar .toolbar-fields {
  flex: 1 1 420px;
  display: grid;
  grid-template-columns: minmax(220px, 1.2fr) minmax(180px, 0.8fr);
}

.problem-toolbar .toolbar-actions {
  flex: 0 0 auto;
}

.problem-search,
.tag-filter {
  width: 100%;
}

.problem-title-cell {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.problem-title-link {
  display: block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.problem-badges {
  max-width: 100%;
  overflow: hidden;
}

.problem-rate {
  font-size: 12px;
  white-space: nowrap;
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

<style scoped>
.problem-search {
  width: min(320px, 100%);
}

.tag-filter {
  width: min(180px, 100%);
}

.solved-icon {
  display: inline-grid;
  width: 18px;
  height: 18px;
  place-items: center;
  color: transparent;
  border: 1px solid var(--border-strong);
  border-radius: 50%;
  font-size: 12px;
  font-weight: 700;
}

.solved-icon.solved {
  color: #fff;
  border-color: var(--brand);
  background: var(--brand);
}
</style>
