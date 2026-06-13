<script setup lang="ts">
import { MdPreview } from 'md-editor-v3'
import 'md-editor-v3/lib/preview.css'
import 'katex/dist/katex.min.css'

defineProps<{
  source: string
}>()

const editorTheme = ref<'light' | 'dark'>('light')
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

function syncTheme() {
  editorTheme.value = document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light'
}
</script>

<template>
  <div class="md-render-view">
    <md-preview
      :model-value="source || ''"
      :theme="editorTheme"
      preview-theme="github"
      code-theme="github"
      language="zh-CN"
      :no-mermaid="false"
      :no-katex="false"
    />
  </div>
</template>

<style scoped lang="scss">
.md-render-view {
  min-width: 0;

  :deep(.md-editor) {
    --md-color: var(--text-color);
    --md-hover-color: var(--text-color);
    --md-bk-color: transparent;
    --md-bk-color-outstand: color-mix(in srgb, var(--surface-bg) 88%, var(--text-color) 10%);
    --md-bk-hover-color: color-mix(in srgb, var(--surface-bg) 86%, var(--brand) 12%);
    --md-border-color: var(--border-color);
    --md-border-hover-color: var(--border-strong);
    --md-border-active-color: var(--brand);
    background: transparent;
  }

  :deep(.md-editor-preview-wrapper) {
    padding: 0;
    background: transparent;
  }

  :deep(.md-editor-preview) {
    color: var(--text-color);
    background: transparent;
    font-size: 15px;
    line-height: 1.72;
    overflow-wrap: anywhere;
  }

  :deep(.md-editor.md-editor-dark .md-editor-preview) {
    --md-theme-bg-color: transparent;
    --md-theme-bg-color-inset: color-mix(in srgb, var(--surface-bg) 88%, white 6%);
    --md-theme-code-block-bg-color: color-mix(in srgb, var(--surface-bg) 84%, black 16%);
    --md-theme-code-before-bg-color: var(--md-theme-code-block-bg-color);
  }

  :deep(.md-editor-code .md-editor-code-head) {
    z-index: 1;
  }

  :deep(code) {
    font-family:
      'SFMono-Regular',
      Consolas,
      'Liberation Mono',
      monospace;
  }
}
</style>
