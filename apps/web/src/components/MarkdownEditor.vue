<script setup lang="ts">
import { MdEditor } from 'md-editor-v3'
import type { ToolbarNames } from 'md-editor-v3'
import { apiFetch } from '../api'
import 'md-editor-v3/lib/style.css'

const props = defineProps<{
  modelValue: string
  problemId?: number | null
  uploadEnabled?: boolean
  placeholder?: string
  previewOnly?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const canUpload = computed(() => props.uploadEnabled === true && !!props.problemId)
const editorTheme = ref<'light' | 'dark'>('light')
const toolbars: ToolbarNames[] = [
  'revoke',
  'next',
  '-',
  'bold',
  'italic',
  'strikeThrough',
  'title',
  'quote',
  'unorderedList',
  'orderedList',
  'task',
  '-',
  'code',
  'link',
  'image',
  'table',
  '=',
  'preview',
  'fullscreen'
]

let themeObserver: MutationObserver | null = null

onMounted(() => {
  syncTheme()
  themeObserver = new MutationObserver(syncTheme)
  themeObserver.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['data-theme']
  })
})

onBeforeUnmount(() => {
  themeObserver?.disconnect()
})

function update(value: string) {
  emit('update:modelValue', value)
}

function syncTheme() {
  editorTheme.value = document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light'
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
      :theme="editorTheme"
      :preview="!previewOnly"
      :preview-only="previewOnly"
      :html-preview="false"
      :footers="[]"
      :toolbars="toolbars"
      input-box-width="50%"
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
    --md-color: var(--text-color);
    --md-hover-color: var(--text-color);
    --md-bk-color: var(--surface-bg);
    --md-bk-color-outstand: color-mix(in srgb, var(--surface-bg) 92%, var(--text-color) 8%);
    --md-bk-hover-color: color-mix(in srgb, var(--surface-bg) 88%, var(--brand) 12%);
    border-color: var(--border-color);
    border-radius: var(--radius-md);
    --md-border-color: var(--border-color);
    --md-border-hover-color: var(--border-strong);
    --md-border-active-color: var(--brand);
  }

  :deep(.md-editor-toolbar-wrapper) {
    border-bottom-color: var(--border-color);
  }

  :deep(.md-editor.md-editor-dark) {
    --md-bk-color: color-mix(in srgb, var(--surface-bg) 94%, black 6%);
    --md-bk-color-outstand: color-mix(in srgb, var(--surface-bg) 82%, white 6%);
    --md-bk-hover-color: color-mix(in srgb, var(--surface-bg) 80%, var(--brand) 12%);
    --md-scrollbar-bg-color: color-mix(in srgb, var(--surface-bg) 88%, black 12%);
    --md-scrollbar-thumb-color: color-mix(in srgb, var(--surface-bg) 70%, white 16%);
    --md-scrollbar-thumb-hover-color: color-mix(in srgb, var(--surface-bg) 62%, white 22%);
  }

  :deep(.md-editor.md-editor-dark .cm-editor),
  :deep(.md-editor.md-editor-dark .cm-gutters),
  :deep(.md-editor.md-editor-dark .cm-content) {
    background: var(--md-bk-color);
  }

  :deep(.md-editor.md-editor-dark .cm-gutters) {
    border-right-color: var(--border-color);
    color: var(--muted-color);
  }

  :deep(.md-editor.md-editor-dark .cm-activeLine),
  :deep(.md-editor.md-editor-dark .cm-activeLineGutter) {
    background: color-mix(in srgb, var(--surface-bg) 78%, var(--brand) 8%);
  }

  :deep(.md-editor.md-editor-dark .md-editor-preview) {
    --md-theme-bg-color: transparent;
    --md-theme-bg-color-inset: color-mix(in srgb, var(--surface-bg) 88%, white 6%);
    --md-theme-code-block-bg-color: color-mix(in srgb, var(--surface-bg) 84%, black 16%);
    --md-theme-code-before-bg-color: var(--md-theme-code-block-bg-color);
    color: var(--text-color);
    background: transparent;
  }

  :deep(.md-editor-code .md-editor-code-head) {
    z-index: 1;
  }
}
</style>
