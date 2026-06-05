<script setup lang="ts">
import {
  NCard,
  NDataTable,
  NEmpty,
  NGrid,
  NGridItem,
  NSpace,
  NSpin,
  NStatistic,
  NTag
} from 'naive-ui'
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
  recentProblems: Array<{
    id: number
    title: string
    tags: string[]
    solvedCount: number
    createdAt: string
  }>
  recentTopics: Array<{
    id: number
    title: string
    tags: string[]
    userName: string
    updatedAt: string
  }>
  myAssignments: Array<{
    id: number
    title: string
    dueAt: string | null
    allowLate: boolean
  }>
}

const loading = ref(true)
const error = ref('')
const dashboard = ref<Dashboard | null>(null)
const { t } = useI18n()
const dateTime = new Intl.DateTimeFormat(undefined, {
  dateStyle: 'medium',
  timeStyle: 'short'
})

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

function formatDateTime(value: string | null) {
  return value ? dateTime.format(new Date(value)) : '-'
}

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
        <section class="dashboard-main">
          <n-card :title="t('dashboard.recentProblems')" :bordered="false">
            <div v-if="dashboard.recentProblems.length" class="dashboard-list">
              <router-link
                v-for="problem in dashboard.recentProblems"
                :key="problem.id"
                :to="`/problems/${problem.id}`"
                class="dashboard-list-item"
              >
                <span class="dashboard-item-title">P{{ problem.id }} {{ problem.title }}</span>
                <n-space :size="6">
                  <n-tag v-for="tag in problem.tags" :key="tag" size="small" :bordered="false">
                    {{ tag }}
                  </n-tag>
                  <span class="muted">{{ t('common.solved') }} {{ problem.solvedCount }}</span>
                </n-space>
              </router-link>
            </div>
            <n-empty v-else :description="t('problems.empty')" />
          </n-card>

          <n-card :title="t('dashboard.myAssignments')" :bordered="false">
            <div v-if="dashboard.myAssignments.length" class="dashboard-list">
              <router-link
                v-for="assignment in dashboard.myAssignments"
                :key="assignment.id"
                :to="`/assignments/${assignment.id}`"
                class="dashboard-list-item"
              >
                <span class="dashboard-item-title">{{ assignment.title }}</span>
                <span class="muted">
                  {{ t('assignments.duePrefix') }} {{ formatDateTime(assignment.dueAt) }}
                </span>
              </router-link>
            </div>
            <n-empty v-else :description="t('dashboard.noAssignments')" />
          </n-card>

          <n-card :title="t('dashboard.recentTopics')" :bordered="false">
            <div v-if="dashboard.recentTopics.length" class="dashboard-list">
              <router-link
                v-for="topic in dashboard.recentTopics"
                :key="topic.id"
                :to="`/bbs/${topic.id}`"
                class="dashboard-list-item"
              >
                <span class="dashboard-item-title">{{ topic.title }}</span>
                <span class="muted">
                  {{ topic.userName }} · {{ formatDateTime(topic.updatedAt) }}
                </span>
              </router-link>
            </div>
            <n-empty v-else :description="t('dashboard.noTopics')" />
          </n-card>
        </section>

        <n-grid
          :cols="5"
          :x-gap="16"
          :y-gap="16"
          responsive="screen"
          item-responsive
          class="stacked-card"
        >
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
