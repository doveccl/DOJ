<script setup lang="ts">
import { NButton, NCard, NInput, NSelect, NSpace, NSpin } from 'naive-ui'
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { apiFetch } from '../api'
import { useAuthStore } from '../stores/auth'

interface Problem {
  id: number
  title: string
  tags: string[]
}

interface ProblemVersion {
  id: number
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
const languageId = ref('cpp')
const sourceCode = ref(
  '#include <bits/stdc++.h>\nusing namespace std;\n\nint main() {\n  long long a, b;\n  if (cin >> a >> b) cout << a + b << "\\n";\n  return 0;\n}\n'
)
const languageOptions = ref<LanguageOption[]>([])

const memoryMb = computed(() =>
  version.value ? Math.round(version.value.memoryLimitBytes / 1024 / 1024) : 0
)
const assignmentId = computed(() =>
  typeof route.query.assignmentId === 'string' ? route.query.assignmentId : ''
)
const contestId = computed(() =>
  typeof route.query.contestId === 'string' ? route.query.contestId : ''
)

onMounted(async () => {
  try {
    const data = await apiFetch<{ problem: Problem; version: ProblemVersion }>(
      `/api/problems/${route.params.id}`
    )
    const languages = await apiFetch<{ list: Array<{ id: string; name: string }> }>(
      '/api/languages'
    )
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
        sourceCode: sourceCode.value,
        assignmentId: assignmentId.value || undefined,
        contestId: contestId.value || undefined
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
  if (value === 'c') {
    sourceCode.value =
      '#include <stdio.h>\n\nint main(void) {\n  long long a, b;\n  if (scanf("%lld%lld", &a, &b) == 2) printf("%lld\\n", a + b);\n  return 0;\n}\n'
  }
  if (value === 'cpp') {
    sourceCode.value =
      '#include <bits/stdc++.h>\nusing namespace std;\n\nint main() {\n  long long a, b;\n  if (cin >> a >> b) cout << a + b << "\\n";\n  return 0;\n}\n'
  }
  if (value === 'py') sourceCode.value = 'a, b = map(int, input().split())\nprint(a + b)\n'
  if (value === 'sh') sourceCode.value = '#!/bin/sh\necho accepted\n'
}
</script>

<template>
  <main class="page">
    <n-spin :show="loading">
      <section v-if="problem && version" class="problem-layout">
        <div>
          <section class="page-header">
            <h1>{{ problem.id }}. {{ problem.title }}</h1>
            <p>{{ version.timeLimitMs }} ms / {{ memoryMb }} MB</p>
            <p v-if="assignmentId" class="muted">Submitting for assignment {{ assignmentId }}</p>
            <p v-if="contestId" class="muted">Submitting for contest {{ contestId }}</p>
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
            <n-button
              type="primary"
              :disabled="!auth.signedIn"
              :loading="submitting"
              @click="submit"
            >
              Submit
            </n-button>
          </n-space>
        </n-card>
      </section>
      <p v-else-if="error" class="form-error">{{ error }}</p>
    </n-spin>
  </main>
</template>
