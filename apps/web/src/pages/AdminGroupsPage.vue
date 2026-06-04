<script setup lang="ts">
import {
  NAlert,
  NButton,
  NCard,
  NDataTable,
  NForm,
  NFormItem,
  NInput,
  NSpace,
  NTag
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { h, onMounted, reactive, ref, watch } from 'vue'
import { apiFetch } from '../api'
import { useAuthStore } from '../stores/auth'

interface GroupRow {
  id: string
  key: string
  name: string
  description: string
  builtin: boolean
}

const auth = useAuthStore()
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const groups = ref<GroupRow[]>([])
const form = reactive({
  key: '',
  name: '',
  description: ''
})

const columns: DataTableColumns<GroupRow> = [
  { title: 'Key', key: 'key' },
  { title: 'Name', key: 'name' },
  {
    title: 'Description',
    key: 'description',
    ellipsis: {
      tooltip: true
    }
  },
  {
    title: 'Type',
    key: 'builtin',
    render(row) {
      return h(NTag, { bordered: false, type: row.builtin ? 'info' : 'default' }, () =>
        row.builtin ? 'builtin' : 'custom'
      )
    }
  }
]

async function loadGroups() {
  loading.value = true
  error.value = ''
  try {
    const data = await apiFetch<{ list: GroupRow[] }>('/api/groups')
    groups.value = data.list
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

async function createGroup() {
  saving.value = true
  error.value = ''
  try {
    await apiFetch<GroupRow>('/api/groups', {
      method: 'POST',
      body: JSON.stringify(form)
    })
    form.key = ''
    form.name = ''
    form.description = ''
    await loadGroups()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

watch(
  () => auth.signedIn,
  (signedIn) => {
    if (signedIn) loadGroups()
  }
)

onMounted(() => {
  if (auth.signedIn) {
    loadGroups()
  } else {
    loading.value = false
  }
})
</script>

<template>
  <main class="page">
    <section class="page-header">
      <h1>Groups</h1>
      <p>Manage coarse-grained access groups for the system.</p>
    </section>

    <n-alert v-if="!auth.user?.groups.includes('admin')" type="warning" class="page-alert">
      Admin group is required.
    </n-alert>

    <n-alert v-if="error" type="error" class="page-alert">
      {{ error }}
    </n-alert>

    <section class="admin-layout">
      <n-card title="Create group" :bordered="false">
        <n-form :model="form" label-placement="top">
          <n-form-item label="Key">
            <n-input v-model:value="form.key" placeholder="team-alpha" />
          </n-form-item>
          <n-form-item label="Name">
            <n-input v-model:value="form.name" placeholder="Team Alpha" />
          </n-form-item>
          <n-form-item label="Description">
            <n-input
              v-model:value="form.description"
              type="textarea"
              placeholder="Optional notes"
              :autosize="{ minRows: 3, maxRows: 5 }"
            />
          </n-form-item>
          <n-space justify="end">
            <n-button type="primary" :loading="saving" :disabled="!form.key || !form.name" @click="createGroup">
              Create
            </n-button>
          </n-space>
        </n-form>
      </n-card>

      <n-data-table
        :columns="columns"
        :data="groups"
        :bordered="false"
        :loading="loading"
        class="admin-table"
      />
    </section>
  </main>
</template>
