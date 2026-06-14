<script setup lang="ts">

const monacoVersion = '0.55.1'
const monacoBase = `https://cdn.jsdelivr.net/npm/monaco-editor@${monacoVersion}/min/vs`

const props = defineProps<{
  modelValue: string
  languageId?: string
  disabled?: boolean
  readonly?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const host = ref<HTMLElement | null>(null)
const isDark = ref(false)
const editorLanguage = computed(() => mapLanguage(props.languageId))
let observer: MutationObserver | undefined
let editor: MonacoEditor | null = null

onMounted(async () => {
  const updateTheme = () => {
    isDark.value = document.documentElement.dataset.theme === 'dark'
    if (window.monaco) window.monaco.editor.setTheme(isDark.value ? 'vs-dark' : 'vs')
  }
  updateTheme()
  observer = new MutationObserver(updateTheme)
  observer.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] })

  await nextTick()
  const monaco = await loadMonaco()
  if (!host.value) return
  editor = monaco.editor.create(host.value, {
    value: props.modelValue,
    language: editorLanguage.value,
    readOnly: props.readonly || props.disabled,
    automaticLayout: true,
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    wordWrap: 'off',
    tabSize: 2,
    fontSize: 14,
    theme: isDark.value ? 'vs-dark' : 'vs'
  })
  editor.onDidChangeModelContent(() => {
    const nextValue = editor?.getValue() ?? ''
    if (nextValue !== props.modelValue) emit('update:modelValue', nextValue)
  })
})

onUnmounted(() => {
  observer?.disconnect()
  editor?.dispose()
  editor = null
})

watch(
  () => props.modelValue,
  (value) => {
    if (editor && editor.getValue() !== value) editor.setValue(value)
  }
)

watch(editorLanguage, (language) => {
  const model = editor?.getModel()
  if (model && window.monaco) window.monaco.editor.setModelLanguage(model, language)
})

watch(
  () => [props.readonly, props.disabled] as const,
  () => {
    editor?.updateOptions({ readOnly: props.readonly || props.disabled })
  }
)

function loadMonaco() {
  if (window.monaco) return Promise.resolve(window.monaco)
  if (window.__dojMonacoReady) return window.__dojMonacoReady

  window.__dojMonacoReady = new Promise((resolve, reject) => {
    const script = document.createElement('script')
    script.src = `${monacoBase}/loader.js`
    script.async = true
    script.onload = () => {
      window.MonacoEnvironment = {
        getWorkerUrl() {
          const worker = [
            'self.MonacoEnvironment={baseUrl:"',
            `${monacoBase}/`,
            '"};importScripts("',
            `${monacoBase}/base/worker/workerMain.js`,
            '");'
          ].join('')
          return `data:text/javascript;charset=utf-8,${encodeURIComponent(worker)}`
        }
      }
      window.require.config({ paths: { vs: monacoBase } })
      window.require(['vs/editor/editor.main'], () => {
        if (!window.monaco) {
          reject(new Error('Monaco editor loaded without global API.'))
          return
        }
        resolve(window.monaco)
      })
    }
    script.onerror = () => reject(new Error('Failed to load Monaco editor from CDN.'))
    document.head.appendChild(script)
  })

  return window.__dojMonacoReady
}

function mapLanguage(languageId?: string) {
  if (languageId === 'py') return 'python'
  if (languageId === 'sh') return 'shell'
  if (languageId === 'ts') return 'typescript'
  if (languageId === 'js') return 'javascript'
  if (languageId === 'c') return 'c'
  if (languageId === 'cc' || languageId === 'cpp') return 'cpp'
  if (languageId === 'rs') return 'rust'
  return languageId || 'cpp'
}

interface MonacoEditor {
  dispose(): void
  getModel(): unknown
  getValue(): string
  onDidChangeModelContent(listener: () => void): void
  setValue(value: string): void
  updateOptions(options: { readOnly: boolean }): void
}

interface MonacoApi {
  editor: {
    create(host: HTMLElement, options: Record<string, unknown>): MonacoEditor
    setModelLanguage(model: unknown, language: string): void
    setTheme(theme: string): void
  }
}

declare global {
  interface Window {
    monaco?: MonacoApi
    MonacoEnvironment?: {
      getWorkerUrl(): string
    }
    require: {
      (modules: string[], callback: () => void): void
      config(options: { paths: { vs: string } }): void
    }
    __dojMonacoReady?: Promise<MonacoApi>
  }
}
</script>

<template>
  <div ref="host" class="code-editor" />
</template>
