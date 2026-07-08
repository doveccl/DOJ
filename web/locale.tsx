import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import enUS from 'antd/es/locale/en_US'
import zhCN from 'antd/es/locale/zh_CN'
import dayjs from 'dayjs'
import 'dayjs/locale/en'
import 'dayjs/locale/zh-cn'

import { en } from './locales/en'
import { zh } from './locales/zh'

export type Lang = 'zh' | 'en'
type DurationUnit = 'second' | 'minute' | 'hour' | 'day'

const dicts = { zh, en }
const localeCodes: Record<Lang, string> = { zh: 'zh-CN', en: 'en-US' }
const htmlLangs: Record<Lang, string> = { zh: localeCodes.zh, en: 'en' }
const dayjsLangs: Record<Lang, string> = { zh: 'zh-cn', en: 'en' }
const antdLocales = { zh: zhCN, en: enUS }
const durationUnits: Record<Lang, Record<DurationUnit, string> & { tight: boolean }> = {
  zh: { second: '秒', minute: '分钟', hour: '小时', day: '天', tight: false },
  en: { second: 's', minute: 'm', hour: 'h', day: 'd', tight: true }
}

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

export function localeCode(lang: Lang) {
  return localeCodes[lang]
}

export function antdLocale(lang: Lang) {
  return antdLocales[lang]
}

export function durationText(value: number, unit: DurationUnit, lang: Lang) {
  const dict = durationUnits[lang]
  return dict.tight ? `${value}${dict[unit]}` : `${value} ${dict[unit]}`
}
