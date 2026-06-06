<script setup lang="ts">
import { NDataTable } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../api'

interface RankRow {
  id: number
  name: string
  solvedCount: number
  submissionCount: number
  introduction: string
}

const loading = ref(true)
const rows = ref<RankRow[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const { t } = useI18n()

const columns = computed<DataTableColumns<RankRow>>(() => [
  {
    title: '#',
    key: 'rank',
    width: 72,
    render(_row, index) {
      return String((page.value - 1) * pageSize.value + index + 1)
    }
  },
  { title: t('common.user'), key: 'name' },
  { title: t('common.solved'), key: 'solvedCount', width: 120 },
  { title: t('dashboard.submissions'), key: 'submissionCount', width: 140 },
  {
    title: t('rank.intro'),
    key: 'introduction',
    ellipsis: {
      tooltip: true
    }
  }
])

onMounted(load)

async function load() {
  loading.value = true
  try {
    const data = await apiFetch<{ list: RankRow[]; total: number }>(
      `/api/rank?page=${page.value}&pageSize=${pageSize.value}`
    )
    rows.value = data.list
    total.value = data.total
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main class="page">
    <n-data-table
      :columns="columns"
      :data="rows"
      :bordered="false"
      :loading="loading"
      :pagination="{
        page,
        pageSize,
        itemCount: total,
        showSizePicker: true,
        pageSizes: [20, 50, 100],
        onUpdatePage: (nextPage: number) => {
          page = nextPage
          load()
        },
        onUpdatePageSize: (nextPageSize: number) => {
          pageSize = nextPageSize
          page = 1
          load()
        }
      }"
    />
  </main>
</template>
