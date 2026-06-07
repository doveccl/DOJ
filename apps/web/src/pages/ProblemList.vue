<script setup lang="ts">
import { NButton, NCard, NDataTable, NEmpty, NInput, NSelect, NSpace, NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { computed, h, onMounted, reactive, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../api'
import { useAuthStore } from '../stores/auth'

interface ProblemRow {
  id: number
  title: string
  tags: string[]
  solvedCount: number
  solved: boolean
}

const { t } = useI18n()
const auth = useAuthStore()
const canManage = computed(() => auth.user?.groups.includes('admin') ?? false)
const loading = ref(true)
const problems = ref<ProblemRow[]>([])
const search = ref('')
const tag = ref<string | null>(null)
const tags = ref<string[]>([])
const pagination = reactive({
  page: 1,
  pageSize: 50,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [20, 50, 100]
})

const columns = computed<DataTableColumns<ProblemRow>>(() => [
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
    title: t('problems.status'),
    key: 'status',
    width: 110,
    render(row) {
      return row.solved
        ? h(NTag, { bordered: false, type: 'success', size: 'small' }, () => t('problems.solved'))
        : '-'
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
  { title: t('common.solved'), key: 'solvedCount', width: 100 },
  ...(canManage.value
    ? [
        {
          title: t('admin.actions'),
          key: 'actions',
          width: 110,
          render() {
            return h(RouterLink, { to: '/admin/problems', class: 'table-link' }, () =>
              t('admin.edit')
            )
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
    if (search.value.trim()) params.set('search', search.value.trim())
    if (tag.value) params.set('tag', tag.value)
    const data = await apiFetch<{ list: ProblemRow[]; total: number; tags: string[] }>(
      `/api/problems?${params}`
    )
    problems.value = data.list
    pagination.itemCount = data.total
    tags.value = data.tags
  } finally {
    loading.value = false
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

onMounted(loadProblems)
</script>

<template>
  <main class="page">
    <n-card :bordered="false">
      <n-space class="table-toolbar" justify="space-between" align="center">
        <n-space>
          <n-input
            v-model:value="search"
            clearable
            :placeholder="t('problems.search')"
            @keyup.enter="applyFilters"
            @clear="applyFilters"
          />
          <n-select
            v-model:value="tag"
            clearable
            filterable
            class="tag-filter"
            :options="tags.map((item) => ({ label: item, value: item }))"
            :placeholder="t('problems.tag')"
            @update:value="applyFilters"
          />
        </n-space>
        <n-space>
          <n-button secondary @click="clearFilters">{{ t('problems.clear') }}</n-button>
          <n-button type="primary" @click="applyFilters">{{ t('problems.searchAction') }}</n-button>
        </n-space>
      </n-space>

      <n-data-table
        remote
        :columns="columns"
        :data="problems"
        :bordered="false"
        :loading="loading"
        :pagination="pagination"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      >
        <template #empty>
          <n-empty :description="t('problems.empty')" />
        </template>
      </n-data-table>
    </n-card>
  </main>
</template>

<style scoped>
.tag-filter {
  min-width: 160px;
}
</style>
