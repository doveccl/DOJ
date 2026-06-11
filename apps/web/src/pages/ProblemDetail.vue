<script setup lang="ts">
import type { UploadCustomRequestOptions } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../api'
import CodeEditor from '../components/CodeEditor.vue'
import MarkdownEditor from '../components/MarkdownEditor.vue'
import MarkdownView from '../components/MarkdownView.vue'
import { useAuthStore } from '../stores/auth'

interface Problem {
  id: number
  title: string
  tags: string[]
  statement?: string
  mode: 'default' | 'strict' | 'custom'
  timeLimit?: number
  memoryLimit?: number
  passRate: number
  recentSubmission: SubmissionListItem | null
  visible: boolean
  deletedAt: string | null
}

interface SubmissionListItem {
  id: number
  languageId: string
  displayStatus: string
  score: number | null
  timeMs: number | null
  memoryBytes: number | null
  createdAt: string
}

interface LanguageOption {
  label: string
  value: string
}

interface AppConfig {
  publicCode?: boolean
}

interface ProblemAsset {
  path: string
  name: string
  section: 'data' | 'assets' | 'root'
  size: number
  updatedAt: string
}

type ProblemAssetSection = ProblemAsset['section']

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { t } = useI18n()
const loading = ref(true)
const submitting = ref(false)
const saving = ref(false)
const editMode = ref(false)
const adminFullView = ref(true)
const error = ref('')
const problem = ref<Problem | null>(null)
const languageId = ref('cpp')
const sourceOpen = ref(false)
const codeText = ref(
  '#include <bits/stdc++.h>\nusing namespace std;\n\nint main() {\n  long long a, b;\n  if (cin >> a >> b) cout << a + b << "\\n";\n  return 0;\n}\n'
)
const languageOptions = ref<LanguageOption[]>([])
const assets = ref<ProblemAsset[]>([])
const assetPath = ref('assets/')
const assetContent = ref('')
const selectedAssetPath = ref('')
const assetSaving = ref(false)
const editForm = reactive({
  title: '',
  tags: [] as string[],
  mode: 'default' as 'default' | 'strict' | 'custom',
  visible: false,
  timeLimit: 1000,
  memoryLimitMb: 256,
  statement: ''
})

const canManage = computed(() => auth.user?.admin ?? false)
const memoryMb = computed(() =>
  problem.value ? Math.round((problem.value.memoryLimit || 0) / 1024 / 1024) : 0
)
const timeLimit = computed(() => problem.value?.timeLimit ?? 0)
const statement = computed(() => problem.value?.statement ?? '')
const assignmentId = computed(() =>
  typeof route.query.assignmentId === 'string' ? route.query.assignmentId : ''
)
const contestId = computed(() =>
  typeof route.query.contestId === 'string' ? route.query.contestId : ''
)
const assetSections = computed<Array<{ key: ProblemAssetSection; title: string; hint: string; draftPath: string }>>(() => {
  const sections: Array<{ key: ProblemAssetSection; title: string; hint: string; draftPath: string }> = [
    { key: 'data', title: t('admin.problems.dataAssets'), hint: 'data/*.in, data/*.out', draftPath: 'data/1.in' },
    { key: 'assets', title: t('admin.problems.publicAssets'), hint: 'assets/filename', draftPath: 'assets/readme.txt' }
  ]
  if (problem.value?.mode === 'custom') {
    sections.push({ key: 'root', title: t('admin.problems.rootResources'), hint: 'Dockerfile, checker.cc', draftPath: 'Dockerfile' })
  }
  return sections
})

