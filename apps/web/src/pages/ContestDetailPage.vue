<script setup lang="ts">
import { NAlert, NCard, NDataTable, NSpin, NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { computed, h, onMounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { apiFetch } from '../api'
import { useAuthStore } from '../stores/auth'

interface Contest {
  id: number
  title: string
  description: string
  type: 'OI' | 'ICPC'
  startAt: string
  endAt: string
  freezeAt: string | null
}

interface ContestProblem {
  id: number
  key: string
  title: string
  score: number
}

interface ContestDetail {
  contest: Contest
  problems: ContestProblem[]
}

interface ScoreboardRow {
  userId: number
  userName: string
  solved: number
  penalty: number
  problems: Record<
    string,
    { attempts: number; solved: boolean; penalty: number; frozenAttempts: number }
  >
}

interface Scoreboard {
  frozen: boolean
  revealed: boolean
  visibleUntil: string | null
  rows: ScoreboardRow[]
}

const route = useRoute()
const auth = useAuthStore()
const loading = ref(true)
const error = ref('')
const detail = ref<ContestDetail | null>(null)
const scoreboard = ref<Scoreboard | null>(null)
const canReveal = computed(() => auth.user?.groups.includes('admin') ?? false)

const columns: DataTableColumns<ContestProblem> = [
  { title: 'Key', key: 'key', width: 90 },
  {
    title: 'Problem',
    key: 'title',
    render(row) {
      return h(
        RouterLink,
        {
          to: `/problems/${row.id}?contestId=${detail.value?.contest.id}`,
          class: 'table-link'
        },
        () => row.title
      )
    }
  },
  { title: 'Score', key: 'score', width: 110 }
]

const scoreboardColumns = computed<DataTableColumns<ScoreboardRow>>(() => [
  {
    title: '#',
    key: 'rank',
    width: 72,
    render(_row, index) {
      return String(index + 1)
    }
  },
  { title: 'User', key: 'userName', minWidth: 160 },
  { title: 'Solved', key: 'solved', width: 110 },
  { title: 'Penalty', key: 'penalty', width: 110 },
  ...(detail.value?.problems.map((problem) => ({
    title: problem.key,
    key: problem.key,
    width: 110,
    render(row: ScoreboardRow) {
      const cell = row.problems[problem.key]
      if (!cell?.attempts && !cell?.frozenAttempts) return '-'
      if (!cell?.attempts && cell?.frozenAttempts) return h(NTag, { bordered: false }, () => '?')
      return cell.solved ? `+${cell.attempts > 1 ? cell.attempts - 1 : ''}` : `-${cell.attempts}`
    }
  })) ?? [])
])

onMounted(async () => {
  try {
    const [detailData, scoreboardData] = await Promise.all([
      apiFetch<ContestDetail>(`/api/contests/${route.params.id}`),
      apiFetch<Scoreboard>(
        `/api/contests/${route.params.id}/scoreboard${canReveal.value ? '/reveal' : ''}`
      )
    ])
    detail.value = detailData
    scoreboard.value = scoreboardData
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
      <template v-if="detail">
        <section class="page-header">
          <h1>{{ detail.contest.title }}</h1>
          <p>{{ detail.contest.description || 'Contest problem set.' }}</p>
        </section>

        <n-card :bordered="false" class="stacked-card">
          <div class="meta-row">
            <n-tag :bordered="false">{{ detail.contest.type }}</n-tag>
            <span class="muted">{{ new Date(detail.contest.startAt).toLocaleString() }}</span>
            <span class="muted">to {{ new Date(detail.contest.endAt).toLocaleString() }}</span>
            <n-tag v-if="detail.contest.freezeAt" :bordered="false" type="warning">
              freezes {{ new Date(detail.contest.freezeAt).toLocaleString() }}
            </n-tag>
          </div>
        </n-card>

        <n-data-table
          :columns="columns"
          :data="detail.problems"
          :bordered="false"
          class="stacked-card"
        />

        <n-card title="Scoreboard" :bordered="false" class="stacked-card">
          <n-alert
            v-if="scoreboard?.frozen && !scoreboard.revealed"
            type="warning"
            class="card-alert"
          >
            Scoreboard is frozen. Submissions after
            {{ scoreboard.visibleUntil ? new Date(scoreboard.visibleUntil).toLocaleString() : '' }}
            are hidden until reveal.
          </n-alert>
          <n-alert
            v-else-if="scoreboard?.frozen && scoreboard.revealed"
            type="info"
            class="card-alert"
          >
            Admin reveal view is showing final standings.
          </n-alert>
          <n-data-table
            :columns="scoreboardColumns"
            :data="scoreboard?.rows ?? []"
            :bordered="false"
          />
        </n-card>
      </template>
      <p v-else-if="error" class="form-error">{{ error }}</p>
    </n-spin>
  </main>
</template>
