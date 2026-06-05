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
import { apiFetch } from '../api'
import { useAuthStore } from '../stores/auth'

interface RunnerRow {
  id: number
  key: string
  name: string
  enabled: boolean
  kind: string
  endpoint: string | null
  concurrency: number
  sortOrder: number
}

const auth = useAuthStore()
const { t } = useI18n()
const canManage = computed(() => auth.user?.groups.includes('admin') ?? false)
const loading = ref(true)
const saving = ref(false)
const checkingId = ref<number | null>(null)
const error = ref('')
const checkMessage = ref('')
const showConfigModal = ref(false)
const runners = ref<RunnerRow[]>([])
const form = reactive({
  key: '',
  name: '',
  enabled: true,
  endpoint: '',
  concurrency: 2,
  sortOrder: 100
})

const columns = computed<DataTableColumns<RunnerRow>>(() => [
  { title: t('admin.key'), key: 'key', width: 140 },
  { title: t('admin.name'), key: 'name', width: 160 },
  {
    title: t('admin.runners.endpoint'),
    key: 'endpoint',
    ellipsis: {
      tooltip: true
    },
    render(row) {
      return row.endpoint || t('admin.runners.local')
    }
  },
  {
    title: t('admin.status'),
    key: 'enabled',
    render(row) {
      return h(NTag, { bordered: false, type: row.enabled ? 'success' : 'default' }, () =>
        row.enabled ? t('admin.enabled') : t('admin.disabled')
      )
    }
  },
  { title: t('admin.runners.concurrency'), key: 'concurrency', width: 128 },
  {
    title: t('admin.actions'),
    key: 'action',
    width: 180,
    render(row) {
      return h(NSpace, { size: 8 }, () => [
        h(NButton, { size: 'small', onClick: () => editRunner(row) }, () => t('admin.edit')),
        h(
          NButton,
          {
            size: 'small',
            loading: checkingId.value === row.id,
            onClick: () => checkRunner(row)
          },
          () => t('admin.runners.check')
        )
      ])
    }
  }
])

async function loadRunners() {
  loading.value = true
  error.value = ''
  try {
    const data = await apiFetch<{ list: RunnerRow[] }>('/api/admin/runners')
    runners.value = data.list
    if (!form.key && data.list.length) editRunner(data.list[0], false)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

function editRunner(runner: RunnerRow, open = true) {
  form.key = runner.key
  form.name = runner.name
  form.enabled = runner.enabled
  form.endpoint = runner.endpoint ?? ''
  form.concurrency = runner.concurrency
  form.sortOrder = runner.sortOrder
  showConfigModal.value = open
}

function newRunner() {
  form.key = ''
  form.name = ''
  form.enabled = true
  form.endpoint = 'https://user:pass@docker.example.com'
  form.concurrency = 1
  form.sortOrder = 100
  showConfigModal.value = true
}

async function saveRunner() {
  saving.value = true
  error.value = ''
  try {
    await apiFetch('/api/admin/runners', {
      method: 'POST',
      body: JSON.stringify({
        key: form.key,
        name: form.name,
        enabled: form.enabled,
        kind: 'docker',
        endpoint: form.endpoint || undefined,
        concurrency: form.concurrency,
        sortOrder: form.sortOrder
      })
    })
    showConfigModal.value = false
    await loadRunners()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

async function checkRunner(runner: RunnerRow) {
  checkingId.value = runner.id
  checkMessage.value = ''
  error.value = ''
  try {
    const result = await apiFetch<{ version: string; apiVersion: string }>(
      `/api/admin/runners/${runner.id}/check`,
      {
        method: 'POST'
      }
    )
    checkMessage.value = `${runner.key}: Docker ${result.version} / API ${result.apiVersion}`
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    checkingId.value = null
  }
}

watch(canManage, (allowed) => {
  if (allowed) loadRunners()
})

onMounted(() => {
  if (canManage.value) {
    loadRunners()
  } else {
    loading.value = false
  }
})
</script>

<template>
  <main class="page">
    <section class="page-header">
      <h1>{{ t('admin.runners.title') }}</h1>
      <p>{{ t('admin.runners.subtitle') }}</p>
    </section>

    <n-alert v-if="!canManage" type="warning" class="page-alert">
      {{ t('admin.requireAdmin') }}
    </n-alert>

    <n-alert v-if="error" type="error" class="page-alert">
      {{ error }}
    </n-alert>
    <n-alert v-if="checkMessage" type="success" class="page-alert">
      {{ checkMessage }}
    </n-alert>
    <n-alert v-if="canManage" type="warning" class="page-alert">
      {{ t('admin.runners.remoteNote') }}
    </n-alert>

    <n-card v-if="canManage" :bordered="false">
      <n-space justify="end" class="table-toolbar">
        <n-button type="primary" @click="newRunner">{{ t('admin.runners.new') }}</n-button>
      </n-space>
      <n-data-table
        :columns="columns"
        :data="runners"
        :bordered="false"
        :loading="loading"
        class="admin-table"
      />
    </n-card>

    <n-modal
      v-model:show="showConfigModal"
      preset="card"
      :title="t('admin.runners.config')"
      class="form-modal"
    >
      <n-form :model="form" label-placement="top">
        <n-form-item :label="t('admin.key')">
          <n-input v-model:value="form.key" placeholder="remote-1" />
        </n-form-item>
        <n-form-item :label="t('admin.name')">
          <n-input v-model:value="form.name" placeholder="Remote Docker 1" />
        </n-form-item>
        <n-form-item :label="t('admin.runners.endpoint')">
          <n-input
            v-model:value="form.endpoint"
            placeholder="unix:///var/run/docker.sock or https://user:pass@host"
          />
        </n-form-item>
        <div class="form-grid two">
          <n-form-item :label="t('admin.runners.concurrency')">
            <n-input-number v-model:value="form.concurrency" class="full-width" :min="1" />
          </n-form-item>
          <n-form-item :label="t('admin.sortOrder')">
            <n-input-number v-model:value="form.sortOrder" class="full-width" :min="0" />
          </n-form-item>
        </div>
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
              :disabled="!form.key || !form.name"
              @click="saveRunner"
            >
              {{ t('admin.save') }}
            </n-button>
          </n-space>
        </n-space>
      </n-form>
    </n-modal>
  </main>
</template>
