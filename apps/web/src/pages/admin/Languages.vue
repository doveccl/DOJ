<script setup lang="ts">
import { NButton, NSpace } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../../api'
import CodeEditor from '../../components/CodeEditor.vue'
import { useAuthStore } from '../../stores/auth'

interface LanguageRow {
  id: string
  name: string
  source: string
  dockerfile: string
  sort?: number
}

const auth = useAuthStore()
const { t } = useI18n()
const canManage = computed(() => auth.user?.admin ?? false)
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const showConfigModal = ref(false)
const languages = ref<LanguageRow[]>([])
const form = reactive({
  id: '',
  name: '',
  source: '',
  dockerfile: '',
  sort: 0
})

const columns = computed<DataTableColumns<LanguageRow>>(() => [
  { title: t('common.id'), key: 'id', width: 96 },
  { title: t('admin.name'), key: 'name' },
  { title: t('admin.languages.source'), key: 'source' },
  {
    title: t('admin.languages.dockerfile'),
    key: 'dockerfile',
    minWidth: 320,
    render(row) {
      return h('pre', { class: 'code-block' }, [h('code', row.dockerfile)])
    }
  },
  {
    title: t('admin.sortOrder'),
    key: 'sort',
    width: 96
  },
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
            onClick: () => editLanguage(row)
          },
          () => t('admin.edit')
        ),
        h(
          NButton,
          {
            size: 'small',
            tertiary: true,
            type: 'error',
            onClick: () => deleteLanguage(row.id)
          },
          () => t('admin.delete')
        )
      ])
    }
  }
])

async function loadLanguages() {
  loading.value = true
  error.value = ''
  try {
    const list = await apiFetch<LanguageRow[]>('/api/admin/languages')
    languages.value = list
    if (!form.id && list.length) editLanguage(list[0], false)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

function editLanguage(language: LanguageRow, showModal = true) {
  form.id = language.id
  form.name = language.name
  form.source = language.source
  form.dockerfile = language.dockerfile
  form.sort = language.sort ?? 0
  showConfigModal.value = showModal
}

function newLanguage() {
  form.id = ''
  form.name = ''
  form.source = 'main.cc'
  form.dockerfile =
    'FROM gcc:14\nWORKDIR /app\nCOPY main.cc /app/main.cc\nRUN g++ -std=c++20 -O2 -pipe -static -s -o /app/main /app/main.cc\nCMD ["/app/main"]'
  form.sort = 0
  showConfigModal.value = true
}

async function saveLanguage() {
  saving.value = true
  error.value = ''
  try {
    const editing = languages.value.some((language) => language.id === form.id)
    await apiFetch(editing ? `/api/admin/languages/${form.id}` : '/api/admin/languages', {
      method: editing ? 'PATCH' : 'POST',
      body: JSON.stringify({
        ...(editing ? {} : { id: form.id }),
        name: form.name,
        source: form.source,
        dockerfile: form.dockerfile,
        sort: form.sort
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

async function deleteLanguage(id: string) {
  error.value = ''
  try {
    await apiFetch(`/api/admin/languages/${id}`, { method: 'DELETE' })
    if (form.id === id) form.id = ''
    await loadLanguages()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
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
        :scroll-x="860"
        class="admin-table"
      />
    </n-card>

    <n-modal
      v-if="showConfigModal"
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
          <n-form-item :label="t('admin.languages.source')">
            <n-input v-model:value="form.source" placeholder="main.cc" />
          </n-form-item>
          <n-form-item :label="t('admin.sortOrder')">
            <n-input-number v-model:value="form.sort" class="full-width" :min="0" />
          </n-form-item>
        </div>
        <n-form-item :label="t('admin.languages.dockerfile')">
          <code-editor v-model="form.dockerfile" />
        </n-form-item>
        <n-space justify="end" class="form-actions">
          <n-button @click="showConfigModal = false">{{ t('admin.cancel') }}</n-button>
          <n-button
            type="primary"
            :loading="saving"
            :disabled="!form.id || !form.name || !form.source || !form.dockerfile"
            @click="saveLanguage"
          >
            {{ t('admin.save') }}
          </n-button>
        </n-space>
      </n-form>
    </n-modal>
  </main>
</template>

<style scoped>
.code-block {
  max-height: 180px;
  padding: 10px;
  overflow: auto;
  white-space: pre-wrap;
  border-radius: var(--radius-md);
  background: color-mix(in srgb, var(--surface-bg) 92%, var(--text-color) 8%);
}
</style>
