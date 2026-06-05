<script setup lang="ts">
import { NCard, NSpace, NSpin, NTag } from 'naive-ui'
import { computed, onMounted, ref } from 'vue'
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
  recentContests: Array<{
    id: number
    title: string
    type: string
    startAt: string
    endAt: string
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
const hasCards = computed(() => {
  const data = dashboard.value
  return Boolean(
    data &&
    (data.recentProblems.length ||
      data.myAssignments.length ||
      data.recentContests.length ||
      data.recentTopics.length)
  )
})

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
    <n-spin :show="loading">
      <p v-if="error" class="form-error">{{ error }}</p>
      <template v-else-if="dashboard">
        <section v-if="hasCards" class="dashboard-main">
          <n-card
            v-if="dashboard.recentProblems.length"
            :title="t('dashboard.recentProblems')"
            :bordered="false"
          >
            <div class="dashboard-list">
              <router-link
                v-for="problem in dashboard.recentProblems.slice(0, 5)"
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
          </n-card>

          <n-card
            v-if="dashboard.myAssignments.length"
            :title="t('dashboard.myAssignments')"
            :bordered="false"
          >
            <div class="dashboard-list">
              <router-link
                v-for="assignment in dashboard.myAssignments.slice(0, 5)"
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
          </n-card>

          <n-card
            v-if="dashboard.recentContests.length"
            :title="t('dashboard.recentContests')"
            :bordered="false"
          >
            <div class="dashboard-list">
              <router-link
                v-for="contest in dashboard.recentContests.slice(0, 5)"
                :key="contest.id"
                :to="`/contests/${contest.id}`"
                class="dashboard-list-item"
              >
                <span class="dashboard-item-title">{{ contest.title }}</span>
                <span class="muted">
                  {{ contest.type }} · {{ formatDateTime(contest.startAt) }}
                </span>
              </router-link>
            </div>
          </n-card>

          <n-card
            v-if="dashboard.recentTopics.length"
            :title="t('dashboard.recentTopics')"
            :bordered="false"
          >
            <div class="dashboard-list">
              <router-link
                v-for="topic in dashboard.recentTopics.slice(0, 5)"
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
          </n-card>
        </section>
      </template>
    </n-spin>
  </main>
</template>
