<script setup lang="ts">
import { NAlert, NButton, NCard, NDataTable, NSpace, NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { computed, h, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../../api'
import { useAuthStore } from '../../stores/auth'

interface UserRow {
  id: number
  name: string
  email: string
  solvedCount: number
  submissionCount: number
  disabledAt: string | null
  createdAt: string
}

const auth = useAuthStore()
const { t } = useI18n()
const canManage = computed(() => auth.user?.groups.includes('admin') ?? false)
const loading = ref(true)
const savingUserId = ref<number | null>(null)
const error = ref('')
const users = ref<UserRow[]>([])
const pagination = reactive({
  page: 1,
  pageSize: 50,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [20, 50, 100]
})

const columns = computed<DataTableColumns<UserRow>>(() => [
  { title: t('common.id'), key: 'id', width: 64 },
  { title: t('admin.name'), key: 'name', width: 150, ellipsis: { tooltip: true } },
  { title: t('app.email'), key: 'email', minWidth: 220, ellipsis: { tooltip: true } },
  { title: t('admin.users.solved'), key: 'solvedCount', width: 82 },
  { title: t('admin.users.submissions'), key: 'submissionCount', width: 96 },
  {
    title: t('admin.status'),
    key: 'disabledAt',
    width: 96,
    render(row) {
      return h(NTag, { bordered: false, type: row.disabledAt ? 'error' : 'success' }, () =>
        row.disabledAt ? t('admin.disabled') : t('admin.active')
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
    width: 90,
    render(row) {
      const disabled = savingUserId.value === row.id || row.id === auth.user?.id
      return h(
        NButton,
        {
          size: 'small',
          secondary: true,
          type: row.disabledAt ? 'primary' : 'error',
          disabled,
          loading: savingUserId.value === row.id,
          onClick: () => updateUserStatus(row)
        },
        () => (row.disabledAt ? t('admin.enable') : t('admin.disable'))
      )
    }
  }
])

async function loadUsers() {
  loading.value = true
  error.value = ''
  try {
    const data = await apiFetch<{ list: UserRow[]; total: number }>(
      `/api/users?page=${pagination.page}&pageSize=${pagination.pageSize}`
    )
    users.value = data.list
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

async function updateUserStatus(row: UserRow) {
  savingUserId.value = row.id
  error.value = ''
  try {
    const updated = await apiFetch<UserRow>(`/api/users/${row.id}`, {
      method: 'PATCH',
      body: JSON.stringify({ disabled: !row.disabledAt })
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
        <n-data-table
          remote
          :columns="columns"
          :data="users"
          :bordered="false"
          :loading="loading"
          :pagination="pagination"
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        />
      </n-space>
    </n-card>
  </div>
</template>
