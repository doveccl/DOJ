import { App as AntApp } from 'antd'

import { useLocale } from '../locale'

export function useApiMessage() {
  const { text } = useLocale()
  const { message } = AntApp.useApp()
  const showError = (error: unknown) => {
    message.error(error instanceof Error ? error.message : text.common.loadingFailed)
  }
  return { message, showError }
}
