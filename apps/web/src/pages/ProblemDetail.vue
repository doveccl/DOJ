<script setup lang="ts">
import {
  NButton,
  NCard,
  NCheckbox,
  NDescriptions,
  NDescriptionsItem,
  NSelect,
  NSpace,
  NSpin,
  NTag
} from 'naive-ui'
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../api'
import CodeEditor from '../components/CodeEditor.vue'
import MarkdownView from '../components/MarkdownView.vue'
import { useAuthStore } from '../stores/auth'

interface Problem {
  id: number
  title: string
  tags: string[]
  statementMarkdown: string
  timeLimitMs: number
  memoryLimitBytes: number
  solvedCount: number
  submissionCount: number
}

interface LanguageOption {
  label: string
  value: string
}

interface AppConfig {
  sourceOpenDefault: boolean
}

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { t } = useI18n()
const loading = ref(true)
const submitting = ref(false)
const error = ref('')
const problem = ref<Problem | null>(null)
const languageId = ref('cc')
const sourceOpen = ref(false)
const sourceCode = ref(
  '#include <bits/stdc++.h>\nusing namespace std;\n\nint main() {\n  long long a, b;\n  if (cin >> a >> b) cout << a + b << "\\n";\n  return 0;\n}\n'
)
const languageOptions = ref<LanguageOption[]>([])

const memoryMb = computed(() =>
  problem.value ? Math.round(problem.value.memoryLimitBytes / 1024 / 1024) : 0
)
const assignmentId = computed(() =>
  typeof route.query.assignmentId === 'string' ? route.query.assignmentId : ''
)
const contestId = computed(() =>
  typeof route.query.contestId === 'string' ? route.query.contestId : ''
)

onMounted(async () => {
  try {
    const [data, languages, config] = await Promise.all([
      apiFetch<{ problem: Problem }>(`/api/problems/${route.params.id}`),
      apiFetch<{ list: Array<{ id: string; name: string }> }>('/api/languages'),
      apiFetch<AppConfig>('/api/config')
    ])
    problem.value = data.problem
    sourceOpen.value = config.sourceOpenDefault
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
  if (!auth.user || !problem.value) return

  error.value = ''
  submitting.value = true
  try {
    await apiFetch('/api/submissions', {
      method: 'POST',
      body: JSON.stringify({
        problemId: problem.value.id,
        languageId: languageId.value,
        sourceCode: sourceCode.value,
        open: sourceOpen.value,
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
      <section v-if="problem" class="problem-layout">
        <div>
          <section class="page-header">
            <h1>{{ problem.id }}. {{ problem.title }}</h1>
            <p>{{ problem.timeLimitMs }} ms / {{ memoryMb }} MB</p>
            <p v-if="assignmentId" class="muted">
              {{ t('problemDetail.assignmentContext') }} {{ assignmentId }}
            </p>
            <p v-if="contestId" class="muted">
              {{ t('problemDetail.contestContext') }} {{ contestId }}
            </p>
          </section>
          <n-card :bordered="false" class="stacked-card">
            <n-descriptions :column="2" bordered size="small">
              <n-descriptions-item :label="t('common.time')">
                {{ problem.timeLimitMs }} ms
              </n-descriptions-item>
              <n-descriptions-item :label="t('common.memory')">
                {{ memoryMb }} MB
              </n-descriptions-item>
              <n-descriptions-item :label="t('common.solved')">
                {{ problem.solvedCount }}
              </n-descriptions-item>
              <n-descriptions-item :label="t('common.submissions')">
                {{ problem.submissionCount }}
              </n-descriptions-item>
              <n-descriptions-item :label="t('common.tags')" :span="2">
                <n-space :size="6">
                  <n-tag v-for="tag in problem.tags" :key="tag" size="small" :bordered="false">
                    {{ tag }}
                  </n-tag>
                  <span v-if="!problem.tags.length" class="muted">-</span>
                </n-space>
              </n-descriptions-item>
            </n-descriptions>
            <n-space class="problem-links">
              <n-button secondary size="small" @click="router.push('/submissions')">
                {{ t('problemDetail.viewSubmissions') }}
              </n-button>
              <n-button secondary size="small" @click="router.push('/discussion')">
                {{ t('problemDetail.viewDiscussion') }}
              </n-button>
            </n-space>
          </n-card>
          <n-card :bordered="false">
            <markdown-view :source="problem.statementMarkdown" />
          </n-card>
        </div>
        <n-card :title="t('problemDetail.submit')" :bordered="false">
          <n-space vertical>
            <template v-if="auth.signedIn">
              <n-select
                v-model:value="languageId"
                :options="languageOptions"
                @update:value="updateTemplate"
              />
              <code-editor v-model="sourceCode" :language-id="languageId" />
              <n-checkbox v-model:checked="sourceOpen">
                {{ t('problemDetail.sourceOpen') }}
              </n-checkbox>
            </template>
            <p v-else class="muted">{{ t('problemDetail.signIn') }}</p>
            <p v-if="error" class="form-error">{{ error }}</p>
            <n-button
              type="primary"
              :disabled="!auth.signedIn"
              :loading="submitting"
              @click="submit"
            >
              {{ t('problemDetail.submit') }}
            </n-button>
          </n-space>
        </n-card>
      </section>
      <p v-else-if="error" class="form-error">{{ error }}</p>
    </n-spin>
  </main>
</template>

<style scoped lang="scss">
.problem-links {
  margin-top: 12px;
}
</style>
