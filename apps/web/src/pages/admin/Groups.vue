<script setup lang="ts">
import { NButton, NSpace } from 'naive-ui'
import type { DataTableColumns, SelectOption } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { apiFetch, DEFAULT_PAGE_SIZE, getItems, PAGE_SIZE_OPTIONS, type Paged } from '../../api'
import { useAuthStore } from '../../stores/auth'

interface GroupRow {
  id: number
  name: string
  createdAt: string
  updatedAt: string
}

interface UserRow {
  id: number
  name: string
  email: string
}

interface MemberRow extends UserRow {
  createdAt: string
}

const auth = useAuthStore()
const { t } = useI18n()
const canManage = computed(() => auth.user?.admin ?? false)
const loading = ref(true)
const memberLoading = ref(false)
const userSearchLoading = ref(false)
const saving = ref(false)
const addingMember = ref(false)
const error = ref('')
const showCreateModal = ref(false)
const showMemberModal = ref(false)
const editingGroupId = ref<number | null>(null)
const groups = ref<GroupRow[]>([])
const users = ref<UserRow[]>([])
const members = ref<MemberRow[]>([])
const selectedGroupId = ref<number | null>(null)
const form = reactive({
  name: ''
})
const memberForm = reactive({
  userId: null as number | null
})
const groupPagination = reactive({
  page: 1,
  pageSize: DEFAULT_PAGE_SIZE,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [...PAGE_SIZE_OPTIONS]
})

const columns = computed<DataTableColumns<GroupRow>>(() => [
  { title: t('common.id'), key: 'id', width: 80 },
  { title: t('admin.name'), key: 'name', minWidth: 180 },
  {
    title: t('admin.users.updatedAt'),
    key: 'updatedAt',
    width: 180,
    render(row) {
      return new Date(row.updatedAt).toLocaleString()
    }
  },
  {
    title: t('admin.actions'),
    key: 'actions',
    width: 170,
    render(row) {
      return h(NSpace, { size: 8 }, () => [
        h(
          NButton,
          { size: 'small', secondary: true, onClick: () => editGroup(row) },
          () => t('admin.edit')
        ),
        h(
          NButton,
          { size: 'small', tertiary: true, type: 'error', onClick: () => deleteGroup(row.id) },
          () => t('admin.delete')
        )
      ])
    }
  }
])

const memberColumns = computed<DataTableColumns<MemberRow>>(() => [
  { title: t('admin.name'), key: 'name' },
  { title: t('app.email'), key: 'email' },
  {
    title: t('admin.users.joined'),
    key: 'createdAt',
    width: 180,
    render(row) {
      return new Date(row.createdAt).toLocaleString()
    }
  },
  {
    title: t('admin.actions'),
    key: 'actions',
    width: 100,
    render(row) {
      return h(
        NButton,
        { size: 'small', tertiary: true, type: 'error', onClick: () => removeMember(row.id) },
        () => t('admin.delete')
      )
    }
  }
])

