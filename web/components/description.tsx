import { Card } from 'antd'
import type { ReactNode } from 'react'

import { MarkdownPreview } from './markdown'

export function DescriptionCard({ id, header, extra, value }: {
  id: string
  header: ReactNode
  extra?: ReactNode
  value: string
}) {
  return (
    <Card
      title={header}
      extra={extra}
      styles={!value.trim() ? { body: { display: 'none' } } : undefined}
    >
      {value.trim() ? <MarkdownPreview id={id} value={value} /> : null}
    </Card>
  )
}
