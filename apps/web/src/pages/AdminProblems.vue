<script setup lang="ts">
import {
  NAlert,
  NButton,
  NCard,
  NCheckbox,
  NDataTable,
  NDynamicTags,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NModal,
  NSpace,
  NUpload
} from 'naive-ui'
import type { DataTableColumns, UploadFileInfo } from 'naive-ui'
import { computed, h, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
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
  problem: ProblemRow
  version: {
    id: number
    version: number
    statementMarkdown: string
    timeLimitMs: number
    memoryLimitBytes: number
  }
}

const auth = useAuthStore()
const { t } = useI18n()
const canManage = computed(() => auth.user?.groups.includes('admin') ?? false)
const loading = ref(true)
const saving = ref(false)
const editing = ref(false)
const uploading = ref(false)
const error = ref('')
const uploadMessage = ref('')
const editMessage = ref('')
const showCreateModal = ref(false)
const showUploadModal = ref(false)
const showEditModal = ref(false)
const problems = ref<ProblemRow[]>([])
const selectedFile = ref<File | null>(null)
const form = reactive({
  title: '',
  tags: [] as string[],
  statementMarkdown: '# New Problem\n\nDescribe the task.',
  timeLimitMs: 1000,
  memoryLimitMb: 256
})
const uploadForm = reactive({
  problemId: null as number | null
})
const editForm = reactive({
  problemId: null as number | null,
  title: '',
  tags: [] as string[],
  visible: true,
  statementMarkdown: '',
  timeLimitMs: 1000,
  memoryLimitMb: 256
})

const columns = computed<DataTableColumns<ProblemRow>>(() => [
  { title: t('common.id'), key: 'id', width: 96 },
  { title: t('common.title'), key: 'title', minWidth: 240 },
  {
    title: t('admin.problems.visible'),
    key: 'visible',
    width: 110,
    render(row) {
      return row.visible ? t('admin.problems.yes') : t('admin.problems.no')
    }
  },
  {
    title: t('common.tags'),
    key: 'tags',
    minWidth: 160,
    render(row) {
      return row.tags.join(', ') || '-'
    }
  },
  { title: t('common.solved'), key: 'solvedCount', width: 110 },
  {
    title: t('admin.actions'),
    key: 'action',
    width: 180,
    render(row) {
      return h(NSpace, { size: 8 }, () => [
        h(
          NButton,
          {
            size: 'small',
            secondary: true,
            onClick: () => loadProblemForEdit(row.id)
          },
          () => t('admin.edit')
        ),
        h(
          NButton,
          {
            size: 'small',
            secondary: true,
            onClick: () => {
              uploadForm.problemId = row.id
              selectedFile.value = null
              showUploadModal.value = true
            }
          },
          () => t('admin.problems.testdata')
        )
      ])
    }
  }
])

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
    editForm.tags = [...data.problem.tags]
    editForm.visible = data.problem.visible
    editForm.statementMarkdown = data.version.statementMarkdown
    editForm.timeLimitMs = data.version.timeLimitMs
    editForm.memoryLimitMb = Math.round(data.version.memoryLimitBytes / 1024 / 1024)
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
  editMessage.value = ''
  try {
    const result = await apiFetch<ProblemDetail>(`/api/problems/${editForm.problemId}`, {
      method: 'PATCH',
      body: JSON.stringify({
        title: editForm.title,
        tags: editForm.tags,
        visible: editForm.visible,
        statementMarkdown: editForm.statementMarkdown,
        timeLimitMs: editForm.timeLimitMs,
        memoryLimitBytes: editForm.memoryLimitMb * 1024 * 1024
      })
    })
    editMessage.value = t('admin.problems.saved', {
      id: result.problem.id,
      version: result.version.version
    })
    showEditModal.value = false
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
    uploadMessage.value = t('admin.problems.uploaded', {
      count: result.caseCount,
      id: uploadForm.problemId
    })
    selectedFile.value = null
    showUploadModal.value = false
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    uploading.value = false
  }
}

function handleUploadChange(options: { fileList: UploadFileInfo[] }) {
  selectedFile.value = options.fileList[0]?.file ?? null
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
      <h1>{{ t('admin.problems.title') }}</h1>
      <p>{{ t('admin.problems.subtitle') }}</p>
    </section>

    <n-alert v-if="!canManage" type="warning" class="page-alert">
      {{ t('admin.requireAdmin') }}
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

    <n-card v-if="canManage" :bordered="false">
      <n-space justify="end" class="table-toolbar">
        <n-button type="primary" @click="showCreateModal = true">
          {{ t('admin.problems.create') }}
        </n-button>
      </n-space>
      <n-data-table
        :columns="columns"
        :data="problems"
        :bordered="false"
        :loading="loading"
        class="admin-table"
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
          <n-input
            v-model:value="form.statementMarkdown"
            type="textarea"
            :autosize="{ minRows: 6, maxRows: 12 }"
          />
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
      v-model:show="showUploadModal"
      preset="card"
      :title="t('admin.problems.uploadTestdata')"
      class="form-modal narrow"
    >
      <n-form :model="uploadForm" label-placement="top">
        <n-form-item :label="t('admin.problems.zipFile')">
          <n-upload
            :max="1"
            accept=".zip,application/zip"
            :default-upload="false"
            @change="handleUploadChange"
          >
            <n-button>{{ t('admin.upload') }}</n-button>
          </n-upload>
        </n-form-item>
        <n-space justify="end" class="form-actions">
          <n-button @click="showUploadModal = false">{{ t('admin.cancel') }}</n-button>
          <n-button
            type="primary"
            :loading="uploading"
            :disabled="!uploadForm.problemId || !selectedFile"
            @click="uploadTestdata"
          >
            {{ t('admin.upload') }}
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
        <n-form-item :label="t('admin.problems.problemId')">
          <n-input-number v-model:value="editForm.problemId" :min="1000" class="full-width" />
        </n-form-item>
        <n-space justify="end">
          <n-button
            secondary
            :loading="editing"
            :disabled="!editForm.problemId"
            @click="editForm.problemId && loadProblemForEdit(editForm.problemId)"
          >
            {{ t('admin.load') }}
          </n-button>
        </n-space>
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
          <n-input
            v-model:value="editForm.statementMarkdown"
            type="textarea"
            :autosize="{ minRows: 6, maxRows: 12 }"
          />
        </n-form-item>
        <div class="form-grid two">
          <n-form-item :label="t('admin.problems.timeMs')">
            <n-input-number v-model:value="editForm.timeLimitMs" :min="100" class="full-width" />
          </n-form-item>
          <n-form-item :label="t('admin.problems.memoryMb')">
            <n-input-number v-model:value="editForm.memoryLimitMb" :min="16" class="full-width" />
          </n-form-item>
        </div>
        <n-space justify="end" class="form-actions">
          <n-button @click="showEditModal = false">{{ t('admin.cancel') }}</n-button>
          <n-button
            type="primary"
            :loading="editing"
            :disabled="!editForm.problemId || !editForm.title"
            @click="updateProblem"
          >
            {{ t('admin.saveNewVersion') }}
          </n-button>
        </n-space>
      </n-form>
    </n-modal>
  </main>
</template>
