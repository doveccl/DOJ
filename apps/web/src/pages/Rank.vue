<script setup lang="ts">
import { NAvatar } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { apiFetch, DEFAULT_PAGE_SIZE, getItems, PAGE_SIZE_OPTIONS, type Paged } from '../api'

interface RankRow {
  rank: number
  user: { id: number; name: string; introduction: string; avatarUrl: string }
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
    minWidth: 260,
    render(row) {
      return h('div', { class: 'rank-user' }, [
        h(NAvatar, { size: 28, src: row.user.avatarUrl, round: true }),
        h('div', { class: 'rank-user-text' }, [
          h('span', { class: 'rank-user-name' }, row.user.name),
          h('span', { class: 'rank-user-intro' }, row.user.introduction || t('profile.noIntroduction'))
        ])
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

<style scoped lang="scss">
.rank-user {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr);
  gap: 9px;
  align-items: center;
  min-width: 0;
}

.rank-user-text {
  display: grid;
  gap: 2px;
  min-width: 0;
}

.rank-user-name,
.rank-user-intro {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rank-user-name {
  font-weight: 600;
}

.rank-user-intro {
  color: var(--muted-color);
  font-size: 12px;
}
</style>
