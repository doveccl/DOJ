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
  NModal,
  NSpace,
  NSwitch,
  NTag
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { computed, h, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../../api'
import CodeEditor from '../../components/CodeEditor.vue'
import { useAuthStore } from '../../stores/auth'

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
const { t } = useI18n()
const canManage = computed(() => auth.user?.groups.includes('admin') ?? false)
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const showConfigModal = ref(false)
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

const columns = computed<DataTableColumns<LanguageRow>>(() => [
  { title: t('common.id'), key: 'id', width: 96 },
  { title: t('admin.name'), key: 'name' },
  { title: t('admin.languages.source'), key: 'sourceFile' },
  {
    title: t('admin.status'),
    key: 'enabled',
    render(row) {
      return h(NTag, { bordered: false, type: row.enabled ? 'success' : 'default' }, () =>
        row.enabled ? t('admin.enabled') : t('admin.disabled')
      )
    }
  },
  { title: t('admin.sortOrder'), key: 'sortOrder', width: 96 },
  {
    title: t('admin.actions'),
    key: 'action',
    width: 120,
    render(row) {
      return h(
        NButton,
        {
          size: 'small',
          onClick: () => editLanguage(row)
        },
        () => t('admin.edit')
      )
    }
  }
])

async function loadLanguages() {
  loading.value = true
  error.value = ''
  try {
    const data = await apiFetch<{ list: LanguageRow[] }>('/api/admin/languages')
    languages.value = data.list
    if (!form.id && data.list.length) editLanguage(data.list[0], false)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

function editLanguage(language: LanguageRow, open = true) {
  form.id = language.id
  form.name = language.name
  form.enabled = language.enabled
  form.sourceFile = language.sourceFile
  form.dockerfile = language.dockerfile
  form.commandText = language.command.join('\n')
  form.sortOrder = language.sortOrder
  showConfigModal.value = open
}

function newLanguage() {
  form.id = ''
  form.name = ''
  form.enabled = true
  form.sourceFile = 'main.cpp'
  form.dockerfile =
    'FROM gcc:latest\nWORKDIR /workspace\nCOPY main.cpp /workspace/main.cpp\nRUN g++ -std=c++20 -O2 -pipe -static -s main.cpp -o main\nCMD ["/workspace/main"]'
  form.commandText = ''
  form.sortOrder = 100
  showConfigModal.value = true
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
    showConfigModal.value = false
    await loadLanguages()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

watch(canManage, (allowed) => {
  if (allowed) loadLanguages()
})

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
    <n-alert v-if="!canManage" type="warning" class="page-alert">
      {{ t('admin.requireAdmin') }}
    </n-alert>

    <n-alert v-if="error" type="error" class="page-alert">
      {{ error }}
    </n-alert>

    <n-card v-if="canManage" :bordered="false">
      <n-space justify="end" class="table-toolbar">
        <n-button type="primary" @click="newLanguage">{{ t('admin.languages.new') }}</n-button>
      </n-space>
      <n-data-table
        :columns="columns"
        :data="languages"
        :bordered="false"
        :loading="loading"
        class="admin-table"
      />
    </n-card>

    <n-modal
      v-model:show="showConfigModal"
      preset="card"
      :title="t('admin.languages.config')"
      class="form-modal"
    >
      <n-form :model="form" label-placement="top">
        <div class="form-grid two">
          <n-form-item :label="t('common.id')">
            <n-input v-model:value="form.id" placeholder="cpp" />
          </n-form-item>
          <n-form-item :label="t('admin.name')">
            <n-input v-model:value="form.name" placeholder="C++ 20" />
          </n-form-item>
        </div>
        <div class="form-grid two">
          <n-form-item :label="t('admin.languages.sourceFile')">
            <n-input v-model:value="form.sourceFile" placeholder="main.cpp" />
          </n-form-item>
          <n-form-item :label="t('admin.sortOrder')">
            <n-input-number v-model:value="form.sortOrder" class="full-width" :min="0" />
          </n-form-item>
        </div>
        <n-form-item :label="t('admin.languages.dockerfile')">
          <code-editor v-model="form.dockerfile" />
        </n-form-item>
        <n-form-item :label="t('admin.languages.commandOverride')">
          <n-input
            v-model:value="form.commandText"
            type="textarea"
            :placeholder="t('admin.languages.commandPlaceholder')"
            :autosize="{ minRows: 3, maxRows: 8 }"
          />
        </n-form-item>
        <n-space align="center" justify="space-between" class="form-actions">
          <n-space align="center">
            <n-switch v-model:value="form.enabled" />
            <span>{{ form.enabled ? t('admin.enabled') : t('admin.disabled') }}</span>
          </n-space>
          <n-space>
            <n-button @click="showConfigModal = false">{{ t('admin.cancel') }}</n-button>
            <n-button
              type="primary"
              :loading="saving"
              :disabled="!form.id || !form.name || !form.sourceFile || !form.dockerfile"
              @click="saveLanguage"
            >
              {{ t('admin.save') }}
            </n-button>
          </n-space>
        </n-space>
      </n-form>
    </n-modal>
  </main>
</template>
