<script setup lang="ts">
import {
  NAlert,
  NButton,
  NCard,
  NCheckbox,
  NDataTable,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSpace
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { computed, h, onMounted, reactive, ref, watch } from 'vue'
import { apiFetch } from '../api'
import { useAuthStore } from '../stores/auth'

interface ProblemRow {
  id: number
  title: string
  tags: string[]
  solvedCount: number
  visible: boolean
}

interface ProblemDetail {
  problem: ProblemRow & { slug: string | null }
  version: {
    id: number
    version: number
    statementMarkdown: string
    timeLimitMs: number
    memoryLimitBytes: number
    outputLimitBytes: number
    testCases: unknown[]
  }
}

const auth = useAuthStore()
const canManage = computed(() => auth.user?.groups.includes('admin') ?? false)
const loading = ref(true)
const saving = ref(false)
const editing = ref(false)
const uploading = ref(false)
const error = ref('')
const uploadMessage = ref('')
const editMessage = ref('')
const problems = ref<ProblemRow[]>([])
const selectedFile = ref<File | null>(null)
const form = reactive({
  title: '',
  slug: '',
  tags: '',
  statementMarkdown: '# New Problem\n\nDescribe the task.',
  timeLimitMs: 1000,
  memoryLimitMb: 256,
  outputLimitMb: 64,
  testCasesText: '[\n  {\n    "name": "sample",\n    "input": "",\n    "output": ""\n  }\n]'
})
const uploadForm = reactive({
  problemId: null as number | null
})
const editForm = reactive({
  problemId: null as number | null,
  title: '',
  slug: '',
  tags: '',
  visible: true,
  statementMarkdown: '',
  timeLimitMs: 1000,
  memoryLimitMb: 256,
  outputLimitMb: 64,
  testCasesText: '[]'
})

const columns: DataTableColumns<ProblemRow> = [
  { title: 'ID', key: 'id', width: 96 },
  { title: 'Title', key: 'title', minWidth: 240 },
  {
    title: 'Visible',
    key: 'visible',
    width: 110,
    render(row) {
      return row.visible ? 'Yes' : 'No'
    }
  },
  {
    title: 'Tags',
    key: 'tags',
    minWidth: 160,
    render(row) {
      return row.tags.join(', ') || '-'
    }
  },
  { title: 'Solved', key: 'solvedCount', width: 110 },
  {
    title: 'Action',
    key: 'action',
    width: 100,
    render(row) {
      return h(
        NButton,
        {
          size: 'small',
          secondary: true,
          onClick: () => loadProblemForEdit(row.id)
        },
        () => 'Edit'
      )
    }
  }
]

async function loadProblems() {
  loading.value = true
  error.value = ''
  try {
    const data = await apiFetch<{ list: ProblemRow[] }>('/api/admin/problems')
    problems.value = data.list
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

async function loadProblemForEdit(problemId: number) {
  editing.value = true
  error.value = ''
  editMessage.value = ''
  try {
    const data = await apiFetch<ProblemDetail>(`/api/admin/problems/${problemId}`)
    editForm.problemId = data.problem.id
    editForm.title = data.problem.title
    editForm.slug = data.problem.slug ?? ''
    editForm.tags = data.problem.tags.join(', ')
    editForm.visible = data.problem.visible
    editForm.statementMarkdown = data.version.statementMarkdown
    editForm.timeLimitMs = data.version.timeLimitMs
    editForm.memoryLimitMb = Math.round(data.version.memoryLimitBytes / 1024 / 1024)
    editForm.outputLimitMb = Math.round(data.version.outputLimitBytes / 1024 / 1024)
    editForm.testCasesText = JSON.stringify(data.version.testCases, null, 2)
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
  editMessage.value = ''
  try {
    const testCases = JSON.parse(editForm.testCasesText || '[]')
    const result = await apiFetch<ProblemDetail>(`/api/problems/${editForm.problemId}`, {
      method: 'PATCH',
      body: JSON.stringify({
        title: editForm.title,
        slug: editForm.slug || undefined,
        tags: editForm.tags
          .split(',')
          .map((tag) => tag.trim())
          .filter(Boolean),
        visible: editForm.visible,
        statementMarkdown: editForm.statementMarkdown,
        timeLimitMs: editForm.timeLimitMs,
        memoryLimitBytes: editForm.memoryLimitMb * 1024 * 1024,
        outputLimitBytes: editForm.outputLimitMb * 1024 * 1024,
        testCases
      })
    })
    editMessage.value = `Saved problem ${result.problem.id} as version ${result.version.version}.`
    await loadProblems()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    editing.value = false
  }
}

async function createProblem() {
  saving.value = true
  error.value = ''
  try {
    const testCases = JSON.parse(form.testCasesText || '[]')
    await apiFetch('/api/problems', {
      method: 'POST',
      body: JSON.stringify({
        title: form.title,
        slug: form.slug || undefined,
        tags: form.tags
          .split(',')
          .map((tag) => tag.trim())
          .filter(Boolean),
        statementMarkdown: form.statementMarkdown,
        timeLimitMs: form.timeLimitMs,
        memoryLimitBytes: form.memoryLimitMb * 1024 * 1024,
        outputLimitBytes: form.outputLimitMb * 1024 * 1024,
        testCases
      })
    })
    form.title = ''
    form.slug = ''
    form.tags = ''
    await loadProblems()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

async function uploadTestdata() {
  if (!uploadForm.problemId || !selectedFile.value) return

  uploading.value = true
  error.value = ''
  uploadMessage.value = ''
  try {
    const formData = new FormData()
    formData.set('file', selectedFile.value)
    const result = await apiFetch<{ caseCount: number }>(
      `/api/problems/${uploadForm.problemId}/testdata`,
      {
        method: 'POST',
        body: formData
      }
    )
    uploadMessage.value = `Uploaded ${result.caseCount} cases for problem ${uploadForm.problemId}.`
    selectedFile.value = null
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    uploading.value = false
  }
}

function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  selectedFile.value = input.files?.[0] ?? null
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
    <section class="page-header">
      <h1>Problems</h1>
      <p>Create statement versions with inline test cases.</p>
    </section>

    <n-alert v-if="!canManage" type="warning" class="page-alert">
      Admin group is required.
    </n-alert>
    <n-alert v-if="error" type="error" class="page-alert">
      {{ error }}
    </n-alert>
    <n-alert v-if="uploadMessage" type="success" class="page-alert">
      {{ uploadMessage }}
    </n-alert>
    <n-alert v-if="editMessage" type="success" class="page-alert">
      {{ editMessage }}
    </n-alert>

    <section v-if="canManage" class="admin-layout">
      <div class="admin-stack">
        <n-card title="Create problem" :bordered="false">
          <n-form :model="form" label-placement="top">
            <n-form-item label="Title">
              <n-input v-model:value="form.title" placeholder="A+B Problem" />
            </n-form-item>
            <n-form-item label="Slug">
              <n-input v-model:value="form.slug" placeholder="a-plus-b" />
            </n-form-item>
            <n-form-item label="Tags">
              <n-input v-model:value="form.tags" placeholder="math, beginner" />
            </n-form-item>
            <n-form-item label="Statement">
              <n-input
                v-model:value="form.statementMarkdown"
                type="textarea"
                :autosize="{ minRows: 6, maxRows: 12 }"
              />
            </n-form-item>
            <div class="form-grid">
              <n-form-item label="Time ms">
                <n-input-number v-model:value="form.timeLimitMs" :min="100" class="full-width" />
              </n-form-item>
              <n-form-item label="Memory MB">
                <n-input-number v-model:value="form.memoryLimitMb" :min="16" class="full-width" />
              </n-form-item>
              <n-form-item label="Output MB">
                <n-input-number v-model:value="form.outputLimitMb" :min="1" class="full-width" />
              </n-form-item>
            </div>
            <n-form-item label="Test cases JSON">
              <n-input
                v-model:value="form.testCasesText"
                type="textarea"
                :autosize="{ minRows: 7, maxRows: 12 }"
              />
            </n-form-item>
            <n-space justify="end" class="form-actions">
              <n-button
                type="primary"
                :loading="saving"
                :disabled="!form.title"
                @click="createProblem"
              >
                Create
              </n-button>
            </n-space>
          </n-form>
        </n-card>

        <n-card title="Upload testdata" :bordered="false">
          <n-form :model="uploadForm" label-placement="top">
            <n-form-item label="Problem ID">
              <n-input-number v-model:value="uploadForm.problemId" :min="1000" class="full-width" />
            </n-form-item>
            <n-form-item label="ZIP file">
              <input type="file" accept=".zip,application/zip" @change="handleFileChange" />
            </n-form-item>
            <n-space justify="end" class="form-actions">
              <n-button
                type="primary"
                :loading="uploading"
                :disabled="!uploadForm.problemId || !selectedFile"
                @click="uploadTestdata"
              >
                Upload
              </n-button>
            </n-space>
          </n-form>
        </n-card>

        <n-card title="Edit problem" :bordered="false">
          <n-form :model="editForm" label-placement="top">
            <n-form-item label="Problem ID">
              <n-input-number v-model:value="editForm.problemId" :min="1000" class="full-width" />
            </n-form-item>
            <n-space justify="end">
              <n-button
                secondary
                :loading="editing"
                :disabled="!editForm.problemId"
                @click="editForm.problemId && loadProblemForEdit(editForm.problemId)"
              >
                Load
              </n-button>
            </n-space>
            <n-form-item label="Title">
              <n-input v-model:value="editForm.title" placeholder="A+B Problem" />
            </n-form-item>
            <n-form-item label="Slug">
              <n-input v-model:value="editForm.slug" placeholder="a-plus-b" />
            </n-form-item>
            <n-form-item label="Visible">
              <n-checkbox v-model:checked="editForm.visible">Show in public problem set</n-checkbox>
            </n-form-item>
            <n-form-item label="Tags">
              <n-input v-model:value="editForm.tags" placeholder="math, beginner" />
            </n-form-item>
            <n-form-item label="Statement">
              <n-input
                v-model:value="editForm.statementMarkdown"
                type="textarea"
                :autosize="{ minRows: 6, maxRows: 12 }"
              />
            </n-form-item>
            <div class="form-grid">
              <n-form-item label="Time ms">
                <n-input-number
                  v-model:value="editForm.timeLimitMs"
                  :min="100"
                  class="full-width"
                />
              </n-form-item>
              <n-form-item label="Memory MB">
                <n-input-number
                  v-model:value="editForm.memoryLimitMb"
                  :min="16"
                  class="full-width"
                />
              </n-form-item>
              <n-form-item label="Output MB">
                <n-input-number
                  v-model:value="editForm.outputLimitMb"
                  :min="1"
                  class="full-width"
                />
              </n-form-item>
            </div>
            <n-form-item label="Test cases JSON">
              <n-input
                v-model:value="editForm.testCasesText"
                type="textarea"
                :autosize="{ minRows: 7, maxRows: 12 }"
              />
            </n-form-item>
            <n-space justify="end" class="form-actions">
              <n-button
                type="primary"
                :loading="editing"
                :disabled="!editForm.problemId || !editForm.title"
                @click="updateProblem"
              >
                Save new version
              </n-button>
            </n-space>
          </n-form>
        </n-card>
      </div>

      <n-data-table
        :columns="columns"
        :data="problems"
        :bordered="false"
        :loading="loading"
        class="admin-table"
      />
    </section>
  </main>
</template>
