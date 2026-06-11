<script setup lang="ts">
import { MdPreview } from 'md-editor-v3'
import 'md-editor-v3/lib/preview.css'
import 'katex/dist/katex.min.css'

defineProps<{
  source: string
}>()
</script>

<template>
  <div class="markdown-view">
    <md-preview
      :model-value="source || ''"
      preview-theme="github"
      code-theme="github"
      language="zh-CN"
      :no-mermaid="false"
      :no-katex="false"
    />
  </div>
</template>

<style scoped lang="scss">
.markdown-view {
  min-width: 0;

  :deep(.md-editor-preview-wrapper) {
    padding: 0;
    background: transparent;
  }

  :deep(.md-editor-preview) {
    color: var(--text-color);
    background: transparent;
    font-size: 15px;
    line-height: 1.72;
  }

  :deep(pre) {
    border-radius: var(--radius-md);
    background: color-mix(in srgb, var(--surface-bg) 86%, var(--text-color) 14%);
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
