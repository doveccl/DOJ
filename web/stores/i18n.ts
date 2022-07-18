/// <reference types="@intlify/vite-plugin-vue-i18n/client" />

import { createI18n } from 'vue-i18n'
import { usePreferredLanguages, useLocalStorage } from '@vueuse/core'
import messages from '@intlify/vite-plugin-vue-i18n/messages'
import zh from 'element-plus/lib/locale/lang/zh-cn'
import en from 'element-plus/lib/locale/lang/en'

const languages = usePreferredLanguages().value
const locale = useLocalStorage('locale', languages[0] ?? 'en')

export const useI18nStore = defineStore('i18n', () => {
  const i18n = useI18n({ useScope: 'global' })
  const { t, availableLocales: locales } = i18n
  const names = locales.map(l => t('language', l, { locale: l }))

  const elocale = computed(() => {
    switch (locale.value) {
      case 'zh': return zh
      default: return en
    }
  })

  function setLocale(value: string) {
    if (locales.includes(value)) {
      locale.value = i18n.locale.value = value
    }
  }

  return { locale, locales, names, elocale, setLocale }
})

export const i18n = createI18n({
  messages,
  locale: locale.value
})
