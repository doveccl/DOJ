<script setup lang="ts">
import { cpp } from '@codemirror/lang-cpp'
import { javascript } from '@codemirror/lang-javascript'
import { python } from '@codemirror/lang-python'
import type { LanguageSupport } from '@codemirror/language'
import CodeMirror from 'vue-codemirror6'
import { computed, onMounted, onUnmounted, ref } from 'vue'

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
let observer: MutationObserver | undefined

const language = computed<LanguageSupport | undefined>(() => {
  if (props.languageId === 'c' || props.languageId === 'cpp') return cpp()
  if (props.languageId === 'py' || props.languageId === 'python') return python()
  if (props.languageId === 'js' || props.languageId === 'javascript' || props.languageId === 'ts') {
    return javascript({ typescript: props.languageId === 'ts' })
  }
  return undefined
})

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
