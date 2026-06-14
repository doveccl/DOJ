<script setup lang="ts">
import type { SelectOption, SelectRenderLabel, UploadCustomRequestOptions } from 'naive-ui'
import {
  AddOutline,
  CloseOutline,
  CreateOutline,
  DownloadOutline,
  EyeOffOutline,
  EyeOutline,
  FolderOpenOutline,
  SaveOutline,
  TrashOutline
} from '@vicons/ionicons5'
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
  solvedCount: number
  attemptedCount: number
  submissionCount: number
  passRate: number
  discussionCount: number
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
  contentType: string
  text: boolean
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
const showManageModal = ref(false)
const showAssetEditor = ref(false)
const error = ref('')
const problem = ref<Problem | null>(null)
const languageId = ref('cpp')
const sourceOpen = ref(false)
const codeText = ref(
  '#include <bits/stdc++.h>\nusing namespace std;\n\nint main() {\n  long long a, b;\n  if (cin >> a >> b) cout << a + b << "\\n";\n  return 0;\n}\n'
)
const languageOptions = ref<LanguageOption[]>([])
const assets = ref<ProblemAsset[]>([])
const selectedAsset = ref<ProblemAsset | null>(null)
const assetContent = ref('')
const assetSaving = ref(false)
const dragSection = ref<ProblemAssetSection | null>(null)
const editForm = reactive({
  title: '',
  tags: [] as string[],
  mode: 'default' as 'default' | 'strict' | 'custom',
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
const dataCaseCount = computed(() => countDataCases(assets.value.filter((asset) => asset.section === 'data')))
const dataAssetSizeText = computed(() =>
  formatBytes(assets.value.filter((asset) => asset.section === 'data').reduce((total, asset) => total + asset.size, 0))
)
const modeOptions = computed(() => [
  {
    label: t('admin.problems.modeDefault'),
    value: 'default',
    hint: t('admin.problems.modeDefaultHint')
  },
  {
    label: t('admin.problems.modeStrict'),
    value: 'strict',
    hint: t('admin.problems.modeStrictHint')
  },
  {
    label: t('admin.problems.modeCustom'),
    value: 'custom',
    hint: t('admin.problems.modeCustomHint')
  }
])
const limitText = computed(() => `${timeLimit.value}ms / ${memoryMb.value}MB`)
const passRateText = computed(() => {
  const solved = problem.value?.solvedCount ?? 0
  const submissions = problem.value?.submissionCount ?? 0
  const percent = submissions > 0 ? (solved / submissions) * 100 : 0
  return `${solved}/${submissions} (${percent.toFixed(0)}%)`
})
const recentSubmissionText = computed(() => {
  const recent = problem.value?.recentSubmission
  if (!recent) return t('problemDetail.noRecentSubmission')
  return recent.score === null ? recent.displayStatus : `${recent.displayStatus} / ${recent.score}`
})
const assignmentId = computed(() =>
  typeof route.query.assignmentId === 'string' ? route.query.assignmentId : ''
)
const contestId = computed(() =>
  typeof route.query.contestId === 'string' ? route.query.contestId : ''
)
const assetSections = computed<Array<{ key: ProblemAssetSection; title: string }>>(() => {
  const sections: Array<{ key: ProblemAssetSection; title: string }> = [
    { key: 'data', title: t('admin.problems.dataAssets') }
  ]
  if (problem.value?.mode === 'custom') {
    sections.push({ key: 'root', title: t('admin.problems.judgerResources') })
  }
  return sections
})
const assetEditorLanguage = computed(() => {
  const path = selectedAsset.value?.path.toLowerCase() ?? ''
  if (path.endsWith('.cc') || path.endsWith('.cpp') || path.endsWith('.hpp') || path.endsWith('.h')) return 'cpp'
  if (path.endsWith('.c')) return 'c'
  if (path.endsWith('.go')) return 'go'
  if (path.endsWith('.rs')) return 'rust'
  if (path.endsWith('.java')) return 'java'
  if (path.endsWith('.py')) return 'py'
  if (path.endsWith('.sh')) return 'sh'
  if (path.endsWith('.js')) return 'js'
  if (path.endsWith('.ts')) return 'ts'
  if (path.endsWith('.json')) return 'json'
  if (path.endsWith('.yaml') || path.endsWith('.yml')) return 'yaml'
  if (path.endsWith('.md')) return 'markdown'
  if (path.endsWith('dockerfile')) return 'dockerfile'
  return 'plaintext'
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

function cancelEdit() {
  if (problem.value) syncEditForm(problem.value)
  editMode.value = false
}

async function updateProblemMode(mode: 'default' | 'strict' | 'custom') {
  if (!problem.value) return
  if (mode === problem.value.mode) return
  const previous = problem.value.mode
  problem.value = { ...problem.value, mode }
  editForm.mode = mode
  saving.value = true
  error.value = ''
  try {
    const updated = await apiFetch<Problem>(`/api/admin/problems/${problem.value.id}`, {
      method: 'PATCH',
      body: JSON.stringify({
        mode
      })
    })
    problem.value = { ...updated, statement: problem.value.statement }
    syncEditForm(problem.value)
    await loadAssets()
  } catch (caught) {
    if (problem.value) problem.value = { ...problem.value, mode: previous }
    editForm.mode = previous
    error.value = caught instanceof Error ? caught.message : String(caught)
  } finally {
    saving.value = false
  }
}

function openManageModal() {
  if (problem.value) syncEditForm(problem.value)
  showManageModal.value = true
}

async function toggleProblemVisible() {
  if (!problem.value) return
  const previous = problem.value.visible
  problem.value = { ...problem.value, visible: !previous }
  try {
    const updated = await apiFetch<Problem>(`/api/admin/problems/${problem.value.id}`, {
      method: 'PATCH',
      body: JSON.stringify({ visible: !previous })
    })
    problem.value = { ...updated, statement: problem.value.statement }
  } catch (caught) {
    if (problem.value) problem.value = { ...problem.value, visible: previous }
    error.value = caught instanceof Error ? caught.message : String(caught)
  }
}

async function toggleProblemDeleted() {
  if (!problem.value) return
  saving.value = true
  error.value = ''
  try {
    await apiFetch(`/api/admin/problems/${problem.value.id}`, { method: 'DELETE' })
    problem.value = { ...problem.value, deletedAt: new Date().toISOString() }
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

async function ensureCustomJudgeTemplates() {
  if (!problem.value) return
  error.value = ''
  const existing = new Set(assets.value.map((asset) => asset.path))
  const templates = [
    {
      path: 'Dockerfile',
      content: [
        'FROM gcc:13',
        'WORKDIR /judge',
        'COPY main.cc .',
        'RUN g++ -std=c++17 -O2 -pipe -static -s -o judge main.cc',
        'CMD ["./judge"]',
        ''
      ].join('\n')
    },
    {
      path: 'main.cc',
      content: [
        '#include <bits/stdc++.h>',
        'using namespace std;',
        '',
        'string readFile(const char* path) {',
        '  ifstream in(path, ios::binary);',
        '  return string((istreambuf_iterator<char>(in)), istreambuf_iterator<char>());',
        '}',
        '',
        'string normalize(string value) {',
        '  while (!value.empty() && (value.back() == \'\\n\' || value.back() == \'\\r\')) value.pop_back();',
        '  stringstream input(value);',
        '  string line, output;',
        '  while (getline(input, line)) {',
        '    while (!line.empty() && isspace(static_cast<unsigned char>(line.back()))) line.pop_back();',
        '    output += line + \'\\n\';',
        '  }',
        '  return output;',
        '}',
        '',
        'int main() {',
        '  // A custom judger talks to the submission through stdin/stdout.',
        '  // INPUT and OUT point to the current test case files under /data.',
        '  const char* inputPath = getenv("INPUT");',
        '  const char* answerPath = getenv("OUT");',
        '  if (!inputPath || !answerPath) {',
        '    cerr << "Missing INPUT or OUT";',
        '    return 8; // SE',
        '  }',
        '',
        '  cout << readFile(inputPath) << flush;',
        '  string contestant((istreambuf_iterator<char>(cin)), istreambuf_iterator<char>());',
        '  string expected = readFile(answerPath);',
        '',
        '  if (normalize(contestant) == normalize(expected)) return 0; // AC',
        '  cerr << "Answer differs from expected output";',
        '  return 1; // WA',
        '}',
        ''
      ].join('\n')
    }
  ]
  try {
    for (const template of templates) {
      if (existing.has(template.path)) continue
      await saveAssetText(template.path, template.content)
    }
    await loadAssets()
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : String(caught)
  }
}

async function createDataCase() {
  error.value = ''
  const caseNo = nextDataCaseNo()
  try {
    await saveAssetText(`data/in${caseNo}.txt`, '')
    await saveAssetText(`data/ans${caseNo}.txt`, '')
    await loadAssets()
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : String(caught)
  }
}

function nextDataCaseNo() {
  const numbers = assets.value
    .filter((asset) => asset.section === 'data')
    .flatMap((asset) => asset.name.match(/\d+/g)?.map(Number) ?? [])
    .filter(Number.isFinite)
  return (numbers.length ? Math.max(...numbers) : 0) + 1
}

async function downloadDataAssets() {
  if (!problem.value) return
  error.value = ''
  try {
    const token = localStorage.getItem('doj.token')
    const response = await fetch(`/api/admin/problems/${problem.value.id}/assets/data.zip`, {
      headers: token ? { authorization: `Bearer ${token}` } : undefined
    })
    if (!response.ok) throw new Error(`${response.status} ${response.statusText}`)
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `P${problem.value.id}-data.zip`
    link.click()
    setTimeout(() => URL.revokeObjectURL(url), 0)
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : String(caught)
  }
}

const renderModeLabel: SelectRenderLabel = (option) => {
  const item = option as SelectOption & { hint?: string }
  if (!item.hint) return String(item.label ?? '')
  return h('div', { class: 'mode-option' }, [
    h('span', { class: 'mode-option-label' }, String(item.label ?? '')),
    h('small', { class: 'mode-option-hint' }, item.hint)
  ])
}

function uploadAssetForSection(section: ProblemAssetSection) {
  return async ({ file, onFinish, onError }: UploadCustomRequestOptions) => {
    if (!file.file || !problem.value) {
      onError()
      return
    }
    try {
      await uploadAssetFile(section, file.file)
      await loadAssets()
      onFinish()
    } catch {
      onError()
    }
  }
}

async function uploadDroppedFiles(section: ProblemAssetSection, event: DragEvent) {
  dragSection.value = null
  const files = [...(event.dataTransfer?.files ?? [])]
  if (!files.length) return
  error.value = ''
  try {
    await Promise.all(files.map((file) => uploadAssetFile(section, file)))
    await loadAssets()
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : String(caught)
  }
}

async function uploadAssetFile(section: ProblemAssetSection, file: File) {
  if (!problem.value) return
  const body = new FormData()
  body.append('file', file)
  body.append('path', `${assetUploadPrefix(section)}${file.name}`)
  await apiFetch(`/api/admin/problems/${problem.value.id}/assets/upload`, {
    method: 'POST',
    body
  })
}

function assetUploadPrefix(section: ProblemAssetSection) {
  if (section === 'data') return 'data/'
  if (section === 'assets') return 'assets/'
  return ''
}

function assetsBySection(section: ProblemAssetSection) {
  return assets.value.filter((asset) => asset.section === section)
}

function formatDate(value: string) {
  return new Date(value).toLocaleString()
}

function countDataCases(items: ProblemAsset[]) {
  const groups = new Map<string, { input: boolean; output: boolean }>()
  for (const item of items) {
    const name = item.name.toLowerCase()
    const number = name.match(/\d+/)?.[0]
    if (!number) continue
    const group = groups.get(number) ?? { input: false, output: false }
    if (/(\.in$|^in\d*|input)/.test(name)) group.input = true
    if (/(\.out$|^out\d*|^ans\d*|answer|output)/.test(name)) group.output = true
    groups.set(number, group)
  }
  return [...groups.values()].filter((group) => group.input && group.output).length
}

function formatBytes(bytes: number) {
  if (bytes < 1024) return `${bytes}B`
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)}KB`
  return `${(bytes / 1024 / 1024).toFixed(1)}MB`
}

function visibilityLabel(visible: boolean) {
  return visible ? t('admin.problems.publicVisibleShort') : t('admin.problems.publicHiddenShort')
}

async function deleteAsset(path: string) {
  if (!problem.value) return
  try {
    await apiFetch(
      `/api/admin/problems/${problem.value.id}/assets?path=${encodeURIComponent(path)}`,
      { method: 'DELETE' }
    )
    await loadAssets()
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : String(caught)
  }
}

async function openAssetEditor(asset: ProblemAsset) {
  if (!problem.value || !asset.text) {
    error.value = t('admin.problems.binaryEditHint')
    return
  }
  error.value = ''
  selectedAsset.value = asset
  assetContent.value = ''
  showAssetEditor.value = true
  try {
    const result = await apiFetch<{ content: string; encoding: 'utf8' | 'base64'; text: boolean }>(
      `/api/admin/problems/${problem.value.id}/assets/content?path=${encodeURIComponent(asset.path)}`
    )
    if (!result.text || result.encoding !== 'utf8') {
      showAssetEditor.value = false
      error.value = t('admin.problems.binaryEditHint')
      return
    }
    assetContent.value = result.content
  } catch (caught) {
    showAssetEditor.value = false
    error.value = caught instanceof Error ? caught.message : String(caught)
  }
}

async function saveSelectedAsset() {
  if (!selectedAsset.value) return
  assetSaving.value = true
  error.value = ''
  try {
    await saveAssetText(selectedAsset.value.path, assetContent.value)
    await loadAssets()
    showAssetEditor.value = false
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : String(caught)
  } finally {
    assetSaving.value = false
  }
}

async function saveAssetText(path: string, content: string) {
  if (!problem.value) return
  await apiFetch(`/api/admin/problems/${problem.value.id}/assets/content`, {
    method: 'PUT',
    body: JSON.stringify({
      path,
      content,
      encoding: 'utf8'
    })
  })
}
</script>

<template>
  <main class="page">
    <n-spin :show="loading">
      <section v-if="problem" class="problem-detail-page">
        <n-alert
          v-if="assignmentId || contestId"
          type="info"
          :show-icon="false"
          class="problem-context"
        >
          <span v-if="assignmentId">{{ t('problemDetail.assignmentContext') }} {{ assignmentId }}</span>
          <span v-if="contestId">{{ t('problemDetail.contestContext') }} {{ contestId }}</span>
        </n-alert>

        <section class="problem-top-grid">
          <n-card :bordered="false" class="statement-card">
            <template #header>
              <span class="statement-title">
                <n-tooltip v-if="canManage">
                  <template #trigger>
                    <n-button
                      quaternary
                      circle
                      size="small"
                      :type="problem.visible ? 'success' : 'default'"
                      :aria-label="visibilityLabel(problem.visible)"
                      @click="toggleProblemVisible"
                    >
                      <template #icon>
                        <n-icon :component="problem.visible ? EyeOutline : EyeOffOutline" />
                      </template>
                    </n-button>
                  </template>
                  {{ visibilityLabel(problem.visible) }}
                </n-tooltip>
                <span>P{{ problem.id }}</span>
                <template v-if="!editMode">
                  <n-tag size="small" :bordered="false" type="info">{{ limitText }}</n-tag>
                  <n-tag
                    v-for="item in problem.tags"
                    :key="item"
                    size="small"
                    :bordered="false"
                  >
                    {{ item }}
                  </n-tag>
                </template>
              </span>
            </template>
            <template #header-extra>
              <n-space :size="8" align="center" class="statement-header-extra">
                <n-button
                  v-if="!editMode"
                  size="small"
                  :aria-label="t('admin.edit')"
                  @click="editMode = true"
                >
                  <template #icon>
                    <n-icon :component="CreateOutline" />
                  </template>
                  {{ t('admin.edit') }}
                </n-button>
                <template v-else>
                  <n-button size="small" @click="cancelEdit">
                    <template #icon>
                      <n-icon :component="CloseOutline" />
                    </template>
                    {{ t('admin.cancel') }}
                  </n-button>
                  <n-button size="small" type="primary" :loading="saving" @click="saveProblem">
                    <template #icon>
                      <n-icon :component="SaveOutline" />
                    </template>
                    {{ t('admin.save') }}
                  </n-button>
                </template>
              </n-space>
            </template>
            <template v-if="editMode">
              <div class="statement-edit-panel">
                <n-form :model="editForm" label-placement="top" size="small">
                  <div class="statement-edit-grid">
                    <n-form-item :label="t('common.title')" class="statement-title-field">
                      <n-input v-model:value="editForm.title" />
                    </n-form-item>
                    <n-form-item :label="t('admin.problems.timeMs')">
                      <n-input-number v-model:value="editForm.timeLimit" :min="100" class="full-width" />
                    </n-form-item>
                    <n-form-item :label="t('admin.problems.memoryMb')">
                      <n-input-number v-model:value="editForm.memoryLimitMb" :min="16" class="full-width" />
                    </n-form-item>
                  </div>
                  <n-form-item :label="t('common.tags')">
                    <n-dynamic-tags v-model:value="editForm.tags" />
                  </n-form-item>
                </n-form>
              </div>
              <markdown-editor
                v-model="editForm.statement"
                :problem-id="problem.id"
                upload-enabled
              />
            </template>
            <template v-else>
              <markdown-view :source="statement" />
            </template>
          </n-card>

          <n-card :bordered="false" class="meta-card">
            <div class="side-status">
              <section class="metric-grid">
                <RouterLink
                  v-if="problem.recentSubmission"
                  class="metric-card"
                  :to="`/submissions/${problem.recentSubmission.id}`"
                >
                  <span>{{ t('problemDetail.myRecord') }}</span>
                  <strong>{{ recentSubmissionText }}</strong>
                </RouterLink>
                <div v-else class="metric-card is-muted">
                  <span>{{ t('problemDetail.myRecord') }}</span>
                  <strong>{{ recentSubmissionText }}</strong>
                </div>
                <RouterLink class="metric-card" :to="`/submissions?problemId=${problem.id}`">
                  <span>{{ t('problems.passRate') }}</span>
                  <strong>{{ passRateText }}</strong>
                </RouterLink>
                <RouterLink class="metric-card" :to="`/discussion?tags=P${problem.id}`">
                  <span>{{ t('problemDetail.discussionCount') }}</span>
                  <strong>{{ problem.discussionCount }}</strong>
                </RouterLink>
              </section>
              <section v-if="canManage" class="meta-section asset-meta-section">
                <div class="meta-row">
                  <span>{{ t('admin.problems.mode') }}</span>
                  <n-select
                    :value="problem.mode"
                    size="small"
                    :options="modeOptions"
                    :render-label="renderModeLabel"
                    :consistent-menu-width="false"
                    :loading="saving"
                    class="mode-select"
                    @update:value="updateProblemMode"
                  />
                </div>
                <div class="meta-row">
                  <span>{{ t('admin.problems.dataCases') }}</span>
                  <strong>{{ dataCaseCount }}</strong>
                </div>
                <div class="meta-row">
                  <span>{{ t('admin.problems.dataSize') }}</span>
                  <strong>{{ dataAssetSizeText }}</strong>
                </div>
                <div v-if="problem.deletedAt" class="meta-row">
                  <span>{{ t('admin.problems.deleted') }}</span>
                  <n-tag size="small" :bordered="false" type="error">
                    {{ formatDate(problem.deletedAt) }}
                  </n-tag>
                </div>
                <div class="meta-actions">
                  <n-button secondary size="small" @click="openManageModal">
                    <template #icon>
                      <n-icon :component="FolderOpenOutline" />
                    </template>
                    {{ t('admin.problems.assets') }}
                  </n-button>
                  <n-popconfirm v-if="!problem.deletedAt" @positive-click="toggleProblemDeleted">
                    <template #trigger>
                      <n-button tertiary size="small" type="error" :loading="saving">
                        <template #icon>
                          <n-icon :component="TrashOutline" />
                        </template>
                        {{ t('admin.delete') }}
                      </n-button>
                    </template>
                    {{ t('problems.deleteConfirm') }}
                  </n-popconfirm>
                </div>
              </section>
            </div>
          </n-card>
        </section>

        <n-modal
          v-if="canManage"
          v-model:show="showManageModal"
          preset="card"
          :title="t('admin.problems.assets')"
          class="problem-manage-modal"
          :z-index="30000"
          style="width: min(760px, calc(100vw - 32px))"
        >
          <div class="asset-panel">
            <div class="asset-sections">
              <section v-for="section in assetSections" :key="section.key" class="asset-section">
                <div class="asset-section-head">
                  <div class="asset-section-title">
                    <strong>{{ section.title }}</strong>
                  </div>
                  <n-space :size="8" justify="end">
                    <n-button
                      v-if="section.key === 'data'"
                      size="small"
                      secondary
                      @click="createDataCase"
                    >
                      <template #icon>
                        <n-icon :component="AddOutline" />
                      </template>
                      {{ t('admin.problems.addDataCase') }}
                    </n-button>
                    <n-button
                      v-if="section.key === 'data'"
                      size="small"
                      secondary
                      @click="downloadDataAssets"
                    >
                      <template #icon>
                        <n-icon :component="DownloadOutline" />
                      </template>
                      {{ t('admin.problems.downloadDataZip') }}
                    </n-button>
                    <n-button
                      v-if="section.key === 'root' && !assetsBySection(section.key).length"
                      size="small"
                      secondary
                      @click="ensureCustomJudgeTemplates"
                    >
                      <template #icon>
                        <n-icon :component="CreateOutline" />
                      </template>
                      {{ t('admin.problems.fillTemplate') }}
                    </n-button>
                    <n-upload
                      :custom-request="uploadAssetForSection(section.key)"
                      :show-file-list="false"
                      accept="*"
                    >
                      <n-button size="small" secondary>
                        <template #icon>
                          <n-icon :component="FolderOpenOutline" />
                        </template>
                        {{ t('admin.upload') }}
                      </n-button>
                    </n-upload>
                  </n-space>
                </div>
                <div
                  class="asset-dropzone"
                  :class="{ 'is-dragging': dragSection === section.key }"
                  @dragenter.prevent="dragSection = section.key"
                  @dragover.prevent="dragSection = section.key"
                  @dragleave.prevent="dragSection = null"
                  @drop.prevent="uploadDroppedFiles(section.key, $event)"
                >
                  <div v-if="assetsBySection(section.key).length" class="asset-list">
                    <div
                      v-for="asset in assetsBySection(section.key)"
                      :key="asset.path"
                      class="asset-row"
                    >
                      <span class="asset-path">{{ asset.path }}</span>
                      <span class="muted">{{ formatBytes(asset.size) }}</span>
                      <div class="asset-row-actions">
                        <n-tooltip v-if="asset.text">
                          <template #trigger>
                            <n-button size="tiny" tertiary @click="openAssetEditor(asset)">
                              <template #icon>
                                <n-icon :component="CreateOutline" />
                              </template>
                            </n-button>
                          </template>
                          {{ t('admin.edit') }}
                        </n-tooltip>
                        <n-popconfirm @positive-click="deleteAsset(asset.path)">
                          <template #trigger>
                            <n-button size="tiny" tertiary type="error">
                              <template #icon>
                                <n-icon :component="TrashOutline" />
                              </template>
                            </n-button>
                          </template>
                          {{ t('admin.problems.assetDeleteConfirm') }}
                        </n-popconfirm>
                      </div>
                    </div>
                  </div>
                  <n-empty v-else size="small" />
                </div>
              </section>
            </div>

          </div>
        </n-modal>

        <n-modal
          v-if="canManage"
          v-model:show="showAssetEditor"
          preset="card"
          :title="selectedAsset?.path || t('admin.problems.packageContent')"
          class="asset-editor-modal"
          :z-index="31000"
          style="width: min(920px, calc(100vw - 32px))"
        >
          <code-editor
            v-model="assetContent"
            :language-id="assetEditorLanguage"
          />
          <n-space justify="end" class="form-actions">
            <n-button @click="showAssetEditor = false">{{ t('admin.cancel') }}</n-button>
            <n-button type="primary" :loading="assetSaving" @click="saveSelectedAsset">
              <template #icon>
                <n-icon :component="SaveOutline" />
              </template>
              {{ t('admin.save') }}
            </n-button>
          </n-space>
        </n-modal>

        <n-card :title="t('problemDetail.submit')" :bordered="false" class="submit-card">
          <template v-if="auth.signedIn" #header-extra>
            <n-checkbox v-model:checked="sourceOpen">
              {{ t('problemDetail.sourceOpen') }}
            </n-checkbox>
          </template>
          <div class="submit-content">
            <template v-if="auth.signedIn">
              <code-editor v-model="codeText" :language-id="languageId" />
              <div class="submit-actions">
                <n-select
                  v-model:value="languageId"
                  :options="languageOptions"
                  class="language-select"
                  @update:value="updateTemplate"
                />
                <n-button
                  type="primary"
                  :loading="submitting"
                  @click="submit"
                >
                  {{ t('problemDetail.submit') }}
                </n-button>
              </div>
            </template>
            <n-alert v-else type="info" :show-icon="false">
              {{ t('problemDetail.signIn') }}
            </n-alert>
            <p v-if="error" class="form-error">{{ error }}</p>
          </div>
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

.statement-card,
.meta-card {
  align-self: start;
}

.statement-card :deep(.n-card-header),
.meta-card :deep(.n-card-header) {
  align-items: center;
  padding-bottom: 10px;
}

.statement-title {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
}

.statement-header-extra {
  max-width: 100%;
}

.statement-edit-panel {
  margin-bottom: 12px;
  padding: 12px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: color-mix(in srgb, var(--surface-bg) 96%, var(--text-color) 4%);
}

.statement-edit-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(160px, 1fr));
  gap: 12px;
  align-items: start;
}

.statement-title-field {
  grid-column: 1 / -1;
}

.problem-context {
  width: fit-content;
}

.side-status {
  display: grid;
  gap: 16px;
}

.metric-grid,
.meta-section {
  display: grid;
  gap: 10px;
}

.asset-meta-section {
  padding-top: 16px;
  border-top: 1px solid var(--border-color);
}

.metric-card {
  display: grid;
  gap: 4px;
  padding: 10px 12px;
  border-radius: var(--radius-md);
  color: inherit;
  text-decoration: none;
  background: color-mix(in srgb, var(--surface-bg) 94%, var(--text-color) 6%);
}

.metric-card:hover {
  color: inherit;
  background: color-mix(in srgb, var(--surface-bg) 90%, var(--brand) 10%);
}

.metric-card.is-muted:hover {
  background: color-mix(in srgb, var(--surface-bg) 94%, var(--text-color) 6%);
}

.metric-card > span,
.meta-row > span {
  color: var(--muted-color);
  font-size: 13px;
}

.metric-card > strong {
  font-size: 17px;
  line-height: 1.2;
}

.meta-row {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  min-width: 0;
}

.mode-select {
  width: 160px;
}

.meta-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding-top: 4px;
}

.asset-panel {
  display: grid;
  gap: 16px;
}

.asset-section-head {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 12px;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 12px;
}

.asset-section-title {
  min-width: 0;
  font-size: 16px;
  line-height: 1.4;
  white-space: nowrap;
}

.asset-dropzone {
  min-height: 96px;
  padding: 6px;
  border: 1px dashed transparent;
  border-radius: var(--radius-md);
  transition:
    border-color 0.15s ease,
    background-color 0.15s ease;
}

.asset-dropzone.is-dragging {
  border-color: var(--brand);
  background: var(--brand-soft);
}

.asset-list {
  display: grid;
  gap: 8px;
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

.asset-row-actions {
  display: inline-flex;
  gap: 4px;
  justify-content: flex-end;
}

.asset-path {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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

.submit-content {
  display: grid;
  gap: 12px;
}

.submit-actions {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  margin-top: 4px;
}

.language-select {
  flex: 0 1 260px;
  width: 260px;
  min-width: 0;
}

@media (max-width: 860px) {
  .problem-top-grid {
    grid-template-columns: 1fr;
  }

  .statement-edit-grid {
    display: grid;
    grid-template-columns: 1fr;
  }

  .asset-section-head {
    grid-template-columns: minmax(0, 1fr) auto;
    align-items: center;
  }

  .asset-row {
    grid-template-columns: minmax(0, 1fr);
  }

  .submit-actions {
    display: grid;
    justify-content: stretch;
    justify-items: stretch;
  }

  .language-select {
    flex-basis: auto;
    width: 100%;
  }
}
</style>