onMounted(async () => {
  try {
    const [data, languages, config] = await Promise.all([
      apiFetch<Problem>(`/api/problems/${route.params.id}`),
      apiFetch<Array<{ id: string; name: string }>>('/api/languages'),
      apiFetch<AppConfig>('/api/config')
    ])
    problem.value = data
    sourceOpen.value = config.publicCode ?? false
    syncEditForm(data)
    languageOptions.value = languages.map((language) => ({
      label: language.name,
      value: language.id
    }))
    if (auth.user?.admin) await loadAssets()
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
        code: codeText.value,
        public: sourceOpen.value,
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
    codeText.value =
      '#include <stdio.h>\n\nint main(void) {\n  long long a, b;\n  if (scanf("%lld%lld", &a, &b) == 2) printf("%lld\\n", a + b);\n  return 0;\n}\n'
  }
  if (value === 'cpp') {
    codeText.value =
      '#include <bits/stdc++.h>\nusing namespace std;\n\nint main() {\n  long long a, b;\n  if (cin >> a >> b) cout << a + b << "\\n";\n  return 0;\n}\n'
  }
  if (value === 'py') codeText.value = 'a, b = map(int, input().split())\nprint(a + b)\n'
  if (value === 'sh') codeText.value = '#!/bin/sh\necho accepted\n'
}

function syncEditForm(next: Problem) {
  editForm.title = next.title
  editForm.tags = [...next.tags]
  editForm.mode = next.mode
  editForm.visible = next.visible
  editForm.timeLimit = next.timeLimit ?? 1000
  editForm.memoryLimitMb = Math.round((next.memoryLimit || 0) / 1024 / 1024) || 256
  editForm.statement = next.statement ?? ''
}

async function saveProblem() {
  if (!problem.value) return
  saving.value = true
  error.value = ''
  try {
    const id = problem.value.id
    const [updated, statementResult] = await Promise.all([
      apiFetch<Problem>(`/api/admin/problems/${id}`, {
        method: 'PATCH',
        body: JSON.stringify({
          title: editForm.title,
          tags: editForm.tags,
          mode: editForm.mode,
          visible: editForm.visible,
          timeLimit: editForm.timeLimit,
          memoryLimit: editForm.memoryLimitMb * 1024 * 1024
        })
      }),
      apiFetch<{ markdown: string }>(`/api/admin/problems/${id}/statement`, {
        method: 'PUT',
        body: JSON.stringify({ markdown: editForm.statement })
      })
    ])
    problem.value = { ...updated, statement: statementResult.markdown }
    syncEditForm(problem.value)
    editMode.value = false
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : String(caught)
  } finally {
    saving.value = false
  }
}

async function toggleProblemDeleted() {
  if (!problem.value) return
  saving.value = true
  error.value = ''
  try {
    const updated = await apiFetch<Problem>(
      problem.value.deletedAt ? `/api/admin/problems/${problem.value.id}/restore` : `/api/admin/problems/${problem.value.id}`,
      { method: problem.value.deletedAt ? 'POST' : 'DELETE' }
    )
    if (problem.value.deletedAt) {
      problem.value = updated
      syncEditForm(updated)
    } else {
      problem.value = { ...problem.value, deletedAt: new Date().toISOString() }
    }
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : String(caught)
  } finally {
    saving.value = false
  }
}

async function loadAssets() {
  if (!problem.value) return
  try {
    assets.value = await apiFetch<ProblemAsset[]>(`/api/admin/problems/${problem.value.id}/assets`)
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : String(caught)
  }
}

async function uploadAsset({ file, onFinish, onError }: UploadCustomRequestOptions) {
  if (!file.file || !problem.value) {
    onError()
    return
  }
  try {
    const body = new FormData()
    body.append('file', file.file)
    const targetPath = assetPath.value.endsWith('/')
      ? `${assetPath.value}${file.name}`
      : assetPath.value || `assets/${file.name}`
    body.append('path', targetPath)
    await apiFetch(`/api/admin/problems/${problem.value.id}/assets/upload`, {
      method: 'POST',
      body
    })
    await loadAssets()
    onFinish()
  } catch {
    onError()
  }
}

async function loadAssetContent(path: string) {
  if (!problem.value) return
  selectedAssetPath.value = path
  assetContent.value = ''
  try {
    const content = await apiFetch<{ content: string; encoding: 'utf8' | 'base64'; text: boolean }>(
      `/api/admin/problems/${problem.value.id}/assets/content?path=${encodeURIComponent(path)}`
    )
    assetContent.value = content.encoding === 'utf8' ? content.content : ''
    if (!content.text) error.value = t('admin.problems.binaryEditHint')
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : String(caught)
  }
}

async function saveAssetContent() {
  if (!problem.value || !selectedAssetPath.value) return
  assetSaving.value = true
  error.value = ''
  try {
    await apiFetch(`/api/admin/problems/${problem.value.id}/assets/content`, {
      method: 'PUT',
      body: JSON.stringify({
        path: selectedAssetPath.value,
        content: assetContent.value,
        encoding: 'utf8'
      })
    })
    await loadAssets()
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : String(caught)
  } finally {
    assetSaving.value = false
  }
}

function newAsset(path: string) {
  selectedAssetPath.value = path
  assetContent.value = ''
}

function assetsBySection(section: ProblemAssetSection) {
  return assets.value.filter((asset) => asset.section === section)
}

function setAssetDraft(path: string) {
  assetPath.value = path
  newAsset(path)
}

function formatDate(value: string) {
  return new Date(value).toLocaleString()
}

async function deleteAsset(path: string) {
  if (!problem.value) return
  try {
    await apiFetch(
      `/api/admin/problems/${problem.value.id}/assets?path=${encodeURIComponent(path)}`,
      { method: 'DELETE' }
    )
    if (selectedAssetPath.value === path) {
      selectedAssetPath.value = ''
      assetContent.value = ''
    }
    await loadAssets()
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : String(caught)
  }
}
</script>

<template>
  <main class="page">
    <n-spin :show="loading">
      <section v-if="problem" class="problem-detail-page">
        <section class="page-header">
          <h1>{{ problem.id }}. {{ problem.title }}</h1>
          <p>{{ timeLimit }} ms / {{ memoryMb }} MB</p>
          <p v-if="assignmentId" class="muted">
            {{ t('problemDetail.assignmentContext') }} {{ assignmentId }}
          </p>
          <p v-if="contestId" class="muted">
            {{ t('problemDetail.contestContext') }} {{ contestId }}
          </p>
        </section>

        <section class="problem-top-grid">
          <n-card :bordered="false" class="statement-card">
            <markdown-view :source="statement" />
          </n-card>

          <n-card :bordered="false" class="meta-card">
            <n-descriptions :column="1" bordered size="small">
              <n-descriptions-item :label="t('common.time')">
                {{ timeLimit }} ms
              </n-descriptions-item>
              <n-descriptions-item :label="t('common.memory')">
                {{ memoryMb }} MB
              </n-descriptions-item>
              <n-descriptions-item :label="t('problems.passRate')">
                {{ ((problem.passRate ?? 0) * 100).toFixed(1) }}%
              </n-descriptions-item>
              <n-descriptions-item :label="t('common.tags')">
                <n-space :size="6">
                  <n-tag v-for="item in problem.tags" :key="item" size="small" :bordered="false">
                    {{ item }}
                  </n-tag>
                  <span v-if="!problem.tags.length" class="muted">-</span>
                </n-space>
              </n-descriptions-item>
              <n-descriptions-item :label="t('problemDetail.recentSubmission')">
                <RouterLink
                  v-if="problem.recentSubmission"
                  class="table-link"
                  :to="`/submissions/${problem.recentSubmission.id}`"
                >
                  #{{ problem.recentSubmission.id }}
                  {{ problem.recentSubmission.displayStatus }}
                  <template v-if="problem.recentSubmission.score !== null">
                    / {{ problem.recentSubmission.score }}
                  </template>
                </RouterLink>
                <span v-else class="muted">{{ t('problemDetail.noRecentSubmission') }}</span>
              </n-descriptions-item>
              <n-descriptions-item v-if="canManage && adminFullView" :label="t('admin.problems.visible')">
                <n-tag :bordered="false" :type="problem.visible ? 'success' : 'default'">
                  {{ problem.visible ? t('admin.problems.yes') : t('admin.problems.no') }}
                </n-tag>
              </n-descriptions-item>
              <n-descriptions-item v-if="canManage && adminFullView" :label="t('admin.problems.deleted')">
                <n-tag :bordered="false" :type="problem.deletedAt ? 'error' : 'success'">
                  {{ problem.deletedAt ? formatDate(problem.deletedAt) : t('admin.problems.no') }}
                </n-tag>
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
        </section>

        <div v-if="canManage" class="admin-mode-bar">
          <n-space justify="space-between" align="center">
            <strong>{{ t('admin.fullView') }}</strong>
            <n-switch v-model:value="adminFullView" />
          </n-space>
        </div>

        <n-card v-if="canManage && adminFullView" :title="t('admin.problems.edit')" :bordered="false" class="stacked-card">
          <template #header-extra>
            <n-space>
              <n-button
                size="small"
                tertiary
                :type="problem.deletedAt ? 'success' : 'error'"
                :loading="saving"
                @click="toggleProblemDeleted"
              >
                {{ problem.deletedAt ? t('admin.restore') : t('admin.delete') }}
              </n-button>
              <n-button v-if="!editMode" size="small" secondary @click="editMode = true">
                {{ t('admin.edit') }}
              </n-button>
              <template v-else>
                <n-button size="small" @click="editMode = false">{{ t('admin.cancel') }}</n-button>
                <n-button size="small" type="primary" :loading="saving" @click="saveProblem">
                  {{ t('admin.save') }}
                </n-button>
              </template>
            </n-space>
          </template>

          <template v-if="editMode">
            <n-form :model="editForm" label-placement="top">
              <n-form-item :label="t('common.title')">
                <n-input v-model:value="editForm.title" />
              </n-form-item>
              <n-form-item :label="t('common.tags')">
                <n-dynamic-tags v-model:value="editForm.tags" />
              </n-form-item>
              <div class="form-grid two">
                <n-form-item :label="t('admin.problems.mode')">
                  <n-select
                    v-model:value="editForm.mode"
                    :options="[
                      { label: 'default', value: 'default' },
                      { label: 'strict', value: 'strict' },
                      { label: 'custom', value: 'custom' }
                    ]"
                  />
                </n-form-item>
                <n-form-item :label="t('admin.problems.visible')">
                  <n-switch v-model:value="editForm.visible" />
                </n-form-item>
                <n-form-item :label="t('admin.problems.timeMs')">
                  <n-input-number v-model:value="editForm.timeLimit" :min="100" class="full-width" />
                </n-form-item>
                <n-form-item :label="t('admin.problems.memoryMb')">
                  <n-input-number v-model:value="editForm.memoryLimitMb" :min="16" class="full-width" />
                </n-form-item>
              </div>
              <n-form-item :label="t('admin.problems.statement')">
                <markdown-editor
                  v-model="editForm.statement"
                  :problem-id="problem.id"
                  upload-enabled
                />
              </n-form-item>
            </n-form>
          </template>
          <template v-else>
            <p class="muted">{{ t('problemDetail.adminHint') }}</p>
          </template>

          <div class="asset-panel">
            <div class="asset-panel-header">
              <div>
                <strong>{{ t('admin.problems.assets') }}</strong>
                <p class="muted">data/ 用于评测，assets/ 用于题面附件。</p>
              </div>
              <n-space class="asset-actions">
                <n-input
                  v-model:value="assetPath"
                  size="small"
                  class="asset-path-input"
                  :placeholder="t('admin.problems.packageNewFilePath')"
                />
                <n-button size="small" secondary @click="newAsset(assetPath)">
                  {{ t('admin.problems.packageNewFile') }}
                </n-button>
                <n-upload :custom-request="uploadAsset" :show-file-list="false" accept="*">
                  <n-button size="small" secondary>{{ t('admin.upload') }}</n-button>
                </n-upload>
              </n-space>
            </div>

            <div class="asset-sections">
              <section v-for="section in assetSections" :key="section.key" class="asset-section">
                <n-space justify="space-between" align="center">
                  <div>
                    <strong>{{ section.title }}</strong>
                    <p class="muted">{{ section.hint }}</p>
                  </div>
                  <n-button size="tiny" secondary @click="setAssetDraft(section.draftPath)">
                    {{ t('admin.problems.packageNewFile') }}
                  </n-button>
                </n-space>
                <div v-if="assetsBySection(section.key).length" class="asset-list">
                  <div
                    v-for="asset in assetsBySection(section.key)"
                    :key="asset.path"
                    class="asset-row"
                    :class="{ selected: selectedAssetPath === asset.path }"
                  >
                    <n-button text size="small" @click="loadAssetContent(asset.path)">
                      {{ asset.path }}
                    </n-button>
                    <span class="muted">{{ Math.round(asset.size / 1024) }} KB</span>
                    <n-button size="tiny" tertiary type="error" @click="deleteAsset(asset.path)">
                      {{ t('admin.problems.packageDelete') }}
                    </n-button>
                  </div>
                </div>
                <p v-else class="muted">{{ t('admin.problems.packageEmpty') }}</p>
              </section>
              <p v-if="problem.mode !== 'custom'" class="muted">{{ t('admin.problems.rootCustomOnly') }}</p>
            </div>

            <div v-if="selectedAssetPath" class="asset-editor">
              <div class="asset-editor-header">
                <span class="muted">{{ t('admin.problems.packagePath') }}</span>
                <n-input v-model:value="selectedAssetPath" />
              </div>
              <code-editor v-model="assetContent" />
              <n-space justify="end">
                <n-button type="primary" :loading="assetSaving" @click="saveAssetContent">
                  {{ t('admin.problems.packageSave') }}
                </n-button>
              </n-space>
            </div>
          </div>
        </n-card>

        <n-card :title="t('problemDetail.submit')" :bordered="false" class="submit-card">
          <n-space vertical>
            <template v-if="auth.signedIn">
              <n-select
                v-model:value="languageId"
                :options="languageOptions"
                @update:value="updateTemplate"
              />
              <code-editor v-model="codeText" :language-id="languageId" />
              <n-checkbox v-model:checked="sourceOpen">
                {{ t('problemDetail.sourceOpen') }}
              </n-checkbox>
              <n-space justify="end" class="submit-actions">
                <n-button
                  type="primary"
                  :loading="submitting"
                  @click="submit"
                >
                  {{ t('problemDetail.submit') }}
                </n-button>
              </n-space>
            </template>
            <n-alert v-else type="info" :show-icon="false">
              {{ t('problemDetail.signIn') }}
            </n-alert>
            <p v-if="error" class="form-error">{{ error }}</p>
          </n-space>
        </n-card>
      </section>
      <p v-else-if="error" class="form-error">{{ error }}</p>
    </n-spin>
  </main>
