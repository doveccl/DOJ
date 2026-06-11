<script setup lang="ts">
import { NButton, NSpace, NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { apiFetch, DEFAULT_PAGE_SIZE, getItems, PAGE_SIZE_OPTIONS, type Paged } from '../../api'
import { useAuthStore } from '../../stores/auth'

interface UserRow {
  id: number
  name: string
  email: string
  introduction: string
  admin: boolean
  disabled: boolean
  mustChangePassword: boolean
  createdAt: string
  updatedAt: string
}

const auth = useAuthStore()
const { t } = useI18n()
const canManage = computed(() => auth.user?.admin ?? false)
const loading = ref(true)
const savingUserId = ref<number | null>(null)
const saving = ref(false)
const error = ref('')
const generatedPassword = ref('')
const showUserModal = ref(false)
const editingUserId = ref<number | null>(null)
const users = ref<UserRow[]>([])
const userForm = reactive({
  name: '',
  email: '',
  password: '',
  introduction: '',
  admin: false,
  disabled: false,
  mustChangePassword: false
})
const pagination = reactive({
  page: 1,
  pageSize: DEFAULT_PAGE_SIZE,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [...PAGE_SIZE_OPTIONS]
})

const columns = computed<DataTableColumns<UserRow>>(() => [
  { title: t('common.id'), key: 'id', width: 64 },
  { title: t('admin.name'), key: 'name', width: 150, ellipsis: { tooltip: true } },
  { title: t('app.email'), key: 'email', minWidth: 220, ellipsis: { tooltip: true } },
  {
    title: t('admin.users.admin'),
    key: 'admin',
    width: 88,
    render(row) {
      return row.admin
        ? h(NTag, { bordered: false, type: 'success', size: 'small' }, () => t('submissions.yes'))
        : h(NTag, { bordered: false, size: 'small' }, () => t('submissions.no'))
    }
  },
  {
    title: t('admin.status'),
    key: 'disabled',
    width: 96,
    render(row) {
      return h(NTag, { bordered: false, type: row.disabled ? 'error' : 'success' }, () =>
        row.disabled ? t('admin.disabled') : t('admin.active')
      )
    }
  },
  {
    title: t('admin.users.joined'),
    key: 'createdAt',
    width: 150,
    render(row) {
      return new Date(row.createdAt).toLocaleString()
    }
  },
  {
    title: t('admin.actions'),
    key: 'action',
    width: 230,
    render(row) {
      const disabled = savingUserId.value === row.id || row.id === auth.user?.id
      return h(NSpace, { size: 8 }, () => [
        h(
          NButton,
          {
            size: 'small',
            secondary: true,
            onClick: () => editUser(row)
          },
          () => t('admin.edit')
        ),
        h(
          NButton,
          {
            size: 'small',
            secondary: true,
            type: row.disabled ? 'primary' : 'error',
            disabled,
            loading: savingUserId.value === row.id,
            onClick: () => updateUserStatus(row)
          },
          () => (row.disabled ? t('admin.enable') : t('admin.disable'))
        )
      ])
    }
  }
])

async function loadUsers() {
  loading.value = true
  error.value = ''
  try {
    const data = await apiFetch<Paged<UserRow>>(
      `/api/admin/users?page=${pagination.page}&pageSize=${pagination.pageSize}`
    )
    users.value = getItems(data)
    pagination.itemCount = data.total
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  void loadUsers()
}

function handlePageSizeChange(pageSize: number) {
  pagination.pageSize = pageSize
  pagination.page = 1
  void loadUsers()
}

function newUser() {
  editingUserId.value = null
  generatedPassword.value = ''
  userForm.name = ''
  userForm.email = ''
  userForm.password = ''
  userForm.introduction = ''
  userForm.admin = false
  userForm.disabled = false
  userForm.mustChangePassword = true
  showUserModal.value = true
}

function editUser(row: UserRow) {
  editingUserId.value = row.id
  generatedPassword.value = ''
  userForm.name = row.name
  userForm.email = row.email
  userForm.password = ''
  userForm.introduction = row.introduction
  userForm.admin = row.admin
  userForm.disabled = row.disabled
  userForm.mustChangePassword = row.mustChangePassword
  showUserModal.value = true
}

async function saveUser() {
  saving.value = true
  error.value = ''
  try {
    if (editingUserId.value) {
      await apiFetch<UserRow>(`/api/admin/users/${editingUserId.value}`, {
        method: 'PATCH',
        body: JSON.stringify({
          name: userForm.name,
          email: userForm.email,
          introduction: userForm.introduction,
          admin: userForm.admin,
          disabled: userForm.disabled,
          mustChangePassword: userForm.mustChangePassword
        })
      })
    } else {
      await apiFetch<UserRow>('/api/admin/users', {
        method: 'POST',
        body: JSON.stringify({
          name: userForm.name,
          email: userForm.email,
          password: userForm.password,
          admin: userForm.admin,
          disabled: userForm.disabled
        })
      })
    }
    showUserModal.value = false
    await loadUsers()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

async function resetPassword() {
  if (!editingUserId.value) return
  saving.value = true
  error.value = ''
  generatedPassword.value = ''
  try {
    const result = await apiFetch<{ password: string }>(
      `/api/admin/users/${editingUserId.value}/reset-password`,
      {
        method: 'POST',
        body: JSON.stringify(userForm.password ? { password: userForm.password } : {})
      }
    )
    generatedPassword.value = result.password
    userForm.password = ''
    await loadUsers()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

async function updateUserStatus(row: UserRow) {
  savingUserId.value = row.id
  error.value = ''
  try {
    const updated = await apiFetch<UserRow>(`/api/admin/users/${row.id}`, {
      method: 'PATCH',
      body: JSON.stringify({ disabled: !row.disabled })
    })
    users.value = users.value.map((user) => (user.id === updated.id ? updated : user))
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    savingUserId.value = null
  }
}

watch(canManage, (allowed) => {
  if (allowed) loadUsers()
})

onMounted(() => {
  if (canManage.value) {
    loadUsers()
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

    <n-card v-if="canManage" :bordered="false">
      <n-space vertical>
        <n-space justify="end" class="table-toolbar">
          <n-button type="primary" @click="newUser">{{ t('admin.users.create') }}</n-button>
        </n-space>
        <n-data-table
          remote
          :columns="columns"
          :data="users"
          :bordered="false"
          :loading="loading"
          :pagination="pagination"
          :scroll-x="1080"
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        />
      </n-space>
    </n-card>

    <n-modal
      v-model:show="showUserModal"
      preset="card"
      :title="editingUserId ? t('admin.users.edit') : t('admin.users.create')"
      class="form-modal"
    >
      <n-form :model="userForm" label-placement="top">
        <div class="form-grid two">
          <n-form-item :label="t('app.userName')">
            <n-input v-model:value="userForm.name" />
          </n-form-item>
          <n-form-item :label="t('app.email')">
            <n-input v-model:value="userForm.email" />
          </n-form-item>
        </div>
        <n-form-item :label="editingUserId ? t('admin.users.newPassword') : t('app.password')">
          <n-input
            v-model:value="userForm.password"
            type="password"
            show-password-on="click"
            :placeholder="editingUserId ? t('profile.newPasswordPlaceholder') : ''"
          />
        </n-form-item>
        <n-form-item v-if="editingUserId" :label="t('profile.introduction')">
          <n-input v-model:value="userForm.introduction" type="textarea" />
        </n-form-item>
        <div class="form-grid three">
          <n-form-item :label="t('admin.users.admin')">
            <n-switch v-model:value="userForm.admin" />
          </n-form-item>
          <n-form-item :label="t('admin.disabled')">
            <n-switch v-model:value="userForm.disabled" />
          </n-form-item>
          <n-form-item v-if="editingUserId" :label="t('admin.users.mustChangePassword')">
            <n-switch v-model:value="userForm.mustChangePassword" />
          </n-form-item>
        </div>
        <n-alert v-if="generatedPassword" type="success" class="page-alert">
          {{ t('admin.users.generatedPassword') }}: {{ generatedPassword }}
        </n-alert>
        <n-space justify="space-between" class="form-actions">
          <n-button v-if="editingUserId" secondary :loading="saving" @click="resetPassword">
            {{ t('admin.users.resetPassword') }}
          </n-button>
          <span v-else />
          <n-space>
            <n-button @click="showUserModal = false">{{ t('admin.cancel') }}</n-button>
            <n-button
              type="primary"
              :loading="saving"
              :disabled="!userForm.name || !userForm.email || (!editingUserId && !userForm.password)"
              @click="saveUser"
            >
              {{ t('admin.save') }}
            </n-button>
          </n-space>
        </n-space>
      </n-form>
    </n-modal>
  </div>
</template>
