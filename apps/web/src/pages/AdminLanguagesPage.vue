<script setup lang="ts">
import {
  NAlert,
  NButton,
  NCard,
  NDataTable,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSpace,
  NSwitch,
  NTag
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { computed, h, onMounted, reactive, ref, watch } from 'vue'
import { apiFetch } from '../api'
import { useAuthStore } from '../stores/auth'

interface LanguageRow {
  id: string
  name: string
  enabled: boolean
  sourceFile: string
  dockerfile: string
  command: string[]
  sortOrder: number
}

const auth = useAuthStore()
const canManage = computed(() => auth.user?.groups.includes('admin') ?? false)
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const languages = ref<LanguageRow[]>([])
const form = reactive({
  id: '',
  name: '',
  enabled: true,
  sourceFile: '',
  dockerfile: '',
  commandText: '',
  sortOrder: 100
})

const columns: DataTableColumns<LanguageRow> = [
  { title: 'ID', key: 'id', width: 96 },
  { title: 'Name', key: 'name' },
  { title: 'Source', key: 'sourceFile' },
  {
    title: 'Status',
    key: 'enabled',
    render(row) {
      return h(NTag, { bordered: false, type: row.enabled ? 'success' : 'default' }, () =>
        row.enabled ? 'enabled' : 'disabled'
      )
    }
  },
  { title: 'Order', key: 'sortOrder', width: 96 },
  {
    title: 'Action',
    key: 'action',
    width: 120,
    render(row) {
      return h(
        NButton,
        {
          size: 'small',
          onClick: () => editLanguage(row)
        },
        () => 'Edit'
      )
    }
  }
]

async function loadLanguages() {
  loading.value = true
  error.value = ''
  try {
    const data = await apiFetch<{ list: LanguageRow[] }>('/api/admin/languages')
    languages.value = data.list
    if (!form.id && data.list.length) editLanguage(data.list[0])
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

function editLanguage(language: LanguageRow) {
  form.id = language.id
  form.name = language.name
  form.enabled = language.enabled
  form.sourceFile = language.sourceFile
  form.dockerfile = language.dockerfile
  form.commandText = language.command.join('\n')
  form.sortOrder = language.sortOrder
}

function newLanguage() {
  form.id = ''
  form.name = ''
  form.enabled = true
  form.sourceFile = 'main.cpp'
  form.dockerfile = 'FROM gcc:latest\nWORKDIR /workspace\nCOPY main.cpp /workspace/main.cpp\nRUN g++ -std=c++20 -O2 -pipe -static -s main.cpp -o main\nCMD ["/workspace/main"]'
  form.commandText = ''
  form.sortOrder = 100
}

async function saveLanguage() {
  saving.value = true
  error.value = ''
  try {
    await apiFetch('/api/admin/languages', {
      method: 'POST',
      body: JSON.stringify({
        id: form.id,
        name: form.name,
        enabled: form.enabled,
        sourceFile: form.sourceFile,
        dockerfile: form.dockerfile,
        command: form.commandText
          .split('\n')
          .map((part) => part.trim())
          .filter(Boolean),
        sortOrder: form.sortOrder
      })
    })
    await loadLanguages()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

watch(
  canManage,
  (allowed) => {
    if (allowed) loadLanguages()
  }
)

onMounted(() => {
  if (canManage.value) {
    loadLanguages()
  } else {
    loading.value = false
  }
})
</script>

<template>
  <main class="page">
    <section class="page-header">
      <h1>Languages</h1>
      <p>Configure enabled judging languages and their Docker build recipes.</p>
    </section>

    <n-alert v-if="!canManage" type="warning" class="page-alert">
      Admin group is required.
    </n-alert>

    <n-alert v-if="error" type="error" class="page-alert">
      {{ error }}
    </n-alert>

    <section v-if="canManage" class="admin-layout">
      <n-card title="Language config" :bordered="false">
        <n-form :model="form" label-placement="top">
          <n-form-item label="ID">
            <n-input v-model:value="form.id" placeholder="cpp" />
          </n-form-item>
          <n-form-item label="Name">
            <n-input v-model:value="form.name" placeholder="C++ 20" />
          </n-form-item>
          <n-form-item label="Source file">
            <n-input v-model:value="form.sourceFile" placeholder="main.cpp" />
          </n-form-item>
          <n-form-item label="Sort order">
            <n-input-number v-model:value="form.sortOrder" class="full-width" :min="0" />
          </n-form-item>
          <n-form-item label="Dockerfile">
            <n-input
              v-model:value="form.dockerfile"
              type="textarea"
              :autosize="{ minRows: 10, maxRows: 18 }"
            />
          </n-form-item>
          <n-form-item label="Command override">
            <n-input
              v-model:value="form.commandText"
              type="textarea"
              placeholder="One argv item per line; leave empty to use Docker CMD"
              :autosize="{ minRows: 3, maxRows: 8 }"
            />
          </n-form-item>
          <n-space align="center" justify="space-between">
            <n-space align="center">
              <n-switch v-model:value="form.enabled" />
              <span>{{ form.enabled ? 'Enabled' : 'Disabled' }}</span>
            </n-space>
            <n-space>
              <n-button @click="newLanguage">New</n-button>
              <n-button
                type="primary"
                :loading="saving"
                :disabled="!form.id || !form.name || !form.sourceFile || !form.dockerfile"
                @click="saveLanguage"
              >
                Save
              </n-button>
            </n-space>
          </n-space>
        </n-form>
      </n-card>

      <n-data-table
        :columns="columns"
        :data="languages"
        :bordered="false"
        :loading="loading"
        class="admin-table"
      />
    </section>
  </main>
</template>
