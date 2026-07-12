import { Card, Flex } from 'antd'
import type { ReactNode } from 'react'

import { MarkdownPreview } from './markdown'

export function DescriptionCard({ id, header, extra, value, children }: {
  id: string
  header: ReactNode
  extra?: ReactNode
  value: string
  children?: ReactNode
}) {
  return (
    <Card
      title={
        <Flex align="center" justify="space-between" gap={12} wrap style={{ width: '100%' }}>
          <div style={{ flex: '1 1 240px', minWidth: 0 }}>{header}</div>
          {extra ? <div style={{ maxWidth: '100%', fontWeight: 400 }}>{extra}</div> : null}
        </Flex>
      }
      styles={{ title: { whiteSpace: 'normal' } }}
    >
      {children ?? (value.trim() ? <MarkdownPreview id={id} value={value} /> : null)}
    </Card>
  )
}
