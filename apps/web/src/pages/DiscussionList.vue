<script setup lang="ts">
import { NAvatar, NButton, NIcon, NSpace, NTag, NTooltip } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import type { Component } from 'vue'
import { AddOutline, PinOutline, PinSharp, TrashOutline } from '@vicons/ionicons5'
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
const canManage = computed(() => auth.user?.admin ?? false)
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
    minWidth: 280,
    render(row) {
      return h('div', { class: 'topic-cell' }, [
        h('div', { class: 'topic-title-row' }, [
          row.pinned
            ? h(NTag, { type: 'success', bordered: false, size: 'small' }, () => t('discussion.pinned'))
            : null,
          h(RouterLink, { to: `/discussion/${row.id}`, class: 'table-link topic-title' }, () => row.title)
        ]),
        h(NSpace, { size: 6, align: 'center', wrap: false, class: 'topic-tags' }, () =>
          row.tags.map((tag) => h(NTag, { key: tag, bordered: false, size: 'small' }, () => tag))
        )
      ])
    }
  },
  {
    title: t('discussion.author'),
    key: 'author',
    width: 170,
    render(row) {
      return h('div', { class: 'topic-author' }, [
        h(NAvatar, { size: 24, src: row.author.avatarUrl, round: true }),
        h('span', row.author.name)
      ])
    }
  },
  {
    title: t('discussion.updated'),
    key: 'updatedAt',
    width: 190,
    render(row) {
      return new Date(row.updatedAt).toLocaleString()
    }
  },
  ...(canManage.value
    ? [
        {
          title: t('admin.actions'),
          key: 'actions',
          width: 112,
          align: 'right' as const,
          render(row: TopicRow) {
            return h(NSpace, { size: 6, justify: 'end' }, () => [
              tooltipIconButton(
                row.pinned ? PinSharp : PinOutline,
                row.pinned ? t('discussion.unpin') : t('discussion.pinned'),
                () => togglePinned(row)
              ),
              tooltipIconButton(TrashOutline, t('admin.delete'), () => deleteTopic(row), { type: 'error' })
            ])
          }
        }
      ]
    : [])
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

async function togglePinned(row: TopicRow) {
  try {
    await apiFetch(`/api/admin/discussion/topics/${row.id}`, {
      method: 'PATCH',
      body: JSON.stringify({ pinned: !row.pinned })
    })
    await loadTopics()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  }
}

async function deleteTopic(row: TopicRow) {
  try {
    await apiFetch(`/api/admin/discussion/topics/${row.id}`, { method: 'DELETE' })
    await loadTopics()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  }
}

function renderIcon(icon: Component) {
  return h(NIcon, { component: icon })
}

function tooltipIconButton(
  icon: Component,
  label: string,
  onClick: () => void,
  options: { type?: 'error' } = {}
) {
  return h(
    NTooltip,
    { trigger: 'hover' },
    {
      trigger: () =>
        h(
          NButton,
          {
            size: 'small',
            quaternary: true,
            circle: true,
            type: options.type,
            onClick
          },
          { icon: () => renderIcon(icon) }
        ),
      default: () => label
    }
  )
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
          <template #icon>
            <n-icon :component="AddOutline" />
          </template>
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
        :scroll-x="canManage ? 840 : 720"
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

.topic-cell {
  display: grid;
  gap: 5px;
  min-width: 0;
}

.topic-title-row {
  display: flex;
  gap: 6px;
  align-items: center;
  min-width: 0;
}

.topic-title {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.topic-tags {
  min-height: 22px;
  overflow: hidden;
}

.topic-author {
  display: grid;
  grid-template-columns: 24px minmax(0, 1fr);
  gap: 8px;
  align-items: center;

  span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}
</style>
