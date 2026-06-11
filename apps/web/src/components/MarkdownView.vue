<script setup lang="ts">
import MarkdownIt from 'markdown-it'
import katexPluginModule from '@vscode/markdown-it-katex'
import { tasklist } from '@mdit/plugin-tasklist'
import 'katex/dist/katex.min.css'

const props = defineProps<{
  source: string
}>()

type MarkdownPlugin = (md: MarkdownIt, options?: unknown) => void
const prismVersion = '1.30.0'
const prismBase = `https://cdn.jsdelivr.net/npm/prismjs@${prismVersion}/`
const root = ref<HTMLElement | null>(null)

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

onMounted(() => {
  void highlightCode()
})

watch(rendered, () => {
  void highlightCode()
})

async function highlightCode() {
  await nextTick()
  if (!root.value?.querySelector('pre code[class*="language-"]')) return

  try {
    const element = root.value
    if (!element) return
    const prism = await loadPrism()
    if (!prism.highlightAllUnder) return
    prism.highlightAllUnder(element)
  } catch (error) {
    console.warn('Markdown code highlight unavailable:', error)
  }
}

async function loadPrism() {
  if (window.Prism?.highlightAllUnder) return window.Prism
  if (window.__dojPrismReady) return window.__dojPrismReady

  window.__dojPrismReady = (async () => {
    loadStyle(`${prismBase}themes/prism-tomorrow.css`, 'doj-prism-theme')
    window.Prism = window.Prism || {}
    window.Prism.manual = true
    await loadScript(`${prismBase}prism.min.js`, 'doj-prism-core')
    await loadScript(
      `${prismBase}plugins/autoloader/prism-autoloader.min.js`,
      'doj-prism-autoloader'
    )
    if (window.Prism?.plugins?.autoloader) {
      window.Prism.plugins.autoloader.languages_path = `${prismBase}components/`
    }
    if (!window.Prism?.highlightAllUnder) throw new Error('Prism did not expose highlightAllUnder.')
    return window.Prism
  })()

  return window.__dojPrismReady
}

function loadStyle(href: string, id: string) {
  if (document.getElementById(id)) return
  const link = document.createElement('link')
  link.id = id
  link.rel = 'stylesheet'
  link.href = href
  document.head.appendChild(link)
}

function loadScript(src: string, id: string) {
  return new Promise<void>((resolve, reject) => {
    const existing = document.getElementById(id) as HTMLScriptElement | null
    if (existing?.dataset.loaded === 'true') {
      resolve()
      return
    }
    if (existing) {
      existing.addEventListener('load', () => resolve(), { once: true })
      existing.addEventListener('error', () => reject(new Error(`Failed to load ${src}`)), {
        once: true
      })
      return
    }

    const script = document.createElement('script')
    script.id = id
    script.src = src
    script.async = true
    script.addEventListener(
      'load',
      () => {
        script.dataset.loaded = 'true'
        resolve()
      },
      { once: true }
    )
    script.addEventListener('error', () => reject(new Error(`Failed to load ${src}`)), {
      once: true
    })
    document.head.appendChild(script)
  })
}

interface PrismApi {
  manual?: boolean
  highlightAllUnder?: (element: Element) => void
  plugins?: {
    autoloader?: {
      languages_path: string
    }
  }
}

declare global {
  interface Window {
    Prism?: PrismApi
    __dojPrismReady?: Promise<PrismApi>
  }
}
</script>

<template>
  <div ref="root" class="markdown-view" v-html="rendered" />
</template>
