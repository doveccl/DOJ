<script setup lang="ts">
import {
  NAlert,
  NButton,
  NCard,
  NDataTable,
  NForm,
  NFormItem,
  NInput,
  NModal,
  NSpace,
  NTag
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { computed, h, onMounted, reactive, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../api'
import { useAuthStore } from '../stores/auth'

interface TopicRow {
  id: number
  title: string
  tags: string[]
  userName: string
  linkedProblemId: number | null
  linkedContestId: number | null
  updatedAt: string
}

const auth = useAuthStore()
const router = useRouter()
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const showCreateModal = ref(false)
const topics = ref<TopicRow[]>([])
const { t } = useI18n()
const form = reactive({
  title: '',
  contentMarkdown: '',
  tags: '',
  linkedProblemId: '',
  linkedContestId: ''
})

const columns = computed<DataTableColumns<TopicRow>>(() => [
  {
    title: t('bbs.topic'),
    key: 'title',
    render(row) {
      return h(RouterLink, { to: `/bbs/${row.id}`, class: 'table-link' }, () => row.title)
    }
  },
  { title: t('bbs.author'), key: 'userName', width: 140 },
  {
    title: t('common.tags'),
    key: 'tags',
    render(row) {
      return row.tags.length
        ? row.tags.map((tag) => h(NTag, { bordered: false, style: 'margin-right: 6px' }, () => tag))
        : '-'
    }
  },
  {
    title: t('bbs.updated'),
    key: 'updatedAt',
    width: 190,
    render(row) {
      return new Date(row.updatedAt).toLocaleString()
    }
  }
])

async function loadTopics() {
  loading.value = true
  error.value = ''
  try {
    const data = await apiFetch<{ list: TopicRow[] }>('/api/bbs/topics')
    topics.value = data.list
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

async function createTopic() {
  saving.value = true
  error.value = ''
  try {
    const detail = await apiFetch<{ topic: { id: number } }>('/api/bbs/topics', {
      method: 'POST',
      body: JSON.stringify({
        title: form.title,
        contentMarkdown: form.contentMarkdown,
        tags: form.tags
          .split(',')
          .map((tag) => tag.trim())
          .filter(Boolean),
        linkedProblemId: form.linkedProblemId ? Number(form.linkedProblemId) : undefined,
        linkedContestId: form.linkedContestId ? Number(form.linkedContestId) : undefined
      })
    })
    showCreateModal.value = false
    await router.push(`/bbs/${detail.topic.id}`)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

onMounted(loadTopics)
</script>

<template>
  <main class="page">
    <section class="page-header">
      <h1>{{ t('bbs.title') }}</h1>
    </section>

    <n-alert v-if="error" type="error" class="page-alert">
      {{ error }}
    </n-alert>

    <n-card :bordered="false">
      <n-space justify="end" class="table-toolbar">
        <n-button type="primary" :disabled="!auth.signedIn" @click="showCreateModal = true">
          {{ t('bbs.newTopic') }}
        </n-button>
      </n-space>
      <p v-if="!auth.signedIn" class="muted table-toolbar">{{ t('bbs.signInTopic') }}</p>
      <n-data-table
        :columns="columns"
        :data="topics"
        :bordered="false"
        :loading="loading"
        class="admin-table"
      />
    </n-card>

    <n-modal
      v-model:show="showCreateModal"
      preset="card"
      :title="t('bbs.newTopic')"
      class="form-modal"
    >
      <n-form :model="form" label-placement="top">
        <n-form-item :label="t('common.title')">
          <n-input v-model:value="form.title" />
        </n-form-item>
        <n-form-item :label="t('bbs.content')">
          <n-input
            v-model:value="form.contentMarkdown"
            type="textarea"
            :autosize="{ minRows: 7, maxRows: 14 }"
          />
        </n-form-item>
        <n-form-item :label="t('common.tags')">
          <n-input v-model:value="form.tags" placeholder="problem, contest" />
        </n-form-item>
        <div class="form-grid two">
          <n-form-item :label="`${t('bbs.problem')} ID`">
            <n-input v-model:value="form.linkedProblemId" />
          </n-form-item>
          <n-form-item :label="`${t('bbs.contest')} ID`">
            <n-input v-model:value="form.linkedContestId" />
          </n-form-item>
        </div>
        <n-space justify="end" class="form-actions">
          <n-button @click="showCreateModal = false">Cancel</n-button>
          <n-button
            type="primary"
            :loading="saving"
            :disabled="!form.title || !form.contentMarkdown"
            @click="createTopic"
          >
            {{ t('bbs.publish') }}
          </n-button>
        </n-space>
      </n-form>
    </n-modal>
  </main>
</template>
