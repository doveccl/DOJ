import React from 'react'
import ReactDOM from 'react-dom/client'
import { ConfigProvider, App as AntApp, theme as antdTheme } from 'antd'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

import 'antd/dist/reset.css'
import './style.css'
import { ColorProvider, useColor } from './color'
import { LiveEvents } from './components/live'
import { antdLocale, LocaleProvider, useLocale } from './locale'
import { AppRoutes } from './routes'
import { SessionProvider } from './session'
import { shouldRetryQuery } from './client'

declare global {
  interface Window {
    __dojRoot?: ReturnType<typeof ReactDOM.createRoot>
  }
}

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
		retry: shouldRetryQuery
    },
    mutations: {
      retry: false
    }
  }
})

function Root() {
  const { lang } = useLocale()
  const { color } = useColor()

  return (
    <ConfigProvider
      locale={antdLocale(lang)}
      theme={{
        algorithm: color === 'dark' ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
        cssVar: { key: 'doj' }
      }}
    >
      <AntApp>
        <AppRoutes />
      </AntApp>
    </ConfigProvider>
  )
}

const container = document.getElementById('root') as HTMLElement
const root = window.__dojRoot ?? ReactDOM.createRoot(container)
window.__dojRoot = root

root.render(
  <React.StrictMode>
    <LocaleProvider>
      <ColorProvider>
        <QueryClientProvider client={queryClient}>
          <SessionProvider>
            <LiveEvents />
            <Root />
          </SessionProvider>
        </QueryClientProvider>
      </ColorProvider>
    </LocaleProvider>
  </React.StrictMode>
)
