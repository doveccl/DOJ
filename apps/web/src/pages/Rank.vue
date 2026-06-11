<script setup lang="ts">
import { NAvatar, NSpace } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { apiFetch, DEFAULT_PAGE_SIZE, getItems, PAGE_SIZE_OPTIONS, type Paged } from '../api'

interface RankRow {
  rank: number
  user: { id: number; name: string; avatarUrl: string }
  solved: number
  submissions: number
  acAt: string | null
}

const loading = ref(true)
const rows = ref<RankRow[]>([])
const page = ref(1)
const pageSize = ref(DEFAULT_PAGE_SIZE)
const total = ref(0)
const { t } = useI18n()

const columns = computed<DataTableColumns<RankRow>>(() => [
  {
    title: '#',
    key: 'rank',
    width: 72,
    render(row) {
      return String(row.rank)
    }
  },
  {
    title: t('common.user'),
    key: 'user',
    render(row) {
      return h(NSpace, { align: 'center', size: 8 }, () => [
        h(NAvatar, { size: 28, src: row.user.avatarUrl, round: true }),
        h('span', row.user.name)
      ])
    }
  },
  {
    title: t('common.solved'),
    key: 'solved',
    width: 120
  },
  {
    title: t('common.submissions'),
    key: 'submissions',
    width: 140
  }
])

onMounted(load)

async function load() {
  loading.value = true
  try {
    const data = await apiFetch<Paged<RankRow>>(
      `/api/ranking?page=${page.value}&pageSize=${pageSize.value}`
    )
    rows.value = getItems(data)
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
      :scroll-x="560"
      :pagination="{
        page,
        pageSize,
        itemCount: total,
        showSizePicker: true,
        pageSizes: [...PAGE_SIZE_OPTIONS],
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
