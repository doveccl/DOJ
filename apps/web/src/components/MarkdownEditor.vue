<script setup lang="ts">
import { MdEditor } from 'md-editor-v3'
import { apiFetch } from '../api'
import 'md-editor-v3/lib/style.css'

const props = defineProps<{
  modelValue: string
  problemId?: number | null
  uploadEnabled?: boolean
  placeholder?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const canUpload = computed(() => props.uploadEnabled === true && !!props.problemId)

function update(value: string) {
  emit('update:modelValue', value)
}

async function uploadImages(files: File[], callback: (urls: string[]) => void) {
  if (!canUpload.value || !props.problemId) {
    callback([])
    return
  }

  const urls = await Promise.all(
    files.map(async (file) => {
      const body = new FormData()
      body.append('file', file)
      body.append('path', `assets/${file.name}`)
      const result = await apiFetch<{ url: string | null }>(
        `/api/admin/problems/${props.problemId}/assets/upload`,
        {
          method: 'POST',
          body
        }
      )
      return result.url ?? ''
    })
  )
  callback(urls.filter(Boolean))
}
</script>

<template>
  <div class="markdown-editor">
    <md-editor
      :model-value="modelValue"
      :placeholder="placeholder"
      :no-upload-img="!canUpload"
      language="zh-CN"
      preview-theme="github"
      code-theme="github"
      @update:model-value="update"
      @on-upload-img="uploadImages"
    />
  </div>
</template>

<style scoped lang="scss">
.markdown-editor {
  width: 100%;

  :deep(.md-editor) {
    border-color: var(--border-color);
    border-radius: var(--radius-md);
  }
}
</style>
