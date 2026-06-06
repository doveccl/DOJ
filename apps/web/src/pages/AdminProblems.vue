<script setup lang="ts">
import {
  NAlert,
  NButton,
  NCard,
  NCheckbox,
  NDataTable,
  NDynamicInput,
  NDynamicTags,
  NForm,
  NFormItem,
  NGrid,
  NGridItem,
  NInput,
  NInputNumber,
  NModal,
  NSpace,
  NTag,
  NText,
  NUpload,
  NUploadDragger
} from 'naive-ui'
import type { DataTableColumns, UploadFileInfo } from 'naive-ui'
import { computed, h, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../api'
import CodeEditor from '../components/CodeEditor.vue'
import MarkdownEditor from '../components/MarkdownEditor.vue'
import { useAuthStore } from '../stores/auth'

interface ProblemSummary {
  timeLimitMs: number
  memoryLimitBytes: number
  caseCount: number
  inlineCaseCount: number
}

interface ProblemRow {
  id: number
  title: string
  tags: string[]
  solvedCount: number
  visible: boolean
  timeLimitMs: number
  memoryLimitBytes: number
  caseCount: number
  summary: ProblemSummary
}

interface TestCase {
  name?: string
  input: string
  output: string
  hidden: boolean
  points?: number | null
}

interface PackageFile {
  path: string
  filename: string
  contentType: string
  sizeBytes: number
  updatedAt: string
}

interface ProblemDetail {
  problem: ProblemRow & { statementMarkdown: string; testCases: TestCase[] }
  summary: ProblemSummary
  package: PackageFile[]
  testCases: TestCase[]
}

const auth = useAuthStore()
const { t } = useI18n()
const canManage = computed(() => auth.user?.groups.includes('admin') ?? false)
const loading = ref(true)
const saving = ref(false)
const editing = ref(false)
const error = ref('')
const message = ref('')
const showCreateModal = ref(false)
const showEditModal = ref(false)
const showPackageModal = ref(false)
const problems = ref<ProblemRow[]>([])
const pagination = reactive({
  page: 1,
  pageSize: 50,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [20, 50, 100]
})

const form = reactive({
  title: '',
  tags: [] as string[],
  statementMarkdown: '# New Problem\n\nDescribe the task.',
  timeLimitMs: 1000,
  memoryLimitMb: 256
})

const editForm = reactive({
  problemId: null as number | null,
  title: '',
  tags: [] as string[],
  visible: true,
  statementMarkdown: '',
  timeLimitMs: 1000,
  memoryLimitMb: 256,
  caseCount: 0,
  testCases: [] as TestCase[]
})

// Package editor state.
const packageProblemId = ref<number | null>(null)
const packageProblemTitle = ref('')
const packageFiles = ref<PackageFile[]>([])
const packageLoading = ref(false)
const packageSaving = ref(false)
const selectedPath = ref<string | null>(null)
const fileContent = ref('')
const newFilePath = ref('')
const uploadPrefix = ref('data/')
const uploadFiles = ref<File[]>([])

const hasDockerfile = computed(() => packageFiles.value.some((file) => file.path === 'Dockerfile'))

const columns = computed<DataTableColumns<ProblemRow>>(() => [
  { title: t('common.id'), key: 'id', width: 90 },
  { title: t('common.title'), key: 'title', minWidth: 220 },
  {
    title: t('admin.problems.visible'),
    key: 'visible',
    width: 90,
    render(row) {
      return h(
        NTag,
        { bordered: false, type: row.visible ? 'success' : 'default', size: 'small' },
        () => (row.visible ? t('admin.problems.yes') : t('admin.problems.no'))
      )
    }
  },
  {
    title: t('admin.problems.limits'),
    key: 'limits',
    width: 130,
    render(row) {
      return h('div', { class: 'compact-stack' }, [
        h('span', `${row.timeLimitMs} ms`),
        h('span', formatBytes(row.memoryLimitBytes))
      ])
    }
  },
  {
    title: t('admin.problems.testdata'),
    key: 'package',
    width: 150,
    render(row) {
      return h(NSpace, { size: 6 }, () => [
        h(NTag, { bordered: false, type: 'info', size: 'small' }, () =>
          t('admin.problems.caseCount', {
            count: row.summary.caseCount || row.summary.inlineCaseCount
          })
        )
      ])
    }
  },
  {
    title: t('common.tags'),
    key: 'tags',
    minWidth: 140,
    render(row) {
      if (!row.tags.length) return '-'
      return h(NSpace, { size: 6, wrapItem: false }, () =>
        row.tags.map((tag) =>
          h(NTag, { key: tag, size: 'small', bordered: false, type: 'info' }, () => tag)
        )
      )
    }
  },
  { title: t('common.solved'), key: 'solvedCount', width: 90 },
  {
    title: t('admin.actions'),
    key: 'action',
    width: 200,
    render(row) {
      return h(NSpace, { size: 8 }, () => [
        h(
          NButton,
          { size: 'small', secondary: true, onClick: () => loadProblemForEdit(row.id) },
          () => t('admin.edit')
        ),
        h(NButton, { size: 'small', secondary: true, onClick: () => openPackageModal(row) }, () =>
          t('admin.problems.package')
        )
      ])
    }
  }
])

async function loadProblems() {
  loading.value = true
  error.value = ''
  try {
    const data = await apiFetch<{ list: ProblemRow[]; total: number }>(
      `/api/admin/problems?page=${pagination.page}&pageSize=${pagination.pageSize}`
    )
    problems.value = data.list
    pagination.itemCount = data.total
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  void loadProblems()
}

function handlePageSizeChange(pageSize: number) {
  pagination.pageSize = pageSize
  pagination.page = 1
  void loadProblems()
}

async function createProblem() {
  saving.value = true
  error.value = ''
  try {
    await apiFetch('/api/problems', {
      method: 'POST',
      body: JSON.stringify({
        title: form.title,
        tags: form.tags,
        statementMarkdown: form.statementMarkdown,
        timeLimitMs: form.timeLimitMs,
        memoryLimitBytes: form.memoryLimitMb * 1024 * 1024
      })
    })
    form.title = ''
    form.tags = []
    showCreateModal.value = false
    await loadProblems()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

async function loadProblemForEdit(problemId: number) {
  editing.value = true
  error.value = ''
  message.value = ''
  try {
    const data = await apiFetch<ProblemDetail>(`/api/admin/problems/${problemId}`)
    editForm.problemId = data.problem.id
    editForm.title = data.problem.title
    editForm.tags = [...data.problem.tags]
    editForm.visible = data.problem.visible
    editForm.statementMarkdown = data.problem.statementMarkdown
    editForm.timeLimitMs = data.problem.timeLimitMs
    editForm.memoryLimitMb = Math.round(data.problem.memoryLimitBytes / 1024 / 1024)
    editForm.caseCount = data.problem.caseCount
    editForm.testCases = (data.testCases ?? []).map((testCase) => ({
      name: testCase.name ?? '',
      input: testCase.input,
      output: testCase.output,
      hidden: testCase.hidden,
      points: testCase.points ?? null
    }))
    showEditModal.value = true
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    editing.value = false
  }
}

async function updateProblem() {
  if (!editForm.problemId) return
  editing.value = true
  error.value = ''
  message.value = ''
  try {
    const result = await apiFetch<ProblemDetail>(`/api/problems/${editForm.problemId}`, {
      method: 'PATCH',
      body: JSON.stringify({
        title: editForm.title,
        tags: editForm.tags,
        visible: editForm.visible,
        statementMarkdown: editForm.statementMarkdown,
        timeLimitMs: editForm.timeLimitMs,
        memoryLimitBytes: editForm.memoryLimitMb * 1024 * 1024,
        caseCount: editForm.caseCount,
        testCases: editForm.testCases.map((testCase) => ({
          name: testCase.name || undefined,
          input: testCase.input,
          output: testCase.output,
          hidden: testCase.hidden,
          points: typeof testCase.points === 'number' ? testCase.points : undefined
        }))
      })
    })
    message.value = t('admin.problems.saved', { id: result.problem.id })
    showEditModal.value = false
    await loadProblems()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    editing.value = false
  }
}

function createTestCase(): TestCase {
  return { name: '', input: '', output: '', hidden: true, points: null }
}

// Package editor.
async function openPackageModal(row: ProblemRow) {
  packageProblemId.value = row.id
  packageProblemTitle.value = row.title
  selectedPath.value = null
  fileContent.value = ''
  newFilePath.value = ''
  uploadFiles.value = []
  message.value = ''
  error.value = ''
  showPackageModal.value = true
  await loadPackageFiles()
}

async function loadPackageFiles() {
  if (!packageProblemId.value) return
  packageLoading.value = true
  try {
    const data = await apiFetch<{ files: PackageFile[] }>(
      `/api/problems/${packageProblemId.value}/package`
    )
    packageFiles.value = data.files
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    packageLoading.value = false
  }
}

async function selectFile(path: string) {
  if (!packageProblemId.value) return
  error.value = ''
  try {
    const data = await apiFetch<{ path: string; content: string }>(
      `/api/problems/${packageProblemId.value}/package/content?path=${encodeURIComponent(path)}`
    )
    selectedPath.value = data.path
    fileContent.value = data.content
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  }
}

function startNewFile() {
  const path = newFilePath.value.trim()
  if (!path) return
  selectedPath.value = path
  fileContent.value = ''
  newFilePath.value = ''
}

async function saveFile() {
  if (!packageProblemId.value || !selectedPath.value) return
  packageSaving.value = true
  error.value = ''
  message.value = ''
  try {
    await apiFetch(`/api/problems/${packageProblemId.value}/package`, {
      method: 'PUT',
      body: JSON.stringify({ path: selectedPath.value, content: fileContent.value })
    })
    message.value = t('admin.problems.packageSaved', { path: selectedPath.value })
    await loadPackageFiles()
    await loadProblems()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    packageSaving.value = false
  }
}

async function deleteFile(path: string) {
  if (!packageProblemId.value) return
  packageSaving.value = true
  error.value = ''
  message.value = ''
  try {
    await apiFetch(
      `/api/problems/${packageProblemId.value}/package?path=${encodeURIComponent(path)}`,
      { method: 'DELETE' }
    )
    message.value = t('admin.problems.packageDeleted', { path })
    if (selectedPath.value === path) {
      selectedPath.value = null
      fileContent.value = ''
    }
    await loadPackageFiles()
    await loadProblems()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    packageSaving.value = false
  }
}

async function uploadPackageFiles() {
  if (!packageProblemId.value || !uploadFiles.value.length) return
  packageSaving.value = true
  error.value = ''
  message.value = ''
  try {
    const formData = new FormData()
    if (uploadPrefix.value.trim()) formData.append('prefix', uploadPrefix.value.trim())
    for (const file of uploadFiles.value) formData.append('file', file)
    await apiFetch(`/api/problems/${packageProblemId.value}/package/upload`, {
      method: 'POST',
      body: formData
    })
    uploadFiles.value = []
    await loadPackageFiles()
    await loadProblems()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    packageSaving.value = false
  }
}

function handleUploadChange(options: { fileList: UploadFileInfo[] }) {
  uploadFiles.value = options.fileList
    .map((item) => item.file)
    .filter((file): file is File => !!file)
}

function editorLanguage(path: string | null) {
  if (!path) return 'plaintext'
  if (path === 'Dockerfile' || path.endsWith('.dockerfile')) return 'dockerfile'
  if (path.endsWith('.cc') || path.endsWith('.cpp') || path.endsWith('.h')) return 'cpp'
  if (path.endsWith('.py')) return 'python'
  if (path.endsWith('.json')) return 'json'
  if (path.endsWith('.sh')) return 'shell'
  return 'plaintext'
}

function formatBytes(bytes: number | null | undefined) {
  if (!bytes || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  const value = bytes / 1024 ** index
  return `${value >= 10 || index === 0 ? value.toFixed(0) : value.toFixed(1)} ${units[index]}`
}

watch(canManage, (allowed) => {
  if (allowed) loadProblems()
})

onMounted(() => {
  if (canManage.value) {
    loadProblems()
  } else {
    loading.value = false
  }
})
</script>

<template>
  <main class="page">
    <n-alert v-if="!canManage" type="warning" class="page-alert">
      {{ t('admin.requireAdmin') }}
    </n-alert>
    <n-alert v-if="error" type="error" class="page-alert">{{ error }}</n-alert>
    <n-alert v-if="message" type="success" class="page-alert">{{ message }}</n-alert>

    <n-card v-if="canManage" :bordered="false">
      <n-space justify="end" class="table-toolbar">
        <n-button type="primary" @click="showCreateModal = true">
          {{ t('admin.problems.create') }}
        </n-button>
      </n-space>
      <n-data-table
        remote
        :columns="columns"
        :data="problems"
        :bordered="false"
        :loading="loading"
        :pagination="pagination"
        class="admin-table"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>

    <n-modal
      v-model:show="showCreateModal"
      preset="card"
      :title="t('admin.problems.create')"
      class="form-modal"
    >
      <n-form :model="form" label-placement="top">
        <n-form-item :label="t('common.title')">
          <n-input v-model:value="form.title" placeholder="A+B Problem" />
        </n-form-item>
        <n-form-item :label="t('common.tags')">
          <n-dynamic-tags v-model:value="form.tags" />
        </n-form-item>
        <n-form-item :label="t('admin.problems.statement')">
          <markdown-editor v-model="form.statementMarkdown" />
        </n-form-item>
        <div class="form-grid two">
          <n-form-item :label="t('admin.problems.timeMs')">
            <n-input-number v-model:value="form.timeLimitMs" :min="100" class="full-width" />
          </n-form-item>
          <n-form-item :label="t('admin.problems.memoryMb')">
            <n-input-number v-model:value="form.memoryLimitMb" :min="16" class="full-width" />
          </n-form-item>
        </div>
        <n-space justify="end" class="form-actions">
          <n-button @click="showCreateModal = false">{{ t('admin.cancel') }}</n-button>
          <n-button type="primary" :loading="saving" :disabled="!form.title" @click="createProblem">
            {{ t('admin.create') }}
          </n-button>
        </n-space>
      </n-form>
    </n-modal>

    <n-modal
      v-model:show="showEditModal"
      preset="card"
      :title="t('admin.problems.edit')"
      class="form-modal"
    >
      <n-form :model="editForm" label-placement="top">
        <n-form-item :label="t('common.title')">
          <n-input v-model:value="editForm.title" placeholder="A+B Problem" />
        </n-form-item>
        <n-form-item :label="t('admin.problems.visible')">
          <n-checkbox v-model:checked="editForm.visible">
            {{ t('admin.problems.publicVisible') }}
          </n-checkbox>
        </n-form-item>
        <n-form-item :label="t('common.tags')">
          <n-dynamic-tags v-model:value="editForm.tags" />
        </n-form-item>
        <n-form-item :label="t('admin.problems.statement')">
          <markdown-editor v-model="editForm.statementMarkdown" />
        </n-form-item>
        <div class="form-grid two">
          <n-form-item :label="t('admin.problems.timeMs')">
            <n-input-number v-model:value="editForm.timeLimitMs" :min="100" class="full-width" />
          </n-form-item>
          <n-form-item :label="t('admin.problems.memoryMb')">
            <n-input-number v-model:value="editForm.memoryLimitMb" :min="16" class="full-width" />
          </n-form-item>
        </div>
        <n-form-item :label="t('admin.problems.caseCountField')">
          <n-input-number v-model:value="editForm.caseCount" :min="0" class="full-width" />
        </n-form-item>
        <n-text depth="3" class="field-hint">{{ t('admin.problems.caseCountHint') }}</n-text>
        <n-form-item :label="t('admin.problems.sampleCases')">
          <n-dynamic-input
            v-model:value="editForm.testCases"
            :on-create="createTestCase"
            #="{ value }"
          >
            <div class="case-row">
              <n-input v-model:value="value.name" :placeholder="t('admin.problems.packagePath')" />
              <n-input
                v-model:value="value.input"
                type="textarea"
                :autosize="{ minRows: 1, maxRows: 4 }"
                placeholder="input"
              />
              <n-input
                v-model:value="value.output"
                type="textarea"
                :autosize="{ minRows: 1, maxRows: 4 }"
                placeholder="output"
              />
              <n-input-number
                v-model:value="value.points"
                :min="0"
                :max="100"
                placeholder="pts"
                class="case-points"
              />
              <n-checkbox v-model:checked="value.hidden">{{ t('admin.problems.no') }}</n-checkbox>
            </div>
          </n-dynamic-input>
        </n-form-item>
        <n-space justify="end" class="form-actions">
          <n-button @click="showEditModal = false">{{ t('admin.cancel') }}</n-button>
          <n-button
            type="primary"
            :loading="editing"
            :disabled="!editForm.problemId || !editForm.title"
            @click="updateProblem"
          >
            {{ t('admin.save') }}
          </n-button>
        </n-space>
      </n-form>
    </n-modal>

    <n-modal
      v-model:show="showPackageModal"
      preset="card"
      :title="`${t('admin.problems.packageTitle')} · P${packageProblemId} ${packageProblemTitle}`"
      class="form-modal wide"
    >
      <n-alert :type="hasDockerfile ? 'info' : 'default'" class="page-alert">
        <strong>{{
          hasDockerfile ? t('admin.problems.custom') : t('admin.problems.default')
        }}</strong>
        — {{ t('admin.problems.packageHint') }}
      </n-alert>

      <n-grid :cols="3" :x-gap="16" class="package-grid">
        <n-grid-item :span="1">
          <div class="package-files">
            <p class="muted">{{ t('admin.problems.packageFiles') }}</p>
            <p v-if="!packageFiles.length && !packageLoading" class="muted">
              {{ t('admin.problems.packageEmpty') }}
            </p>
            <button
              v-for="file in packageFiles"
              :key="file.path"
              type="button"
              class="package-file"
              :class="{ active: file.path === selectedPath }"
              @click="selectFile(file.path)"
            >
              <span class="package-file-path">{{ file.path }}</span>
              <span class="muted package-file-size">{{ formatBytes(file.sizeBytes) }}</span>
              <n-button
                size="tiny"
                quaternary
                type="error"
                :disabled="packageSaving"
                @click.stop="deleteFile(file.path)"
              >
                {{ t('admin.problems.packageDelete') }}
              </n-button>
            </button>

            <div class="package-new">
              <n-input
                v-model:value="newFilePath"
                size="small"
                :placeholder="t('admin.problems.packageNewFilePath')"
                @keyup.enter="startNewFile"
              />
              <n-button
                size="small"
                secondary
                :disabled="!newFilePath.trim()"
                @click="startNewFile"
              >
                {{ t('admin.problems.packageNewFile') }}
              </n-button>
            </div>

            <div class="package-upload">
              <n-input
                v-model:value="uploadPrefix"
                size="small"
                :placeholder="t('admin.problems.packageUploadPrefix')"
              />
              <n-upload multiple :default-upload="false" @change="handleUploadChange">
                <n-upload-dragger>
                  <div class="upload-dragger-content">
                    <strong>{{ t('admin.problems.packageUpload') }}</strong>
                    <span class="muted">{{ t('admin.problems.packageUploadHint') }}</span>
                  </div>
                </n-upload-dragger>
              </n-upload>
              <n-button
                size="small"
                type="primary"
                :loading="packageSaving"
                :disabled="!uploadFiles.length"
                @click="uploadPackageFiles"
              >
                {{ t('admin.upload') }}
              </n-button>
            </div>
          </div>
        </n-grid-item>

        <n-grid-item :span="2">
          <div v-if="selectedPath" class="package-editor">
            <p class="package-editor-path">{{ selectedPath }}</p>
            <code-editor
              v-model="fileContent"
              :language-id="editorLanguage(selectedPath)"
              class="package-code"
            />
            <n-space justify="end" class="form-actions">
              <n-button type="primary" :loading="packageSaving" @click="saveFile">
                {{ t('admin.problems.packageSave') }}
              </n-button>
            </n-space>
          </div>
          <p v-else class="muted package-empty">{{ t('admin.problems.packageSelectHint') }}</p>
        </n-grid-item>
      </n-grid>
    </n-modal>
  </main>
</template>

<style scoped>
.field-hint {
  display: block;
  margin: -8px 0 12px;
  font-size: 12px;
}

.case-row {
  display: grid;
  grid-template-columns: 120px 1fr 1fr 72px auto;
  gap: 8px;
  align-items: start;
  width: 100%;
}

.case-points {
  width: 72px;
}

.package-grid {
  margin-top: 12px;
}

.package-files {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.package-file {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border: 1px solid var(--doj-border, #e0e0e6);
  border-radius: 6px;
  background: transparent;
  cursor: pointer;
  text-align: left;
}

.package-file.active {
  border-color: var(--doj-primary, #18a058);
}

.package-file-path {
  flex: 1;
  font-family: var(--doj-mono, monospace);
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.package-file-size {
  font-size: 12px;
}

.package-new,
.package-upload {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 8px;
}

.package-editor-path {
  font-family: var(--doj-mono, monospace);
  font-size: 13px;
  margin: 0 0 8px;
}

.package-code {
  height: 420px;
}

.package-empty {
  padding: 40px 0;
  text-align: center;
}
</style>
