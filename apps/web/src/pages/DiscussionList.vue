<script setup lang="ts">
import { NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { apiFetch, DEFAULT_PAGE_SIZE, getItems, PAGE_SIZE_OPTIONS, type Paged } from '../api'
import MarkdownEditor from '../components/MarkdownEditor.vue'
import { useAuthStore } from '../stores/auth'

interface TopicRow {
  id: number
  title: string
  tags: string[]
  pinned: boolean
  author: { id: number; name: string; avatarUrl: string }
  createdAt: string
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
const pagination = reactive({
  page: 1,
  pageSize: DEFAULT_PAGE_SIZE,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [...PAGE_SIZE_OPTIONS]
})
const form = reactive({
  title: '',
  contentMarkdown: '',
  tags: [] as string[]
})

const columns = computed<DataTableColumns<TopicRow>>(() => [
  {
    title: t('discussion.topic'),
    key: 'title',
    render(row) {
      return h(RouterLink, { to: `/discussion/${row.id}`, class: 'table-link' }, () => row.title)
    }
  },
  {
    title: t('discussion.pinned'),
    key: 'pinned',
    width: 96,
    render(row) {
      return row.pinned ? h(NTag, { type: 'success', bordered: false }, () => t('discussion.pinned')) : '-'
    }
  },
  {
    title: t('discussion.author'),
    key: 'author',
    width: 140,
    render(row) {
      return row.author.name
    }
  },
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
    const data = await apiFetch<Paged<TopicRow>>(
      `/api/discussion/topics?page=${pagination.page}&pageSize=${pagination.pageSize}`
    )
    topics.value = getItems(data)
    pagination.itemCount = data.total
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  void loadTopics()
}

function handlePageSizeChange(pageSize: number) {
  pagination.pageSize = pageSize
  pagination.page = 1
  void loadTopics()
}

async function createTopic() {
  saving.value = true
  error.value = ''
  try {
    const detail = await apiFetch<{ id: number }>('/api/discussion/topics', {
      method: 'POST',
      body: JSON.stringify({
        title: form.title,
        content: form.contentMarkdown,
        tags: form.tags
      })
    })
    showCreateModal.value = false
    await router.push(`/discussion/${detail.id}`)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  void loadTopics()
})
</script>

<template>
  <main class="page">
    <n-alert v-if="error" type="error" class="page-alert">
      {{ error }}
    </n-alert>

    <n-card :bordered="false">
      <n-space v-if="auth.signedIn" justify="end" class="table-toolbar">
        <n-button v-if="auth.signedIn" type="primary" @click="showCreateModal = true">
          {{ t('discussion.newTopic') }}
        </n-button>
      </n-space>
      <n-empty
        v-if="!loading && !topics.length"
        class="empty-state"
        :description="t('discussion.empty')"
      >
        <template #extra>
          <n-button v-if="auth.signedIn" secondary size="small" @click="showCreateModal = true">
            {{ t('discussion.newTopic') }}
          </n-button>
          <span v-else class="muted">{{ t('discussion.signInTopic') }}</span>
        </template>
      </n-empty>
      <n-data-table
        v-else
        remote
        :columns="columns"
        :data="topics"
        :bordered="false"
        :loading="loading"
        :pagination="pagination"
        :scroll-x="760"
        class="admin-table"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
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

<style scoped lang="scss">
.empty-state {
  padding: 48px 0;
}
</style>
