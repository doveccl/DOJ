<script setup lang="ts">
import type { DataTableColumns } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { apiFetch, getItems, type Paged } from '../../api'
import { useAuthStore } from '../../stores/auth'

interface AgentRow {
  key: string
  name: string
  concurrency: number
  activeJobs: number
  version: string
  connectedAt: string
  heartbeatAt: string
}

const auth = useAuthStore()
const { t } = useI18n()
const canManage = computed(() => auth.user?.admin ?? false)
const loading = ref(true)
const error = ref('')
const instructions = ref<{ server: string; secretEnv: string; command: string } | null>(null)
const agents = ref<AgentRow[]>([])

const columns = computed<DataTableColumns<AgentRow>>(() => [
  {
    title: t('admin.key'),
    key: 'key',
    width: 160
  },
  { title: t('admin.name'), key: 'name', width: 160 },
  { title: t('admin.agents.concurrency'), key: 'concurrency', width: 116 },
  { title: t('admin.agents.activeJobs'), key: 'activeJobs', width: 116 },
  { title: t('admin.agents.version'), key: 'version', width: 140 },
  {
    title: t('admin.agents.connectedAt'),
    key: 'connectedAt',
    width: 190,
    render(row) {
      return row.connectedAt ? new Date(row.connectedAt).toLocaleString() : t('admin.agents.never')
    }
  },
  {
    title: t('admin.agents.heartbeatAt'),
    key: 'heartbeatAt',
    width: 190,
    render(row) {
      return row.heartbeatAt ? new Date(row.heartbeatAt).toLocaleString() : t('admin.agents.never')
    }
  }
])

async function loadAgents() {
  loading.value = true
  error.value = ''
  try {
    const data = await apiFetch<Paged<AgentRow>>('/api/admin/agents')
    agents.value = getItems(data)
    instructions.value = await apiFetch<{ server: string; secretEnv: string; command: string }>(
      '/api/admin/agents/instructions'
    )
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
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

    <n-card v-if="canManage" :bordered="false" class="stacked-card">
      <n-data-table
        :columns="columns"
        :data="agents"
        :bordered="false"
        :loading="loading"
        :scroll-x="1070"
        class="admin-table"
      >
        <template #empty>
          {{ t('admin.agents.empty') }}
        </template>
      </n-data-table>
    </n-card>

    <n-card v-if="canManage && instructions" :title="t('admin.agents.instructions')" :bordered="false" class="stacked-card">
      <n-space vertical>
        <div class="compact-stack">
          <span>{{ t('admin.agents.server') }}: {{ instructions.server }}</span>
          <span>{{ t('admin.agents.secretEnv') }}: {{ instructions.secretEnv }}</span>
        </div>
        <pre class="code-block"><code>{{ instructions.command }}</code></pre>
      </n-space>
    </n-card>
  </main>
</template>

<style scoped>
.code-block {
  padding: 12px;
  overflow-x: auto;
  white-space: pre-wrap;
  border-radius: var(--radius-md);
  background: color-mix(in srgb, var(--surface-bg) 92%, var(--text-color) 8%);
}
</style>
