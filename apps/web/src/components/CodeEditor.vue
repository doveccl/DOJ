<script setup lang="ts">
import type { LanguageSupport } from '@codemirror/language'
import CodeMirror from 'vue-codemirror6'
import { onMounted, onUnmounted, ref, watch } from 'vue'

const props = defineProps<{
  modelValue: string
  languageId?: string
  disabled?: boolean
  readonly?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const isDark = ref(false)
const language = ref<LanguageSupport>()
let observer: MutationObserver | undefined
let languageLoadId = 0

watch(
  () => props.languageId,
  async (languageId) => {
    const loadId = ++languageLoadId
    language.value = undefined
    if (languageId === 'c' || languageId === 'cpp') {
      const { cpp } = await import('@codemirror/lang-cpp')
      if (loadId === languageLoadId) language.value = cpp()
      return
    }
    if (languageId === 'py' || languageId === 'python') {
      const { python } = await import('@codemirror/lang-python')
      if (loadId === languageLoadId) language.value = python()
      return
    }
    if (languageId === 'js' || languageId === 'javascript' || languageId === 'ts') {
      const { javascript } = await import('@codemirror/lang-javascript')
      if (loadId === languageLoadId)
        language.value = javascript({ typescript: languageId === 'ts' })
    }
  },
  { immediate: true }
)

onMounted(() => {
  const updateTheme = () => {
    isDark.value = document.documentElement.dataset.theme === 'dark'
  }
  updateTheme()
  observer = new MutationObserver(updateTheme)
  observer.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] })
})

onUnmounted(() => {
  observer?.disconnect()
})

function updateValue(value: unknown) {
  emit('update:modelValue', typeof value === 'string' ? value : String(value ?? ''))
}
</script>

<template>
  <code-mirror
    class="code-editor"
    :model-value="modelValue"
    :lang="language"
    :disabled="disabled"
    :readonly="readonly"
    :dark="isDark"
    basic
    tab
    wrap
    @update:model-value="updateValue"
  />
</template>
