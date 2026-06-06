<script setup lang="ts">
import { NAlert, NButton, NCard, NSpace, NSpin, NTag } from 'naive-ui'
import { onMounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../api'
import MarkdownEditor from '../components/MarkdownEditor.vue'
import MarkdownView from '../components/MarkdownView.vue'
import { useAuthStore } from '../stores/auth'

interface Topic {
  id: number
  title: string
  tags: string[]
  userName: string
  linkedProblemId: number | null
  linkedContestId: number | null
  createdAt: string
}

interface Reply {
  id: number
  userName: string
  contentMarkdown: string
  createdAt: string
}

interface TopicDetail {
  topic: Topic
  replies: Reply[]
}

const route = useRoute()
const auth = useAuthStore()
const loading = ref(true)
const replying = ref(false)
const error = ref('')
const detail = ref<TopicDetail | null>(null)
const replyText = ref('')
const { t } = useI18n()

async function loadDetail() {
  loading.value = true
  error.value = ''
  try {
    detail.value = await apiFetch<TopicDetail>(`/api/discussion/topics/${route.params.id}`)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

async function createReply() {
  if (!detail.value) return

  replying.value = true
  error.value = ''
  try {
    await apiFetch(`/api/discussion/topics/${detail.value.topic.id}/replies`, {
      method: 'POST',
      body: JSON.stringify({ contentMarkdown: replyText.value })
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
</script>

<template>
  <main class="page">
    <n-spin :show="loading">
      <template v-if="detail">
        <section class="page-header">
          <h1>{{ detail.topic.title }}</h1>
          <p>
            {{ t('discussion.by') }} {{ detail.topic.userName }} ·
            {{ new Date(detail.topic.createdAt).toLocaleString() }}
          </p>
        </section>

        <div class="meta-row">
          <n-tag v-for="tag in detail.topic.tags" :key="tag" :bordered="false">{{ tag }}</n-tag>
          <RouterLink
            v-if="detail.topic.linkedProblemId"
            class="table-link"
            :to="`/problems/${detail.topic.linkedProblemId}`"
          >
            {{ t('discussion.problem') }} {{ detail.topic.linkedProblemId }}
          </RouterLink>
          <RouterLink
            v-if="detail.topic.linkedContestId"
            class="table-link"
            :to="`/contests/${detail.topic.linkedContestId}`"
          >
            {{ t('discussion.contest') }} {{ detail.topic.linkedContestId }}
          </RouterLink>
        </div>

        <n-card
          v-for="reply in detail.replies"
          :key="reply.id"
          :title="reply.userName"
          :bordered="false"
          class="stacked-card"
        >
          <markdown-view :source="reply.contentMarkdown" />
          <p class="muted reply-time">{{ new Date(reply.createdAt).toLocaleString() }}</p>
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

      <n-alert v-else-if="error" type="error" class="page-alert">
        {{ error }}
      </n-alert>
    </n-spin>
  </main>
</template>
