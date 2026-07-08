import { DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons'
import { Button, Popconfirm, Space, Table, Tooltip, Typography } from 'antd'

import type { AdminLang } from '../../../client'
import type { LanguageRow } from '../types'
import type { Block } from './shared'
import { useAdminText } from './shared'

export function LanguagesTab({
  block,
  data,
  onAdd,
  onEdit,
  onDelete
}: {
  block: Block
  data?: AdminLang[]
  onAdd: () => void
  onEdit: (row: LanguageRow) => void
  onDelete: (id: string) => void
}) {
  const text = useAdminText()
  return block ?? (
    <Space orientation="vertical" size={16} style={{ width: '100%' }}>
      <Button type="primary" icon={<PlusOutlined />} onClick={onAdd}>
        {text.admin.addLang}
      </Button>
      <Table<LanguageRow>
        rowKey="id"
        scroll={{ x: 980 }}
        pagination={false}
        dataSource={data ?? []}
        columns={[
          { title: text.admin.name, dataIndex: 'name' },
          { title: text.admin.source, dataIndex: 'source' },
          {
            title: text.admin.image,
            dataIndex: 'image',
            width: 220,
            ellipsis: { showTitle: false },
            render: (image: string) => <Typography.Text ellipsis={{ tooltip: image }}>{image}</Typography.Text>
          },
          {
            title: text.admin.run,
            dataIndex: 'run',
            width: 320,
            ellipsis: { showTitle: false },
            render: (value: string) => {
              const firstLine = value.split('\n')[0]
              return <Typography.Text ellipsis={{ tooltip: firstLine }}>{firstLine}</Typography.Text>
            }
          },
          {
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
