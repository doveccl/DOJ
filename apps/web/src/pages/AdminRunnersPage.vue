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

const columns: DataTableColumns<RunnerRow> = [
  { title: 'Key', key: 'key', width: 140 },
  { title: 'Name', key: 'name', width: 160 },
  {
    title: 'Endpoint',
    key: 'endpoint',
    ellipsis: {
      tooltip: true
    },
    render(row) {
      return row.endpoint || 'local'
    }
  },
  {
    title: 'Status',
    key: 'enabled',
    render(row) {
      return h(NTag, { bordered: false, type: row.enabled ? 'success' : 'default' }, () =>
        row.enabled ? 'enabled' : 'disabled'
      )
    }
  },
  { title: 'Concurrency', key: 'concurrency', width: 128 },
  {
    title: 'Action',
    key: 'action',
    width: 180,
    render(row) {
      return h(NSpace, { size: 8 }, () => [
        h(NButton, { size: 'small', onClick: () => editRunner(row) }, () => 'Edit'),
        h(
          NButton,
          {
            size: 'small',
            loading: checkingId.value === row.id,
            onClick: () => checkRunner(row)
          },
          () => 'Check'
        )
      ])
    }
  }
]

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
      <h1>Runners</h1>
      <p>Configure local or remote Docker API judging backends.</p>
    </section>

    <n-alert v-if="!canManage" type="warning" class="page-alert">
      Admin group is required.
    </n-alert>

    <n-alert v-if="error" type="error" class="page-alert">
      {{ error }}
    </n-alert>
    <n-alert v-if="checkMessage" type="success" class="page-alert">
      {{ checkMessage }}
    </n-alert>

    <n-card v-if="canManage" :bordered="false">
      <n-space justify="end" class="table-toolbar">
        <n-button type="primary" @click="newRunner">New runner</n-button>
      </n-space>
      <n-data-table
        :columns="columns"
        :data="runners"
        :bordered="false"
        :loading="loading"
        class="admin-table"
      />
    </n-card>

    <n-modal v-model:show="showConfigModal" preset="card" title="Runner config" class="form-modal">
      <n-form :model="form" label-placement="top">
        <n-form-item label="Key">
          <n-input v-model:value="form.key" placeholder="remote-1" />
        </n-form-item>
        <n-form-item label="Name">
          <n-input v-model:value="form.name" placeholder="Remote Docker 1" />
        </n-form-item>
        <n-form-item label="Docker endpoint">
          <n-input
            v-model:value="form.endpoint"
            placeholder="unix:///var/run/docker.sock or https://user:pass@host"
          />
        </n-form-item>
        <div class="form-grid two">
          <n-form-item label="Concurrency">
            <n-input-number v-model:value="form.concurrency" class="full-width" :min="1" />
          </n-form-item>
          <n-form-item label="Sort order">
            <n-input-number v-model:value="form.sortOrder" class="full-width" :min="0" />
          </n-form-item>
        </div>
        <n-space align="center" justify="space-between" class="form-actions">
          <n-space align="center">
            <n-switch v-model:value="form.enabled" />
            <span>{{ form.enabled ? 'Enabled' : 'Disabled' }}</span>
          </n-space>
          <n-space>
            <n-button @click="showConfigModal = false">Cancel</n-button>
            <n-button
              type="primary"
              :loading="saving"
              :disabled="!form.key || !form.name"
              @click="saveRunner"
            >
              Save
            </n-button>
          </n-space>
        </n-space>
      </n-form>
    </n-modal>
  </main>
</template>
