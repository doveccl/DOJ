<script setup lang="ts">
import {
  NAlert,
  NButton,
  NCard,
  NDataTable,
  NDynamicTags,
  NForm,
  NFormItem,
  NInput,
  NModal,
  NSelect,
  NSpace,
  NTag,
  NTooltip
} from 'naive-ui'
import type { DataTableColumns, SelectOption } from 'naive-ui'
import { computed, h, onMounted, reactive, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../api'
import MarkdownEditor from '../components/MarkdownEditor.vue'
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
const linkType = ref<'none' | 'problem' | 'contest'>('none')
const linkId = ref<number | null>(null)
const problemOptions = ref<SelectOption[]>([])
const contestOptions = ref<SelectOption[]>([])
const form = reactive({
  title: '',
  contentMarkdown: '',
  tags: [] as string[]
})

const linkTypeOptions = computed(() => [
  { label: t('discussion.linkNone'), value: 'none' },
  { label: t('discussion.linkProblem'), value: 'problem' },
  { label: t('discussion.linkContest'), value: 'contest' }
])

const linkOptions = computed(() =>
  linkType.value === 'problem'
    ? problemOptions.value
    : linkType.value === 'contest'
      ? contestOptions.value
      : []
)

const columns = computed<DataTableColumns<TopicRow>>(() => [
  {
    title: t('discussion.topic'),
    key: 'title',
    render(row) {
      return h(RouterLink, { to: `/discussion/${row.id}`, class: 'table-link' }, () => row.title)
    }
  },
  { title: t('discussion.author'), key: 'userName', width: 140 },
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
    title: t('discussion.updated'),
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
    const data = await apiFetch<{ list: TopicRow[] }>('/api/discussion/topics')
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
    const detail = await apiFetch<{ topic: { id: number } }>('/api/discussion/topics', {
      method: 'POST',
      body: JSON.stringify({
        title: form.title,
        contentMarkdown: form.contentMarkdown,
        tags: form.tags,
        linkedProblemId: linkType.value === 'problem' ? (linkId.value ?? undefined) : undefined,
        linkedContestId: linkType.value === 'contest' ? (linkId.value ?? undefined) : undefined
      })
    })
    showCreateModal.value = false
    await router.push(`/discussion/${detail.topic.id}`)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

async function loadLinkOptions() {
  try {
    const [problems, contests] = await Promise.all([
      apiFetch<{ list: Array<{ id: number; title: string }> }>('/api/problems'),
      apiFetch<{ list: Array<{ id: number; title: string }> }>('/api/contests')
    ])
    problemOptions.value = problems.list.map((p) => ({ label: `P${p.id} ${p.title}`, value: p.id }))
    contestOptions.value = contests.list.map((c) => ({ label: c.title, value: c.id }))
  } catch {
    problemOptions.value = []
    contestOptions.value = []
  }
}

onMounted(() => {
  void loadTopics()
  if (auth.signedIn) void loadLinkOptions()
})
</script>

<template>
  <main class="page">
    <n-alert v-if="error" type="error" class="page-alert">
      {{ error }}
    </n-alert>

    <n-card :bordered="false">
      <n-space justify="end" class="table-toolbar">
        <n-button v-if="auth.signedIn" type="primary" @click="showCreateModal = true">
          {{ t('discussion.newTopic') }}
        </n-button>
        <n-tooltip v-else trigger="hover">
          <template #trigger>
            <span class="tooltip-button-wrap">
              <n-button type="primary" disabled>
                {{ t('discussion.newTopic') }}
              </n-button>
            </span>
          </template>
          {{ t('discussion.signInTopic') }}
        </n-tooltip>
      </n-space>
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
      :title="t('discussion.newTopic')"
      class="form-modal"
    >
      <n-form :model="form" label-placement="top">
        <n-form-item :label="t('common.title')">
          <n-input v-model:value="form.title" />
        </n-form-item>
        <n-form-item :label="t('discussion.content')">
          <markdown-editor v-model="form.contentMarkdown" />
        </n-form-item>
        <n-form-item :label="t('common.tags')">
          <n-dynamic-tags v-model:value="form.tags" />
        </n-form-item>
        <div class="form-grid two">
          <n-form-item :label="t('discussion.link')">
            <n-select
              v-model:value="linkType"
              :options="linkTypeOptions"
              @update:value="linkId = null"
            />
          </n-form-item>
          <n-form-item
            v-if="linkType !== 'none'"
            :label="t(`discussion.link${linkType === 'problem' ? 'Problem' : 'Contest'}`)"
          >
            <n-select
              v-model:value="linkId"
              :options="linkOptions"
              filterable
              clearable
              :placeholder="t('discussion.linkPlaceholder')"
            />
          </n-form-item>
        </div>
        <n-space justify="end" class="form-actions">
          <n-button @click="showCreateModal = false">{{ t('admin.cancel') }}</n-button>
          <n-button
            type="primary"
            :loading="saving"
            :disabled="!form.title || !form.contentMarkdown"
            @click="createTopic"
          >
            {{ t('discussion.publish') }}
          </n-button>
        </n-space>
      </n-form>
    </n-modal>
  </main>
</template>
