import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'

export type ColorMode = 'system' | 'light' | 'dark'
type ResolvedColor = 'light' | 'dark'

type ColorState = {
  mode: ColorMode
  color: ResolvedColor
  setMode: (mode: ColorMode) => void
}

const ColorContext = createContext<ColorState | null>(null)

function readMode(): ColorMode {
  if (typeof window === 'undefined') {
    return 'system'
  }
  const stored = window.localStorage.getItem('doj.theme')
  if (stored === 'system' || stored === 'light' || stored === 'dark') {
    return stored
  }
  return 'system'
}

function systemColor(): ResolvedColor {
  if (typeof window === 'undefined') {
    return 'light'
  }
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export function ColorProvider({ children }: { children: ReactNode }) {
  const [mode, setValue] = useState<ColorMode>(readMode)
  const [system, setSystem] = useState<ResolvedColor>(systemColor)

  useEffect(() => {
    const query = window.matchMedia('(prefers-color-scheme: dark)')
    const update = () => setSystem(query.matches ? 'dark' : 'light')
    update()
    query.addEventListener('change', update)
    return () => query.removeEventListener('change', update)
  }, [])

  const color = mode === 'system' ? system : mode

  useEffect(() => {
    document.documentElement.dataset.theme = color
    document.documentElement.style.colorScheme = color
  }, [color])

  const setMode = useCallback((next: ColorMode) => {
    setValue(next)
    window.localStorage.setItem('doj.theme', next)
  }, [])

  const value = useMemo(() => ({ mode, color, setMode }), [mode, color, setMode])

  return <ColorContext.Provider value={value}>{children}</ColorContext.Provider>
}

export function useColor() {
  const value = useContext(ColorContext)
  if (!value) {
    throw new Error('ColorProvider is missing')
  }
  return value
}
