import { DeleteOutlined, EditOutlined, PlusOutlined, SearchOutlined } from '@ant-design/icons'
import { Button, Input, Popconfirm, Space, Table, Tooltip, Typography } from 'antd'

import type { AdminGroupPage } from '../../../client'
import type { GroupRow } from '../types'
import type { Block } from './shared'
import { useAdminText } from './shared'

export function GroupsTab({
  block,
  data,
  search,
  setSearch,
  page,
  pageSize,
  setPage,
  setPageSize,
  onAdd,
  onEdit,
  onDelete
}: {
  block: Block
  data?: AdminGroupPage
  search: string
  setSearch: (value: string) => void
  page: number
  pageSize: number
  setPage: (value: number) => void
  setPageSize: (value: number) => void
  onAdd: () => void
  onEdit: (row: GroupRow) => void
  onDelete: (id: number) => void
}) {
  const text = useAdminText()
  return block ?? (
    <Space orientation="vertical" size={16} style={{ width: '100%' }}>
      <Space style={{ width: '100%', justifyContent: 'space-between' }} wrap>
        <Input
          allowClear
          prefix={<SearchOutlined />}
          placeholder={text.admin.searchGroups}
          value={search}
          onChange={(event) => {
            setSearch(event.target.value)
            setPage(1)
          }}
        />
        <Button type="primary" icon={<PlusOutlined />} onClick={onAdd}>
          {text.admin.addGroup}
        </Button>
      </Space>
      <Table<GroupRow>
        rowKey="id"
        scroll={{ x: 520 }}
        pagination={{ current: data?.page ?? page, pageSize: data?.pageSize ?? pageSize, total: data?.total ?? 0, showSizeChanger: true }}
        dataSource={data?.items ?? []}
        onChange={(pagination) => {
          setPage(pagination.current ?? page)
          setPageSize(pagination.pageSize ?? pageSize)
        }}
        columns={[
          { title: text.admin.groups, dataIndex: 'name', width: 360, ellipsis: { showTitle: false }, render: (name: string) => <Typography.Text ellipsis={{ tooltip: name }}>{name}</Typography.Text> },
          { title: text.admin.userCount, render: (_, row) => <Typography.Text>{row.users?.length ?? 0}</Typography.Text> },
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
