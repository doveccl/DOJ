<script setup lang="ts">
import { NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../api'
import { useAuthStore } from '../stores/auth'

interface Assignment {
  id: number
  title: string
  description: string
  endAt: string
  deletedAt?: string | null
}

interface AssignmentProblem {
  id?: number | null
  problemId?: number | null
  title: string | null
  unavailable?: boolean
  attempts?: number
  bestScore?: number
  ac?: boolean
  submissionId?: number | null
  submittedAt?: string | null
}

interface AssignmentDetail {
  id?: number
  title?: string
  description?: string
  endAt?: string
  assignment?: Assignment
  problems: AssignmentProblem[]
}

interface AssignmentReportRow {
  user: { id: number; name: string; avatarUrl: string }
  problems: Record<string, AssignmentProblem>
}

interface AssignmentReport {
  rows: AssignmentReportRow[]
}

const route = useRoute()
const auth = useAuthStore()
const loading = ref(true)
const reportLoading = ref(false)
const error = ref('')
const detail = ref<AssignmentDetail | null>(null)
const report = ref<AssignmentReport | null>(null)
const activeTab = ref('problems')
const { t } = useI18n()
const canManage = computed(() => auth.user?.admin ?? false)
const assignment = computed(() => detail.value?.assignment ?? {
  id: detail.value?.id ?? 0,
  title: detail.value?.title ?? '',
  description: detail.value?.description ?? '',
  endAt: detail.value?.endAt ?? '',
  deletedAt: null
})
const problems = computed(() => detail.value?.problems ?? [])

const columns = computed<DataTableColumns<AssignmentProblem>>(() => [
  {
    title: t('common.id'),
    key: 'id',
    width: 96,
    render(row) {
      return row.problemId ?? row.id ?? '-'
    }
  },
  {
    title: t('common.problem'),
    key: 'title',
    render(row) {
      const problemId = row.problemId ?? row.id
      if (!problemId || row.unavailable) return row.title ?? t('assignments.unavailable')
      return h(
        RouterLink,
        {
          to: `/problems/${problemId}?assignmentId=${assignment.value.id}`,
          class: 'table-link'
        },
        () => row.title ?? `P${problemId}`
      )
    }
  },
  {
    title: t('admin.assignments.submitted'),
    key: 'attempts',
    width: 120,
    render(row) {
      return row.attempts ?? '-'
    }
  },
  {
    title: t('submissions.score'),
    key: 'bestScore',
    width: 110,
    render(row) {
      return row.bestScore ?? '-'
    }
  },
  {
    title: t('common.status'),
    key: 'ac',
    width: 100,
    render(row) {
      if (row.ac === undefined) return '-'
      return h(NTag, { bordered: false, type: row.ac ? 'success' : 'warning' }, () =>
        row.ac ? 'AC' : '-'
      )
    }
  },
  {
    title: t('assignments.lastSubmission'),
    key: 'submittedAt',
    minWidth: 180,
    render(row) {
      if (!row.submissionId || !row.submittedAt) return '-'
      return h(
        RouterLink,
        { to: `/submissions/${row.submissionId}`, class: 'table-link' },
        () => new Date(row.submittedAt as string).toLocaleString()
      )
    }
  }
])

const reportColumns = computed<DataTableColumns<AssignmentReportRow>>(() => [
  {
    title: t('admin.assignments.student'),
    key: 'user',
    minWidth: 140,
    render(row) {
      return row.user.name
    }
  },
  ...(problems.value.map((problem) => ({
    title: `P${problem.problemId ?? problem.id}`,
    key: String(problem.problemId ?? problem.id),
    width: 120,
    render(row: AssignmentReportRow) {
      const cell = row.problems[String(problem.problemId ?? problem.id)]
      if (!cell?.attempts) return '-'
      return h(NTag, { bordered: false, type: cell.ac ? 'success' : 'warning' }, () =>
        `${cell.ac ? 'AC' : cell.bestScore} (${cell.attempts})`
      )
    }
  })) ?? [])
])

async function loadDetail() {
  loading.value = true
  error.value = ''
  try {
    detail.value = await apiFetch<AssignmentDetail>(
      canManage.value ? `/api/admin/assignments/${route.params.id}` : `/api/my/assignments/${route.params.id}`
    )
    if (canManage.value && route.query.report === '1') {
      activeTab.value = 'report'
      await loadReport()
    }
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

async function loadReport() {
  reportLoading.value = true
  error.value = ''
  try {
    report.value = await apiFetch<AssignmentReport>(`/api/admin/assignments/${route.params.id}/report`)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    reportLoading.value = false
  }
}

function handleTabUpdate(value: string) {
  activeTab.value = value
  if (value === 'report' && canManage.value && !report.value) void loadReport()
}

watch(
  () => auth.signedIn,
  (signedIn) => {
    if (signedIn) loadDetail()
  }
)

onMounted(() => {
  if (auth.signedIn) {
    loadDetail()
  } else {
    loading.value = false
  }
})
</script>

<template>
  <main class="page">
    <n-spin :show="loading">
      <template v-if="detail">
        <section class="page-header">
          <h1>{{ assignment.title }}</h1>
          <p v-if="assignment.description">{{ assignment.description }}</p>
          <p class="muted">
            {{ t('assignments.duePrefix') }}
            {{ assignment.endAt ? new Date(assignment.endAt).toLocaleString() : '-' }}
          </p>
        </section>

        <n-card :bordered="false" class="stacked-card assignment-detail-card">
          <n-tabs
            :value="activeTab"
            type="line"
            animated
            @update:value="handleTabUpdate"
          >
            <n-tab-pane name="problems" :tab="t('common.problem')">
              <n-data-table
                :columns="columns"
                :data="problems"
                :bordered="false"
                :scroll-x="760"
              />
            </n-tab-pane>
            <n-tab-pane v-if="canManage" name="report" :tab="t('admin.assignments.report')">
              <n-data-table
                :columns="reportColumns"
                :data="report?.rows ?? []"
                :bordered="false"
                :loading="reportLoading"
                :scroll-x="Math.max(520, 160 + problems.length * 120)"
              />
            </n-tab-pane>
          </n-tabs>
        </n-card>
      </template>

      <n-alert v-else-if="!auth.signedIn" type="warning" class="page-alert">
        {{ t('assignments.signIn') }}
      </n-alert>
      <n-alert v-else-if="error" type="error" class="page-alert">
        {{ error }}
      </n-alert>
    </n-spin>
  </main>
</template>
