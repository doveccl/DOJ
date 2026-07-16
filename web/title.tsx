import { createContext, useContext, useEffect, useLayoutEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { useLocation, useNavigationType } from 'react-router-dom'

type PageTitleState = {
  title: string
  setTitle: (title: string) => void
}

const PageTitleContext = createContext<PageTitleState | null>(null)

export function PageTitleProvider({ children }: { children: ReactNode }) {
  const location = useLocation()
  const [title, setTitle] = useState('')

  useEffect(() => setTitle(''), [location.pathname])

  const value = useMemo(() => ({ title, setTitle }), [title])
  return <PageTitleContext.Provider value={value}>{children}</PageTitleContext.Provider>
}

export function usePageTitle(title: string | undefined) {
  const state = useContext(PageTitleContext)
  useEffect(() => {
    state?.setTitle(title?.trim() ?? '')
  }, [state, title])
}

export function useCurrentPageTitle() {
  return useContext(PageTitleContext)?.title ?? ''
}

export function ScrollManager() {
  const location = useLocation()
  const navigationType = useNavigationType()

  useLayoutEffect(() => {
    if (navigationType !== 'POP') {
      window.scrollTo({ top: 0, left: 0 })
    }
  }, [location.key, navigationType])

  return null
}
