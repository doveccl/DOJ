import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import dayjs from 'dayjs'
import 'dayjs/locale/en'
import 'dayjs/locale/zh-cn'

import { en } from './locales/en'
import { zh } from './locales/zh'

export type Lang = 'zh' | 'en'

const dicts = { zh, en }
const htmlLangs: Record<Lang, string> = { zh: 'zh-CN', en: 'en' }
const dayjsLangs: Record<Lang, string> = { zh: 'zh-cn', en: 'en' }

type LocaleState = {
  lang: Lang
  text: typeof zh
  setLang: (lang: Lang) => void
}

const LocaleContext = createContext<LocaleState | null>(null)
const localeStorageKey = 'doj.locale'

function readLang(): Lang {
  if (typeof window === 'undefined') {
    return 'zh'
  }
  const stored = window.localStorage.getItem(localeStorageKey)
  if (stored === 'zh' || stored === 'en') {
    return stored
  }
  return window.navigator.language.toLowerCase().startsWith('zh') ? 'zh' : 'en'
}

export function LocaleProvider({ children }: { children: ReactNode }) {
  const [lang, setValue] = useState<Lang>(readLang)

  useEffect(() => {
    document.documentElement.lang = htmlLangs[lang]
    dayjs.locale(dayjsLangs[lang])
  }, [lang])

  const setLang = useCallback((next: Lang) => {
    setValue(next)
    window.localStorage.setItem(localeStorageKey, next)
  }, [])

  const value = useMemo(() => ({ lang, setLang, text: dicts[lang] }), [lang, setLang])

  return <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>
}

export function useLocale() {
  const value = useContext(LocaleContext)
  if (!value) {
    throw new Error('LocaleProvider is missing')
  }
  return value
}
