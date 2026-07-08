import { Alert, Empty, Spin } from 'antd'

import { useLocale } from '../locale'
import './state.css'

export function LoadingBlock() {
  return (
    <div className="centerBlock">
      <Spin />
    </div>
  )
}

export function ErrorBlock({ error }: { error: unknown }) {
  const { text } = useLocale()
  const description = error instanceof Error ? error.message : String(error)

  return <Alert type="error" title={text.common.loadingFailed} description={description} showIcon />
}

export function EmptyBlock() {
  return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} />
}
