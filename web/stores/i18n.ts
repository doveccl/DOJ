import { createI18n } from 'vue-i18n'
import messages from '@intlify/unplugin-vue-i18n/messages'

import en from 'element-plus/lib/locale/lang/en'
import zhCN from 'element-plus/lib/locale/lang/zh-cn'

const lan = usePreferredLanguages().value.find(l => l in messages)
const locale = useLocalStorage('locale', lan ?? 'en')

export const useI18nStore = defineStore('i18n', () => {
  const i18n = useI18n()

  watchEffect(() => (locale.value = i18n.locale.value))
  const elocale = computed(() => {
    switch (i18n.locale.value) {
      case 'zh-CN':
        return zhCN
      default:
        return en
    }
  })

  return { elocale }
})

export const i18n = createI18n({ messages, locale: locale.value })