</template>

<style scoped lang="scss">
.problem-detail-page {
  display: grid;
  gap: 16px;
}

.problem-top-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(280px, 360px);
  gap: 24px;
  align-items: start;
}

.statement-card,
.meta-card,
.submit-card {
  min-width: 0;
}

.problem-links {
  margin-top: 12px;
}

.admin-mode-bar {
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: color-mix(in srgb, var(--surface-bg) 88%, var(--brand) 12%);
}

.asset-panel {
  display: grid;
  gap: 14px;
  margin-top: 16px;
}

.asset-panel-header {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  justify-content: space-between;
}

.asset-panel-header p {
  margin-top: 4px;
}

.asset-actions {
  justify-content: flex-end;
}

.asset-list {
  display: grid;
  gap: 8px;
  margin-top: 10px;
}

.asset-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  gap: 10px;
  align-items: center;
  padding: 8px 10px;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  background: color-mix(in srgb, var(--surface-bg) 94%, var(--text-color) 6%);
}

.asset-row.selected {
  border-color: var(--brand);
  background: var(--brand-soft);
}

.asset-sections {
  display: grid;
  gap: 12px;
}

.asset-section {
  padding: 12px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
}

.asset-path-input {
  width: 220px;
}

.asset-editor {
  display: grid;
  gap: 10px;
  padding-top: 4px;
}

.asset-editor-header {
  display: grid;
  gap: 6px;
}

.submit-actions {
  margin-top: 4px;
}

@media (max-width: 860px) {
  .problem-top-grid {
    grid-template-columns: 1fr;
  }

  .asset-panel-header {
    display: grid;
  }

  .asset-actions,
  .asset-path-input {
    width: 100%;
  }

  .asset-row {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
