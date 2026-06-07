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
  NModal,
  NSelect,
  NSpace,
  NTag
} from 'naive-ui'
import type { DataTableColumns, SelectOption } from 'naive-ui'
import { computed, h, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../../api'
import { useAuthStore } from '../../stores/auth'

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
const { t } = useI18n()
const canManage = computed(() => auth.user?.groups.includes('admin') ?? false)
const loading = ref(true)
const memberLoading = ref(false)
const saving = ref(false)
const addingMember = ref(false)
const error = ref('')
const showCreateModal = ref(false)
const showMemberModal = ref(false)
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

const columns = computed<DataTableColumns<GroupRow>>(() => [
  { title: t('admin.key'), key: 'key' },
  { title: t('admin.name'), key: 'name' },
  {
    title: t('admin.description'),
    key: 'description',
    ellipsis: {
      tooltip: true
    }
  },
  {
    title: t('admin.groups.type'),
    key: 'builtin',
    render(row) {
      return h(NTag, { bordered: false, type: row.builtin ? 'info' : 'default' }, () =>
        row.builtin ? t('admin.groups.builtin') : t('admin.groups.custom')
      )
    }
  }
])

const memberColumns = computed<DataTableColumns<MemberRow>>(() => [
  { title: t('admin.name'), key: 'name' },
  { title: t('app.email'), key: 'email' },
  {
    title: t('admin.groups.role'),
    key: 'manager',
    render(row) {
      return h(NTag, { bordered: false, type: row.manager ? 'success' : 'default' }, () =>
        row.manager ? t('admin.groups.manager') : t('admin.groups.member')
      )
    }
  }
])

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
    showCreateModal.value = false
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
    showMemberModal.value = false
    await loadMembers()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    addingMember.value = false
  }
}

watch(canManage, (allowed) => {
  if (allowed) loadData()
})

watch(selectedGroupId, () => {
  if (canManage.value) loadMembers()
})

onMounted(() => {
  if (canManage.value) {
    loadData()
  } else {
    loading.value = false
  }
})
</script>

<template>
  <div>
    <n-alert v-if="!canManage" type="warning" class="page-alert">
      {{ t('admin.requireAdmin') }}
    </n-alert>

    <n-alert v-if="error" type="error" class="page-alert">
      {{ error }}
    </n-alert>

    <section v-if="canManage" class="admin-stack">
      <n-card :bordered="false">
        <n-space justify="space-between" align="center" class="table-toolbar">
          <n-select
            v-model:value="selectedGroupId"
            :options="groupOptions"
            filterable
            class="toolbar-select"
          />
          <n-button type="primary" @click="showCreateModal = true">
            {{ t('admin.groups.create') }}
          </n-button>
        </n-space>
        <n-data-table
          :columns="columns"
          :data="groups"
          :bordered="false"
          :loading="loading"
          class="admin-table"
        />
      </n-card>

      <n-card :title="t('admin.groups.members')" :bordered="false">
        <n-space justify="end" class="table-toolbar">
          <n-button type="primary" :disabled="!selectedGroupId" @click="showMemberModal = true">
            {{ t('admin.groups.addMember') }}
          </n-button>
        </n-space>
        <n-data-table
          :columns="memberColumns"
          :data="members"
          :bordered="false"
          :loading="memberLoading"
          class="admin-table"
        />
      </n-card>
    </section>

    <n-modal
      v-model:show="showCreateModal"
      preset="card"
      :title="t('admin.groups.create')"
      class="form-modal"
    >
      <n-form :model="form" label-placement="top">
        <n-form-item :label="t('admin.key')">
          <n-input v-model:value="form.key" placeholder="team-alpha" />
        </n-form-item>
        <n-form-item :label="t('admin.name')">
          <n-input v-model:value="form.name" placeholder="Team Alpha" />
        </n-form-item>
        <n-form-item :label="t('admin.description')">
          <n-input
            v-model:value="form.description"
            type="textarea"
            :placeholder="t('admin.optionalNotes')"
            :autosize="{ minRows: 3, maxRows: 5 }"
          />
        </n-form-item>
        <n-space justify="end" class="form-actions">
          <n-button @click="showCreateModal = false">{{ t('admin.cancel') }}</n-button>
          <n-button
            type="primary"
            :loading="saving"
            :disabled="!form.key || !form.name"
            @click="createGroup"
          >
            {{ t('admin.create') }}
          </n-button>
        </n-space>
      </n-form>
    </n-modal>

    <n-modal
      v-model:show="showMemberModal"
      preset="card"
      :title="t('admin.groups.addMember')"
      class="form-modal narrow"
    >
      <n-form :model="memberForm" label-placement="top">
        <n-form-item :label="t('admin.groups.group')">
          <n-select v-model:value="selectedGroupId" :options="groupOptions" filterable />
        </n-form-item>
        <n-form-item :label="t('common.user')">
          <n-select v-model:value="memberForm.userId" :options="userOptions" filterable />
        </n-form-item>
        <n-checkbox v-model:checked="memberForm.manager">
          {{ t('admin.groups.groupManager') }}
        </n-checkbox>
        <n-space justify="end" class="form-actions">
          <n-button @click="showMemberModal = false">{{ t('admin.cancel') }}</n-button>
          <n-button
            type="primary"
            :loading="addingMember"
            :disabled="!selectedGroupId || !memberForm.userId"
            @click="addMember"
          >
            {{ t('admin.groups.addMember') }}
          </n-button>
        </n-space>
      </n-form>
    </n-modal>
  </div>
</template>
