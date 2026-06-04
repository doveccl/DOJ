<script setup lang="ts">
import { NButton, NCard, NInput, NSelect, NSpace, NSpin } from 'naive-ui'
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { apiFetch } from '../api'
import { useAuthStore } from '../stores/auth'

interface Problem {
  id: string
  title: string
  tags: string[]
}

interface ProblemVersion {
  id: string
  statementMarkdown: string
  timeLimitMs: number
  memoryLimitBytes: number
  outputLimitBytes: number
}

interface LanguageOption {
  label: string
  value: string
}

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const loading = ref(true)
const submitting = ref(false)
const error = ref('')
const problem = ref<Problem | null>(null)
const version = ref<ProblemVersion | null>(null)
const languageId = ref('sh')
const sourceCode = ref('#!/bin/sh\necho accepted\n')
const languageOptions = ref<LanguageOption[]>([])

const memoryMb = computed(() =>
  version.value ? Math.round(version.value.memoryLimitBytes / 1024 / 1024) : 0
)

onMounted(async () => {
  try {
    const data = await apiFetch<{ problem: Problem; version: ProblemVersion }>(
      `/api/problems/${route.params.id}`
    )
    const languages = await apiFetch<{ list: Array<{ id: string; name: string }> }>('/api/languages')
    problem.value = data.problem
    version.value = data.version
    languageOptions.value = languages.list.map((language) => ({
      label: language.name,
      value: language.id
    }))
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : String(caught)
  } finally {
    loading.value = false
  }
})

async function submit() {
  if (!auth.user || !problem.value || !version.value) return

  error.value = ''
  submitting.value = true
  try {
    await apiFetch('/api/submissions', {
      method: 'POST',
      body: JSON.stringify({
        problemId: problem.value.id,
        problemVersionId: version.value.id,
        languageId: languageId.value,
        sourceCode: sourceCode.value
      })
    })
    await router.push('/submissions')
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : String(caught)
  } finally {
    submitting.value = false
  }
}

function updateTemplate(value: string) {
  if (value === 'py') sourceCode.value = 'print("accepted")\n'
  if (value === 'sh') sourceCode.value = '#!/bin/sh\necho accepted\n'
}
</script>

<template>
  <main class="page">
    <n-spin :show="loading">
      <section v-if="problem && version" class="problem-layout">
        <div>
          <section class="page-header">
            <h1>{{ problem.title }}</h1>
            <p>{{ version.timeLimitMs }} ms / {{ memoryMb }} MB</p>
          </section>
          <n-card :bordered="false">
            <pre class="statement">{{ version.statementMarkdown }}</pre>
          </n-card>
        </div>
        <n-card title="Submit" :bordered="false">
          <n-space vertical>
            <n-select
              v-model:value="languageId"
              :options="languageOptions"
              :disabled="!auth.signedIn"
              @update:value="updateTemplate"
            />
            <n-input
              v-model:value="sourceCode"
              type="textarea"
              :autosize="{ minRows: 12, maxRows: 18 }"
              :disabled="!auth.signedIn"
            />
            <p v-if="!auth.signedIn" class="muted">Sign in to submit.</p>
            <p v-if="error" class="form-error">{{ error }}</p>
            <n-button type="primary" :disabled="!auth.signedIn" :loading="submitting" @click="submit">
              Submit
            </n-button>
          </n-space>
        </n-card>
      </section>
      <p v-else-if="error" class="form-error">{{ error }}</p>
    </n-spin>
  </main>
</template>
