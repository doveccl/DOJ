<script setup lang="ts">
import {
  NAlert,
  NButton,
  NCard,
  NDataTable,
  NDynamicTags,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NModal,
  NSpace,
  NSwitch,
  NTag,
  NTooltip
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { computed, h, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../../api'
import { useAuthStore } from '../../stores/auth'

interface AgentRow {
  id: number
  key: string
  name: string
  enabled: boolean
  labels: string[]
  concurrency: number
  sortOrder: number
  lastSeenAt: string | null
}

const auth = useAuthStore()
const { t } = useI18n()
const canManage = computed(() => auth.user?.groups.includes('admin') ?? false)
const loading = ref(true)
const saving = ref(false)
const rotatingId = ref<number | null>(null)
const error = ref('')
const tokenMessage = ref('')
const showConfigModal = ref(false)
const agents = ref<AgentRow[]>([])
const form = reactive({
  key: '',
  name: '',
  enabled: true,
  labels: ['local'],
  token: '',
  concurrency: 2,
  sortOrder: 100
})

const columns = computed<DataTableColumns<AgentRow>>(() => [
  { title: t('admin.key'), key: 'key', width: 140 },
  { title: t('admin.name'), key: 'name', width: 160 },
  {
    title: t('admin.agents.labels'),
    key: 'labels',
    render(row) {
      return h(NSpace, { size: 6 }, () =>
        row.labels.map((label) =>
          h(NTag, { size: 'small', bordered: false, type: 'info' }, () => label)
        )
      )
    }
  },
  {
    title: t('admin.status'),
    key: 'enabled',
    width: 130,
    render(row) {
      const online = isOnline(row)
      return h(NSpace, { size: 6 }, () => [
        h(NTag, { bordered: false, type: row.enabled ? 'success' : 'default' }, () =>
          row.enabled ? t('admin.enabled') : t('admin.disabled')
        ),
        row.enabled
          ? h(NTag, { bordered: false, type: online ? 'success' : 'warning' }, () =>
              online ? t('admin.agents.online') : t('admin.agents.offline')
            )
          : null
      ])
    }
  },
  { title: t('admin.agents.concurrency'), key: 'concurrency', width: 116 },
  {
    title: t('admin.agents.lastSeen'),
    key: 'lastSeenAt',
    width: 190,
    render(row) {
      return row.lastSeenAt ? new Date(row.lastSeenAt).toLocaleString() : t('admin.agents.never')
    }
  },
  {
    title: t('admin.actions'),
    key: 'action',
    width: 190,
    render(row) {
      return h(NSpace, { size: 8 }, () => [
        h(NButton, { size: 'small', onClick: () => editAgent(row) }, () => t('admin.edit')),
        h(
          NTooltip,
          {},
          {
            trigger: () =>
              h(
                NButton,
                {
                  size: 'small',
                  loading: rotatingId.value === row.id,
                  onClick: () => rotateToken(row)
                },
                () => t('admin.agents.rotate')
              ),
            default: () => t('admin.agents.rotateHint')
          }
        )
      ])
    }
  }
])

async function loadAgents() {
  loading.value = true
  error.value = ''
  try {
    const data = await apiFetch<{ list: AgentRow[] }>('/api/admin/agents')
    agents.value = data.list
    if (!form.key && data.list.length) editAgent(data.list[0], false)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

function editAgent(agent: AgentRow, open = true) {
  form.key = agent.key
  form.name = agent.name
  form.enabled = agent.enabled
  form.labels = [...agent.labels]
  form.token = ''
  form.concurrency = agent.concurrency
  form.sortOrder = agent.sortOrder
  showConfigModal.value = open
}

function newAgent() {
  form.key = ''
  form.name = ''
  form.enabled = true
  form.labels = []
  form.token = ''
  form.concurrency = 1
  form.sortOrder = 100
  showConfigModal.value = true
}

async function saveAgent() {
  saving.value = true
  error.value = ''
  tokenMessage.value = ''
  try {
    const saved = await apiFetch<AgentRow & { token?: string }>('/api/admin/agents', {
      method: 'POST',
      body: JSON.stringify({
        key: form.key,
        name: form.name,
        enabled: form.enabled,
        labels: form.labels,
        token: form.token || undefined,
        concurrency: form.concurrency,
        sortOrder: form.sortOrder
      })
    })
    if (saved.token) tokenMessage.value = buildAgentCommand(saved.key, saved.token)
    showConfigModal.value = false
    await loadAgents()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

async function rotateToken(agent: AgentRow) {
  rotatingId.value = agent.id
  tokenMessage.value = ''
  error.value = ''
  try {
    const result = await apiFetch<{ key: string; token: string }>(
      `/api/admin/agents/${agent.id}/rotate-token`,
      {
        method: 'POST'
      }
    )
    tokenMessage.value = buildAgentCommand(result.key, result.token)
    await loadAgents()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    rotatingId.value = null
  }
}

function isOnline(agent: AgentRow) {
  if (!agent.lastSeenAt) return false
  return Date.now() - new Date(agent.lastSeenAt).getTime() < 90_000
}

function buildAgentCommand(key: string, token: string) {
  return `DOJ_AGENT_KEY=${key} DOJ_AGENT_TOKEN=${token} DOJ_WORKER_WS_URL=ws://localhost:7975/agents/connect bun run dev:agent`
}

watch(canManage, (allowed) => {
  if (allowed) loadAgents()
})

onMounted(() => {
  if (canManage.value) {
    loadAgents()
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
    <n-alert v-if="tokenMessage" type="success" class="page-alert">
      <code>{{ tokenMessage }}</code>
    </n-alert>

    <n-card v-if="canManage" :bordered="false">
      <n-space justify="end" class="table-toolbar">
        <n-button type="primary" @click="newAgent">{{ t('admin.agents.new') }}</n-button>
      </n-space>
      <n-data-table
        :columns="columns"
        :data="agents"
        :bordered="false"
        :loading="loading"
        class="admin-table"
      />
    </n-card>

    <n-modal
      v-model:show="showConfigModal"
      preset="card"
      :title="t('admin.agents.config')"
      class="form-modal"
    >
      <n-form :model="form" label-placement="top">
        <n-form-item :label="t('admin.key')">
          <n-input v-model:value="form.key" placeholder="local-agent" />
        </n-form-item>
        <n-form-item :label="t('admin.name')">
          <n-input v-model:value="form.name" placeholder="Local Agent" />
        </n-form-item>
        <n-form-item :label="t('admin.agents.labels')">
          <n-dynamic-tags v-model:value="form.labels" />
        </n-form-item>
        <n-form-item :label="t('admin.agents.token')">
          <n-input
            v-model:value="form.token"
            type="password"
            show-password-on="click"
            :placeholder="t('admin.agents.tokenPlaceholder')"
          />
        </n-form-item>
        <div class="form-grid two">
          <n-form-item :label="t('admin.agents.concurrency')">
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
              @click="saveAgent"
            >
              {{ t('admin.save') }}
            </n-button>
          </n-space>
        </n-space>
      </n-form>
    </n-modal>
  </main>
</template>
