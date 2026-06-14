<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { apiFetch, isUnauthorized } from '../api'
import MarkdownEditor from '../components/MarkdownEditor.vue'
import MarkdownView from '../components/MarkdownView.vue'
import { useAuthStore } from '../stores/auth'

interface Topic {
  id: number
  title: string
  tags: string[]
  pinned: boolean
  createdAt: string
  updatedAt: string
}

interface UserBrief {
  id: number
  name: string
  avatarUrl: string
}

interface Post {
  id: number
  topicId: number
  user: UserBrief
  content: string
  createdAt: string
}

interface TopicDetail {
  id: number
  title: string
  tags: string[]
  pinned: boolean
  author?: UserBrief
  createdAt: string
  updatedAt: string
  posts: Post[]
}

const route = useRoute()
const auth = useAuthStore()
const loading = ref(true)
const replying = ref(false)
const error = ref('')
const requireSignIn = ref(false)
const detail = ref<TopicDetail | null>(null)
const replyText = ref('')
const { t } = useI18n()
const topic = computed<Topic>(() => ({
  id: detail.value?.id ?? 0,
  title: detail.value?.title ?? '',
  tags: detail.value?.tags ?? [],
  pinned: detail.value?.pinned ?? false,
  createdAt: detail.value?.createdAt ?? '',
  updatedAt: detail.value?.updatedAt ?? ''
}))
const posts = computed(() => detail.value?.posts ?? [])

async function loadDetail() {
  loading.value = true
  error.value = ''
  try {
    detail.value = await apiFetch<TopicDetail>(`/api/discussion/topics/${route.params.id}`)
    requireSignIn.value = false
  } catch (cause) {
    if (isUnauthorized(cause)) {
      requireSignIn.value = true
      detail.value = null
    } else {
      error.value = cause instanceof Error ? cause.message : String(cause)
    }
  } finally {
    loading.value = false
  }
}

async function createReply() {
  if (!detail.value) return

  replying.value = true
  error.value = ''
  try {
    await apiFetch(`/api/discussion/topics/${topic.value.id}/posts`, {
      method: 'POST',
      body: JSON.stringify({ content: replyText.value })
    })
    replyText.value = ''
    await loadDetail()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    replying.value = false
  }
}

onMounted(loadDetail)

watch(
  () => auth.signedIn,
  () => {
    void loadDetail()
  }
)
</script>

<template>
  <main class="page">
    <n-spin :show="loading">
      <template v-if="detail">
        <section class="page-header">
          <h1>{{ topic.title }}</h1>
        </section>

        <div class="meta-row">
          <n-tag v-for="tag in topic.tags" :key="tag" :bordered="false">{{ tag }}</n-tag>
        </div>

        <n-card
          v-for="(post, index) in posts"
          :key="post.id"
          :bordered="false"
          class="stacked-card"
        >
          <template #header>
            <div class="post-head">
              <strong>{{ post.user.name }}</strong>
              <n-tag v-if="index === 0" size="small" :bordered="false" type="success">
                {{ t('discussion.topicOwner') }}
              </n-tag>
              <span class="muted">#{{ index + 1 }}</span>
            </div>
          </template>
          <markdown-view :source="post.content" />
          <p class="muted reply-time">{{ new Date(post.createdAt).toLocaleString() }}</p>
        </n-card>

        <n-card
          v-if="auth.signedIn"
          :title="t('discussion.reply')"
          :bordered="false"
          class="stacked-card"
        >
          <markdown-editor v-model="replyText" />
          <n-space justify="end" class="form-actions">
            <n-button
              type="primary"
              :loading="replying"
              :disabled="!replyText"
              @click="createReply"
            >
              {{ t('discussion.reply') }}
            </n-button>
          </n-space>
        </n-card>
      </template>

      <sign-in-required v-else-if="requireSignIn" />

      <n-alert v-else-if="error" type="error" class="page-alert">
        {{ error }}
      </n-alert>
    </n-spin>
  </main>
</template>

<style scoped lang="scss">
.post-head {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.post-head strong {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
