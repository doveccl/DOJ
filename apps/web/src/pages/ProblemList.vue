<script setup lang="ts">
import { NButton, NSpace, NSwitch, NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
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
  {
    title: t('problems.status'),
    key: 'status',
    width: 72,
    align: 'center',
    render(row) {
      return h(
        'span',
        {
          class: ['solved-icon', { solved: row.solved }],
          'aria-label': row.solved ? t('problems.solved') : t('problems.unsolved')
        },
        row.solved ? '✓' : ''
      )
    }
  },
  { title: t('common.id'), key: 'id', width: 96 },
  {
    title: t('common.title'),
    key: 'title',
    minWidth: 240,
    render(row) {
      return h(RouterLink, { to: `/problems/${row.id}`, class: 'table-link' }, () => row.title)
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
  {
    title: t('problems.passRate'),
    key: 'passRate',
    width: 110,
    render(row) {
      return `${((row.passRate ?? 0) * 100).toFixed(1)}%`
    }
  },
  ...(canManage.value
    ? [
        {
          title: t('admin.problems.visible'),
          key: 'visible',
          width: 120,
          render(row: ProblemRow) {
            return h(NSwitch, {
              value: row.visible,
              disabled: !!row.deletedAt,
              onUpdateValue: (visible: boolean) => toggleVisible(row, visible)
            })
          }
        },
        {
          title: t('admin.actions'),
          key: 'actions',
          width: 180,
          render(row: ProblemRow) {
            return h(NSpace, { size: 8 }, () => [
              h(
                NButton,
                {
                  size: 'small',
                  secondary: true,
                  onClick: () => router.push(`/problems/${row.id}`)
                },
                () => t('admin.edit')
              ),
              h(
                NButton,
                {
                  size: 'small',
                  tertiary: true,
                  type: row.deletedAt ? 'success' : 'error',
                  onClick: () => toggleDeleted(row)
                },
                () => (row.deletedAt ? t('admin.restore') : t('admin.delete'))
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
          <n-button secondary @click="clearFilters">{{ t('problems.clear') }}</n-button>
          <n-button type="primary" @click="applyFilters">{{ t('problems.searchAction') }}</n-button>
          <n-button v-if="canManage" type="primary" secondary @click="showCreateModal = true">
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
        :scroll-x="canManage ? 920 : 680"
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