const groupOptions = computed<SelectOption[]>(() =>
  groups.value.map((group) => ({
    label: `${group.name} (#${group.id})`,
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
    const groupData = await apiFetch<Paged<GroupRow>>(
      `/api/admin/groups?page=${groupPagination.page}&pageSize=${groupPagination.pageSize}`
    )
    groups.value = getItems(groupData)
    groupPagination.itemCount = groupData.total
    if (!selectedGroupId.value && groups.value.length) {
      selectedGroupId.value = groups.value[0].id
    }
    if (selectedGroupId.value) await loadMembers()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

function handleGroupPageChange(page: number) {
  groupPagination.page = page
  void loadData()
}

function handleGroupPageSizeChange(pageSize: number) {
  groupPagination.pageSize = pageSize
  groupPagination.page = 1
  void loadData()
}

async function searchUsers(query = '') {
  userSearchLoading.value = true
  error.value = ''
  try {
    const data = await apiFetch<Paged<UserRow>>(
      `/api/admin/users?page=1&pageSize=20&q=${encodeURIComponent(query)}`
    )
    users.value = getItems(data)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    userSearchLoading.value = false
  }
}

function newGroup() {
  editingGroupId.value = null
  form.name = ''
  showCreateModal.value = true
}

function editGroup(group: GroupRow) {
  editingGroupId.value = group.id
  form.name = group.name
  showCreateModal.value = true
}

async function saveGroup() {
  saving.value = true
  error.value = ''
  try {
    const group = await apiFetch<GroupRow>(
      editingGroupId.value ? `/api/admin/groups/${editingGroupId.value}` : '/api/admin/groups',
      {
        method: editingGroupId.value ? 'PATCH' : 'POST',
        body: JSON.stringify({ name: form.name })
      }
    )
    selectedGroupId.value = group.id
    showCreateModal.value = false
    await loadData()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

async function deleteGroup(id: number) {
  error.value = ''
  try {
    await apiFetch(`/api/admin/groups/${id}`, { method: 'DELETE' })
    if (selectedGroupId.value === id) selectedGroupId.value = null
    await loadData()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  }
}

async function loadMembers() {
  if (!selectedGroupId.value) return

  memberLoading.value = true
  error.value = ''
  try {
    const data = await apiFetch<Paged<MemberRow>>(
      `/api/admin/groups/${selectedGroupId.value}/users`
    )
    members.value = getItems(data)
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
    await apiFetch(`/api/admin/groups/${selectedGroupId.value}/users`, {
      method: 'POST',
      body: JSON.stringify({ userId: memberForm.userId })
    })
    memberForm.userId = null
    showMemberModal.value = false
    await loadMembers()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    addingMember.value = false
  }
}

function openMemberModal() {
  memberForm.userId = null
  showMemberModal.value = true
  void searchUsers()
}

async function removeMember(userId: number) {
  if (!selectedGroupId.value) return
  error.value = ''
  try {
    await apiFetch(`/api/admin/groups/${selectedGroupId.value}/users/${userId}`, {
      method: 'DELETE'
    })
    await loadMembers()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
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
          <n-button type="primary" @click="newGroup">
            {{ t('admin.groups.create') }}
          </n-button>
        </n-space>
        <n-data-table
          remote
          :columns="columns"
          :data="groups"
          :bordered="false"
          :loading="loading"
          :pagination="groupPagination"
          :scroll-x="620"
          class="admin-table"
          @update:page="handleGroupPageChange"
          @update:page-size="handleGroupPageSizeChange"
        />
      </n-card>

      <n-card :title="t('admin.groups.members')" :bordered="false">
        <n-space justify="end" class="table-toolbar">
          <n-button type="primary" :disabled="!selectedGroupId" @click="openMemberModal">
            {{ t('admin.groups.addMember') }}
          </n-button>
        </n-space>
        <n-data-table
          :columns="memberColumns"
          :data="members"
          :bordered="false"
          :loading="memberLoading"
          :scroll-x="560"
          class="admin-table"
        />
      </n-card>
    </section>

    <n-modal
      v-model:show="showCreateModal"
      preset="card"
      :title="editingGroupId ? t('admin.edit') : t('admin.groups.create')"
      class="form-modal"
    >
      <n-form :model="form" label-placement="top">
        <n-form-item :label="t('admin.name')">
          <n-input v-model:value="form.name" placeholder="Class A" />
        </n-form-item>
        <n-space justify="end" class="form-actions">
          <n-button @click="showCreateModal = false">{{ t('admin.cancel') }}</n-button>
          <n-button
            type="primary"
            :loading="saving"
            :disabled="!form.name"
            @click="saveGroup"
          >
            {{ editingGroupId ? t('admin.save') : t('admin.create') }}
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
          <n-select
            v-model:value="memberForm.userId"
            :options="userOptions"
            :loading="userSearchLoading"
            filterable
            remote
            clearable
            @search="searchUsers"
          />
        </n-form-item>
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
