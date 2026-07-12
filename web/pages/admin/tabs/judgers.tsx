import type { ReactNode } from 'react'

import { DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons'
import { Button, Popconfirm, Space, Statistic, Table, Tag, Tooltip, Typography } from 'antd'

import type { AdminJudgers } from '../../../client'
import { useLocale } from '../../../locale'
import type { Lang } from '../../../locale'
import { formatDuration } from '../../../utils/format'
import type { JudgerRow } from '../types'

export function JudgersTab({
  block,
  data,
  lang,
  onAdd,
  onEdit,
  onDelete
}: {
  block: ReactNode
  data?: AdminJudgers
  lang: Lang
  onAdd: () => void
  onEdit: (row: JudgerRow) => void
  onDelete: (id: number) => void
}) {
  const text = useLocale().text
  return block ?? (
    <Space orientation="vertical" size={16} style={{ width: '100%' }}>
      <Space size={24} wrap>
        <Statistic title={text.admin.queued} value={data?.queue.queued ?? 0} />
        <Statistic title={text.admin.running} value={data?.queue.running ?? 0} />
        <Statistic title={text.admin.done} value={data?.queue.done ?? 0} />
      </Space>
      <Button type="primary" icon={<PlusOutlined />} onClick={onAdd}>
        {text.admin.addJudger}
      </Button>
      <Table<JudgerRow>
        rowKey="id"
        scroll={{ x: 680 }}
        pagination={false}
        dataSource={data?.judgers ?? []}
        columns={[
          { title: text.admin.name, dataIndex: 'name', width: 280, ellipsis: { showTitle: false }, render: (name: string) => <Typography.Text ellipsis={{ tooltip: name }}>{name}</Typography.Text> },
          { title: text.admin.status, dataIndex: 'online', render: (online: boolean) => (online ? <Tag color="success">{text.admin.online}</Tag> : <Tag>{text.admin.offline}</Tag>) },
          { title: text.admin.uptime, dataIndex: 'uptimeSeconds', render: (value: number, row) => (row.online ? formatDuration(value, lang) : '-') },
          {
            title: text.common.actions,
            align: 'right',
            render: (_, row) => (
              <Space size={4}>
                <Tooltip title={text.common.edit}>
                  <Button type="text" icon={<EditOutlined />} onClick={() => onEdit(row)} />
                </Tooltip>
                <Popconfirm title={text.common.confirmDelete} okText={text.common.delete} cancelText={text.common.cancel} onConfirm={() => onDelete(row.id)}>
                  <Button type="text" danger icon={<DeleteOutlined />} />
                </Popconfirm>
              </Space>
            )
          }
        ]}
      />
    </Space>
  )
}
