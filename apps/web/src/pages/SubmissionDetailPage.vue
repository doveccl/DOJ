<script setup lang="ts">
import { NButton, NCard, NDataTable, NDescriptions, NDescriptionsItem, NSpin, NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { computed, h, onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../api'
import CodeEditor from '../components/CodeEditor.vue'
import MarkdownView from '../components/MarkdownView.vue'

interface Submission {
  id: number
  languageId: string
  sourceCode: string
  status: string
  timeMs: number
  memoryBytes: number
  message: string
  contestId: number | null
  restricted: boolean
  cases: SubmissionCase[]
}

interface SubmissionCase {
  caseIndex: number
  status: string
  timeMs: number
  memoryBytes: number
  message: string
}

interface CoachingSession {
  id: number
  responseMarkdown: string
}

const route = useRoute()
const loading = ref(true)
const coachingLoading = ref(false)
const error = ref('')
const coaching = ref('')
const submission = ref<Submission | null>(null)
const { t } = useI18n()
let timer: number | undefined

const statusType: Record<string, 'success' | 'warning' | 'error' | 'info'> = {
  AC: 'success',
  WAITING: 'info',
  JUDGING: 'info',
  CE: 'warning',
  WA: 'error',
  RE: 'error',
  TLE: 'error',
  MLE: 'error',
  OLE: 'error',
  SE: 'error'
}

const canCoach = computed(() => {
  if (submission.value?.contestId || submission.value?.restricted) return false
  const status = submission.value?.status
  return !!status && !['AC', 'WAITING', 'JUDGING', 'FROZEN'].includes(status)
})

const caseColumns = computed<DataTableColumns<SubmissionCase>>(() => [
  { title: '#', key: 'caseIndex', width: 72 },
  {
    title: t('common.status'),
    key: 'status',
    width: 120,
    render(row) {
      return h(
        NTag,
        { bordered: false, type: statusType[row.status] ?? 'default' },
        () => row.status
      )
    }
  },
  { title: t('common.time'), key: 'timeMs', width: 120 },
  {
    title: t('common.memory'),
    key: 'memoryBytes',
    width: 140,
    render(row) {
      return `${Math.round(row.memoryBytes / 1024)} KB`
    }
  },
  {
    title: t('common.message'),
    key: 'message',
    ellipsis: {
      tooltip: true
    }
  }
])

onMounted(async () => {
  await load()
  timer = window.setInterval(() => {
    if (['WAITING', 'JUDGING'].includes(submission.value?.status ?? '')) {
      load(false)
    }
  }, 2000)
})

onUnmounted(() => {
  if (timer) window.clearInterval(timer)
})

async function load(showLoading = true) {
  if (showLoading) loading.value = true
  try {
    submission.value = await apiFetch<Submission>(`/api/submissions/${route.params.id}`)
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : String(caught)
  } finally {
    loading.value = false
  }
}

async function getCoaching() {
  if (!submission.value) return

  coachingLoading.value = true
  error.value = ''
  try {
    const session = await apiFetch<CoachingSession>(
      `/api/submissions/${submission.value.id}/coach`,
      {
        method: 'POST'
      }
    )
    coaching.value = session.responseMarkdown
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : String(caught)
  } finally {
    coachingLoading.value = false
  }
}
</script>

<template>
  <main class="page">
    <n-spin :show="loading">
      <section v-if="submission" class="submission-layout">
        <div>
          <section class="page-header">
            <h1>{{ t('submissions.detailTitle') }} {{ submission.id }}</h1>
            <p>{{ submission.languageId }}</p>
          </section>
          <n-card :bordered="false">
            <n-descriptions label-placement="left" bordered :column="2">
              <n-descriptions-item :label="t('common.status')">
                <n-tag :bordered="false">{{ submission.status }}</n-tag>
              </n-descriptions-item>
              <n-descriptions-item :label="t('common.time')">
                {{ submission.timeMs }} ms
              </n-descriptions-item>
              <n-descriptions-item :label="t('common.memory')">
                {{ Math.round(submission.memoryBytes / 1024) }} KB
              </n-descriptions-item>
              <n-descriptions-item :label="t('submissions.contest')">
                {{ submission.contestId ? t('submissions.yes') : t('submissions.no') }}
              </n-descriptions-item>
            </n-descriptions>
          </n-card>
          <n-card
            v-if="submission.sourceCode"
            :title="t('submissions.source')"
            :bordered="false"
            class="stacked-card"
          >
            <code-editor
              :model-value="submission.sourceCode"
              :language-id="submission.languageId"
              readonly
            />
          </n-card>
          <n-card
            v-else-if="submission.restricted"
            :title="t('submissions.source')"
            :bordered="false"
            class="stacked-card"
          >
            <p class="muted">{{ t('submissions.restricted') }}</p>
          </n-card>
          <n-card
            v-if="submission.message"
            :title="t('submissions.judgeMessage')"
            :bordered="false"
            class="stacked-card"
          >
            <pre class="code-block">{{ submission.message }}</pre>
          </n-card>
          <n-card
            v-if="submission.cases?.length"
            :title="t('submissions.testCases')"
            :bordered="false"
            class="stacked-card"
          >
            <n-data-table :columns="caseColumns" :data="submission.cases" :bordered="false" />
          </n-card>
        </div>
        <n-card :title="t('submissions.aiCoaching')" :bordered="false">
          <p v-if="!canCoach" class="muted">
            {{ t('submissions.coachingUnavailable') }}
          </p>
          <p v-if="error" class="form-error">{{ error }}</p>
          <n-button
            type="primary"
            :disabled="!canCoach"
            :loading="coachingLoading"
            @click="getCoaching"
          >
            {{ t('submissions.getCoaching') }}
          </n-button>
          <markdown-view v-if="coaching" :source="coaching" class="coaching-output" />
        </n-card>
      </section>
      <p v-else-if="error" class="form-error">{{ error }}</p>
    </n-spin>
  </main>
</template>
