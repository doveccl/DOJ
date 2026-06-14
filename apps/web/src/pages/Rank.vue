<script setup lang="ts">
import { NAvatar } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import {
  apiFetch,
  DEFAULT_PAGE_SIZE,
  getItems,
  isUnauthorized,
  PAGE_SIZE_OPTIONS,
  type Paged
} from '../api'
import { useAuthStore } from '../stores/auth'

interface RankRow {
  rank: number
  user: { id: number; name: string; introduction: string; avatarUrl: string }
  solved: number
  submissions: number
  acAt: string | null
}

const loading = ref(true)
const rows = ref<RankRow[]>([])
const requireSignIn = ref(false)
const error = ref('')
const page = ref(1)
const pageSize = ref(DEFAULT_PAGE_SIZE)
const total = ref(0)
const { t } = useI18n()
const auth = useAuthStore()

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
        h('span', { class: 'rank-user-name', title: row.user.name }, row.user.name)
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
    requireSignIn.value = false
  } catch (cause) {
    if (isUnauthorized(cause)) {
      requireSignIn.value = true
      rows.value = []
      total.value = 0
    } else {
      error.value = cause instanceof Error ? cause.message : String(cause)
    }
  } finally {
    loading.value = false
  }
}

watch(
  () => auth.signedIn,
  () => {
    void load()
  }
)
</script>

<template>
  <main class="page">
    <n-alert v-if="error" type="error" class="page-alert">{{ error }}</n-alert>
    <n-card :bordered="false">
      <sign-in-required v-if="requireSignIn" />
      <n-data-table
        v-else
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
    </n-card>
  </main>
</template>

<style scoped lang="scss">
.rank-user {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr);
  gap: 9px;
  align-items: center;
  min-width: 0;
  min-height: 32px;
}

.rank-user-name {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 600;
}
</style>
