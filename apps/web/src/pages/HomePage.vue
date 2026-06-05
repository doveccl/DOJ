<script setup lang="ts">
import { NCard, NDataTable, NGrid, NGridItem, NSpin, NStatistic, NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { computed, h, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../api'

interface Dashboard {
  stats: {
    problems: number
    submissions: number
    users: number
    contests: number
    assignments: number
  }
  recentSubmissions: Array<{
    id: number
    status: string
    languageId: string
    timeMs: number
    memoryBytes: number
    createdAt: string
    userId: number
    userName: string
    problemId: number
    problemTitle: string
  }>
}

const loading = ref(true)
const error = ref('')
const dashboard = ref<Dashboard | null>(null)
const { t } = useI18n()

const submissionColumns = computed<DataTableColumns<Dashboard['recentSubmissions'][number]>>(() => [
  {
    title: t('common.id'),
    key: 'id',
    width: 84,
    render(row) {
      return h(RouterLink, { to: `/submissions/${row.id}`, class: 'table-link' }, () => row.id)
    }
  },
  {
    title: t('common.problem'),
    key: 'problemTitle',
    minWidth: 220,
    render(row) {
      return h(
        RouterLink,
        { to: `/problems/${row.problemId}`, class: 'table-link' },
        () => row.problemTitle
      )
    }
  },
  { title: t('common.user'), key: 'userName', minWidth: 140 },
  {
    title: t('common.status'),
    key: 'status',
    width: 110,
    render(row) {
      return h(
        NTag,
        { bordered: false, type: row.status === 'AC' ? 'success' : 'warning' },
        () => row.status
      )
    }
  },
  { title: t('common.language'), key: 'languageId', width: 100 },
  {
    title: t('common.time'),
    key: 'timeMs',
    width: 100,
    render(row) {
      return `${row.timeMs} ms`
    }
  }
])

onMounted(async () => {
  try {
    dashboard.value = await apiFetch<Dashboard>('/api/dashboard')
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <main class="page">
    <section class="page-header">
      <h1>{{ t('dashboard.title') }}</h1>
      <p>{{ t('dashboard.subtitle') }}</p>
    </section>

    <n-spin :show="loading">
      <p v-if="error" class="form-error">{{ error }}</p>
      <template v-else-if="dashboard">
        <n-grid :cols="5" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
          <n-grid-item span="1 m:1 s:2 xs:5">
            <n-card :bordered="false">
              <n-statistic :label="t('dashboard.problems')" :value="dashboard.stats.problems" />
            </n-card>
          </n-grid-item>
          <n-grid-item span="1 m:1 s:2 xs:5">
            <n-card :bordered="false">
              <n-statistic
                :label="t('dashboard.submissions')"
                :value="dashboard.stats.submissions"
              />
            </n-card>
          </n-grid-item>
          <n-grid-item span="1 m:1 s:2 xs:5">
            <n-card :bordered="false">
              <n-statistic :label="t('dashboard.users')" :value="dashboard.stats.users" />
            </n-card>
          </n-grid-item>
          <n-grid-item span="1 m:1 s:2 xs:5">
            <n-card :bordered="false">
              <n-statistic :label="t('dashboard.contests')" :value="dashboard.stats.contests" />
            </n-card>
          </n-grid-item>
          <n-grid-item span="1 m:1 s:2 xs:5">
            <n-card :bordered="false">
              <n-statistic
                :label="t('dashboard.assignments')"
                :value="dashboard.stats.assignments"
              />
            </n-card>
          </n-grid-item>
        </n-grid>

        <n-card :title="t('dashboard.recentSubmissions')" :bordered="false" class="stacked-card">
          <n-data-table
            :columns="submissionColumns"
            :data="dashboard.recentSubmissions"
            :bordered="false"
          />
        </n-card>
      </template>
    </n-spin>
  </main>
</template>
