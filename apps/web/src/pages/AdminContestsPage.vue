<script setup lang="ts">
import {
  NAlert,
  NButton,
  NCard,
  NDataTable,
  NDatePicker,
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
import { apiFetch } from '../api'
import { useAuthStore } from '../stores/auth'

interface ContestRow {
  id: number
  title: string
  type: 'OI' | 'ICPC'
  startAt: string
  endAt: string
  freezeAt: string | null
}

interface ProblemRow {
  id: number
  title: string
}

const auth = useAuthStore()
const canManage = computed(() => auth.user?.groups.includes('admin') ?? false)
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const showCreateModal = ref(false)
const contests = ref<ContestRow[]>([])
const problemOptions = ref<SelectOption[]>([])
const form = reactive({
  title: '',
  description: '',
  type: 'OI' as 'OI' | 'ICPC',
  startAt: Date.now(),
  endAt: Date.now() + 2 * 60 * 60 * 1000,
  freezeAt: null as number | null,
  problemIds: [] as number[]
})

const columns: DataTableColumns<ContestRow> = [
  { title: 'Title', key: 'title' },
  {
    title: 'Type',
    key: 'type',
    width: 100,
    render(row) {
      return h(NTag, { bordered: false }, () => row.type)
    }
  },
  {
    title: 'Start',
    key: 'startAt',
    render(row) {
      return new Date(row.startAt).toLocaleString()
    }
  },
  {
    title: 'End',
    key: 'endAt',
    render(row) {
      return new Date(row.endAt).toLocaleString()
    }
  },
  {
    title: 'Freeze',
    key: 'freezeAt',
    render(row) {
      return row.freezeAt ? new Date(row.freezeAt).toLocaleString() : '-'
    }
  }
]

async function loadData() {
  loading.value = true
  error.value = ''
  try {
    const [contestData, problemData] = await Promise.all([
      apiFetch<{ list: ContestRow[] }>('/api/contests'),
      apiFetch<{ list: ProblemRow[] }>('/api/problems')
    ])
    contests.value = contestData.list
    problemOptions.value = problemData.list.map((problem) => ({
      label: `${problem.id}. ${problem.title}`,
      value: problem.id
    }))
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

async function createContest() {
  saving.value = true
  error.value = ''
  try {
    await apiFetch('/api/contests', {
      method: 'POST',
      body: JSON.stringify({
        title: form.title,
        description: form.description,
        type: form.type,
        startAt: new Date(form.startAt).toISOString(),
        endAt: new Date(form.endAt).toISOString(),
        freezeAt: form.freezeAt ? new Date(form.freezeAt).toISOString() : undefined,
        problems: form.problemIds.map((problemId, index) => ({
          problemId,
          key: String.fromCharCode(65 + index),
          score: 100
        }))
      })
    })
    form.title = ''
    form.description = ''
    form.freezeAt = null
    form.problemIds = []
    showCreateModal.value = false
    await loadData()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

watch(canManage, (allowed) => {
  if (allowed) loadData()
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
  <main class="page">
    <section class="page-header">
      <h1>Contests</h1>
      <p>Create timed problem sets with contest-only submissions.</p>
    </section>

    <n-alert v-if="!canManage" type="warning" class="page-alert">
      Admin group is required.
    </n-alert>
    <n-alert v-if="error" type="error" class="page-alert">
      {{ error }}
    </n-alert>

    <n-card v-if="canManage" :bordered="false">
      <n-space justify="end" class="table-toolbar">
        <n-button type="primary" @click="showCreateModal = true">Create contest</n-button>
      </n-space>
      <n-data-table
        :columns="columns"
        :data="contests"
        :bordered="false"
        :loading="loading"
        class="admin-table"
      />
    </n-card>

    <n-modal v-model:show="showCreateModal" preset="card" title="Create contest" class="form-modal">
      <n-form :model="form" label-placement="top">
        <n-form-item label="Title">
          <n-input v-model:value="form.title" placeholder="Weekly Contest" />
        </n-form-item>
        <n-form-item label="Type">
          <n-select
            v-model:value="form.type"
            :options="[
              { label: 'OI', value: 'OI' },
              { label: 'ICPC', value: 'ICPC' }
            ]"
          />
        </n-form-item>
        <n-form-item label="Problems">
          <n-select
            v-model:value="form.problemIds"
            multiple
            filterable
            :options="problemOptions"
            placeholder="Select problems"
          />
        </n-form-item>
        <div class="form-grid">
          <n-form-item label="Start at">
            <n-date-picker v-model:value="form.startAt" type="datetime" class="full-width" />
          </n-form-item>
          <n-form-item label="End at">
            <n-date-picker v-model:value="form.endAt" type="datetime" class="full-width" />
          </n-form-item>
        </div>
        <n-form-item label="Freeze at">
          <n-date-picker
            v-model:value="form.freezeAt"
            type="datetime"
            clearable
            class="full-width"
          />
        </n-form-item>
        <n-form-item label="Description">
          <n-input
            v-model:value="form.description"
            type="textarea"
            placeholder="Optional notes"
            :autosize="{ minRows: 3, maxRows: 5 }"
          />
        </n-form-item>
        <n-space justify="end" class="form-actions">
          <n-button @click="showCreateModal = false">Cancel</n-button>
          <n-button
            type="primary"
            :loading="saving"
            :disabled="!form.title || !form.problemIds.length || form.endAt <= form.startAt"
            @click="createContest"
          >
            Create
          </n-button>
        </n-space>
      </n-form>
    </n-modal>
  </main>
</template>
