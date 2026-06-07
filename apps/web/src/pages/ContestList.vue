<script setup lang="ts">
import { NButton, NCard, NDataTable, NEmpty, NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { computed, h, onMounted, reactive, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../api'

interface ContestRow {
  id: number
  title: string
  type: 'OI' | 'ICPC'
  startAt: string
  endAt: string
}

const loading = ref(true)
const contests = ref<ContestRow[]>([])
const { t } = useI18n()
const pagination = reactive({
  page: 1,
  pageSize: 50,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [20, 50, 100]
})

const columns = computed<DataTableColumns<ContestRow>>(() => [
  {
    title: t('common.title'),
    key: 'title',
    render(row) {
      return h(RouterLink, { to: `/contests/${row.id}`, class: 'table-link' }, () => row.title)
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
  }
])

onMounted(loadContests)

async function loadContests() {
  loading.value = true
  try {
    const data = await apiFetch<{ list: ContestRow[]; total: number }>(
      `/api/contests?page=${pagination.page}&pageSize=${pagination.pageSize}`
    )
    contests.value = data.list
    pagination.itemCount = data.total
  } finally {
    loading.value = false
  }
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
</script>

<template>
  <main class="page">
    <n-card :bordered="false">
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
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>
  </main>
</template>

<style scoped lang="scss">
.empty-state {
  padding: 48px 0;
}
</style>
