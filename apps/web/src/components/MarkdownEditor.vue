<script setup lang="ts">
import { NButton, NTabPane, NTabs, NUpload, type UploadCustomRequestOptions } from 'naive-ui'
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../api'
import CodeEditor from './CodeEditor.vue'
import MarkdownView from './MarkdownView.vue'

const props = defineProps<{
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const { t } = useI18n()
const tab = ref<'write' | 'preview'>('write')

function update(value: string) {
  emit('update:modelValue', value)
}

function insert(snippet: string) {
  const next = props.modelValue ? `${props.modelValue}\n${snippet}\n` : `${snippet}\n`
  emit('update:modelValue', next)
}

async function uploadImage({ file, onFinish, onError }: UploadCustomRequestOptions) {
  if (!file.file) {
    onError()
    return
  }
  try {
    const body = new FormData()
    body.append('file', file.file)
    const result = await apiFetch<{ url: string; filename: string }>('/api/media', {
      method: 'POST',
      body
    })
    insert(`![${result.filename}](${result.url})`)
    onFinish()
  } catch {
    onError()
  }
}
</script>

<template>
  <div class="markdown-editor">
    <n-tabs v-model:value="tab" type="segment" size="small">
      <n-tab-pane name="write" :tab="t('mdEditor.write')">
        <code-editor
          :model-value="modelValue"
          language-id="markdown"
          @update:model-value="update"
        />
      </n-tab-pane>
      <n-tab-pane name="preview" :tab="t('mdEditor.preview')">
        <markdown-view :source="modelValue" class="markdown-editor-preview" />
      </n-tab-pane>
    </n-tabs>
    <div class="markdown-editor-toolbar">
      <n-upload :custom-request="uploadImage" :show-file-list="false" accept="image/*">
        <n-button size="small" quaternary>{{ t('mdEditor.uploadImage') }}</n-button>
      </n-upload>
    </div>
  </div>
</template>

<style scoped lang="scss">
.markdown-editor {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;

  :deep(.n-tabs),
  :deep(.n-tab-pane) {
    width: 100%;
  }
}

.markdown-editor-preview {
  min-height: 320px;
  padding: 12px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
}

.markdown-editor-toolbar {
  display: flex;
  justify-content: flex-end;
}
</style>
