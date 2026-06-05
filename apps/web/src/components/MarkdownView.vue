<script setup lang="ts">
import MarkdownIt from 'markdown-it'
import katexPluginModule from '@vscode/markdown-it-katex'
import { tasklist } from '@mdit/plugin-tasklist'
import { computed } from 'vue'
import 'katex/dist/katex.min.css'

const props = defineProps<{
  source: string
}>()

type MarkdownPlugin = (md: MarkdownIt, options?: unknown) => void

const katexPlugin =
  typeof katexPluginModule === 'function'
    ? (katexPluginModule as MarkdownPlugin)
    : ((katexPluginModule as unknown as { default: MarkdownPlugin }).default as MarkdownPlugin)

const markdown = new MarkdownIt({
  html: false,
  linkify: true,
  typographer: true
})
  .use(katexPlugin, { throwOnError: false })
  .use(tasklist, { disabled: true })

const rendered = computed(() => markdown.render(props.source || ''))
</script>

<template>
  <div class="markdown-view" v-html="rendered" />
</template>
