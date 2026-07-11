import { Alert, Empty, Spin } from 'antd'

import { APIError } from '../client'
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
	const title = error instanceof APIError && error.status === 403
		? text.common.forbidden
		: error instanceof APIError && error.status === 404
			? text.common.notFound
			: text.common.loadingFailed

	return <Alert type="error" title={title} description={description} showIcon />
}

export function EmptyBlock() {
  return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} />
}
