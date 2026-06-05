<script setup lang="ts">
import { NAlert, NButton, NCard, NDataTable, NSpace, NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { computed, h, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../api'
import { useAuthStore } from '../stores/auth'

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

const columns = computed<DataTableColumns<UserRow>>(() => [
  { title: t('common.id'), key: 'id', width: 80 },
  { title: t('admin.name'), key: 'name', minWidth: 140 },
  { title: t('app.email'), key: 'email', minWidth: 220 },
  { title: t('admin.users.solved'), key: 'solvedCount', width: 100 },
  { title: t('admin.users.submissions'), key: 'submissionCount', width: 120 },
  {
    title: t('admin.status'),
    key: 'disabledAt',
    width: 120,
    render(row) {
      return h(NTag, { bordered: false, type: row.disabledAt ? 'error' : 'success' }, () =>
        row.disabledAt ? t('admin.disabled') : t('admin.active')
      )
    }
  },
  {
    title: t('admin.users.joined'),
    key: 'createdAt',
    minWidth: 160,
    render(row) {
      return new Date(row.createdAt).toLocaleString()
    }
  },
  {
    title: t('admin.actions'),
    key: 'action',
    width: 120,
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
    const data = await apiFetch<{ list: UserRow[] }>('/api/users')
    users.value = data.list
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
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
  <main class="page">
    <section class="page-header">
      <h1>{{ t('admin.users.title') }}</h1>
      <p>{{ t('admin.users.subtitle') }}</p>
    </section>

    <n-alert v-if="!canManage" type="warning" class="page-alert">
      {{ t('admin.requireAdmin') }}
    </n-alert>
    <n-alert v-if="error" type="error" class="page-alert">
      {{ error }}
    </n-alert>

    <n-card v-if="canManage" :bordered="false">
      <n-space vertical>
        <n-data-table :columns="columns" :data="users" :bordered="false" :loading="loading" />
      </n-space>
    </n-card>
  </main>
</template>
