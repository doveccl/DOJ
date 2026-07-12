import { EditOutlined, UnorderedListOutlined } from '@ant-design/icons'
import { Button, Flex, Space, Typography } from 'antd'
import type { ReactNode } from 'react'

import { DescriptionCard } from './description'
import { useLocale } from '../locale'

export function ScheduledDetailHeader({
  descriptionId,
  descriptionValue,
  title,
  titleTag,
  deadline,
  recordsHref,
  recordsLabel,
  admin,
  editing,
  onStartEdit,
  onCancelEdit,
  saving,
  editFormId,
  children
}: {
  descriptionId: string
  descriptionValue: string
  title: string
  titleTag?: ReactNode
  deadline: ReactNode
  recordsHref: string
  recordsLabel: string
  admin: boolean
  editing: boolean
  onStartEdit: () => void
  onCancelEdit: () => void
  saving: boolean
  editFormId: string
  children?: ReactNode
}) {
  const { text } = useLocale()
  return (
    <DescriptionCard
      id={descriptionId}
      value={descriptionValue}
      header={
        <Flex align="center" gap={10} style={{ minWidth: 0 }}>
          <Typography.Text ellipsis={{ tooltip: title }} style={{ minWidth: 0 }}>
            {title}
          </Typography.Text>
          {titleTag}
        </Flex>
      }
      extra={
        editing ? (
          <Space size={8}>
            <Button onClick={onCancelEdit}>{text.common.cancel}</Button>
            <Button type="primary" htmlType="submit" form={editFormId} loading={saving}>{text.common.save}</Button>
          </Space>
        ) : (
          <Space size={16} wrap>
            {deadline}
            <Space size={8} wrap>
              <Button icon={<UnorderedListOutlined />} href={recordsHref}>{recordsLabel}</Button>
              {admin ? <Button icon={<EditOutlined />} onClick={onStartEdit}>{text.common.edit}</Button> : null}
            </Space>
          </Space>
        )
      }
    >
      {editing ? children : undefined}
    </DescriptionCard>
  )
}
