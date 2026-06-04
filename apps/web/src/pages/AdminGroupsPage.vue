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
  NSelect,
  NSpace,
  NTag
} from 'naive-ui'
import type { DataTableColumns, SelectOption } from 'naive-ui'
import { computed, h, onMounted, reactive, ref, watch } from 'vue'
import { apiFetch } from '../api'
import { useAuthStore } from '../stores/auth'

interface GroupRow {
  id: number
  key: string
  name: string
  description: string
  builtin: boolean
}

interface UserRow {
  id: number
  name: string
  email: string
}

interface MemberRow extends UserRow {
  manager: boolean
}

const auth = useAuthStore()
const loading = ref(true)
const memberLoading = ref(false)
const saving = ref(false)
const addingMember = ref(false)
const error = ref('')
const groups = ref<GroupRow[]>([])
const users = ref<UserRow[]>([])
const members = ref<MemberRow[]>([])
const selectedGroupId = ref<number | null>(null)
const form = reactive({
  key: '',
  name: '',
  description: ''
})
const memberForm = reactive({
  userId: null as number | null,
  manager: false
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

const memberColumns: DataTableColumns<MemberRow> = [
  { title: 'Name', key: 'name' },
  { title: 'Email', key: 'email' },
  {
    title: 'Role',
    key: 'manager',
    render(row) {
      return h(NTag, { bordered: false, type: row.manager ? 'success' : 'default' }, () =>
        row.manager ? 'manager' : 'member'
      )
    }
  }
]

const groupOptions = computed<SelectOption[]>(() =>
  groups.value.map((group) => ({
    label: `${group.name} (${group.key})`,
    value: group.id
  }))
)

const userOptions = computed<SelectOption[]>(() =>
  users.value.map((user) => ({
    label: `${user.name} (${user.email})`,
    value: user.id
  }))
)

async function loadData() {
  loading.value = true
  error.value = ''
  try {
    const [groupData, userData] = await Promise.all([
      apiFetch<{ list: GroupRow[] }>('/api/groups'),
      apiFetch<{ list: UserRow[] }>('/api/users')
    ])
    groups.value = groupData.list
    users.value = userData.list
    if (!selectedGroupId.value && groupData.list.length) {
      selectedGroupId.value = groupData.list[0].id
    }
    if (selectedGroupId.value) await loadMembers()
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
    const group = await apiFetch<GroupRow>('/api/groups', {
      method: 'POST',
      body: JSON.stringify(form)
    })
    form.key = ''
    form.name = ''
    form.description = ''
    selectedGroupId.value = group.id
    await loadData()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

async function loadMembers() {
  if (!selectedGroupId.value) return

  memberLoading.value = true
  error.value = ''
  try {
    const data = await apiFetch<{ list: MemberRow[] }>(`/api/groups/${selectedGroupId.value}/users`)
    members.value = data.list
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    memberLoading.value = false
  }
}

async function addMember() {
  if (!selectedGroupId.value || !memberForm.userId) return

  addingMember.value = true
  error.value = ''
  try {
    await apiFetch(`/api/groups/${selectedGroupId.value}/users`, {
      method: 'POST',
      body: JSON.stringify(memberForm)
    })
    memberForm.userId = null
    memberForm.manager = false
    await loadMembers()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    addingMember.value = false
  }
}

watch(
  () => auth.signedIn,
  (signedIn) => {
    if (signedIn) loadData()
  }
)

watch(selectedGroupId, () => {
  if (auth.signedIn) loadMembers()
})

onMounted(() => {
  if (auth.signedIn) {
    loadData()
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
      <div class="admin-stack">
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

        <n-card title="Add member" :bordered="false">
          <n-form :model="memberForm" label-placement="top">
            <n-form-item label="Group">
              <n-select v-model:value="selectedGroupId" :options="groupOptions" filterable />
            </n-form-item>
            <n-form-item label="User">
              <n-select v-model:value="memberForm.userId" :options="userOptions" filterable />
            </n-form-item>
            <n-checkbox v-model:checked="memberForm.manager">Group manager</n-checkbox>
            <n-space justify="end" class="form-actions">
              <n-button
                type="primary"
                :loading="addingMember"
              :disabled="!selectedGroupId || !memberForm.userId"
                @click="addMember"
              >
                Add
              </n-button>
            </n-space>
          </n-form>
        </n-card>
      </div>

      <div class="admin-stack">
        <n-data-table
          :columns="columns"
          :data="groups"
          :bordered="false"
          :loading="loading"
          class="admin-table"
        />
        <n-data-table
          :columns="memberColumns"
          :data="members"
          :bordered="false"
          :loading="memberLoading"
          class="admin-table"
        />
      </div>
    </section>
  </main>
</template>
