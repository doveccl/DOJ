import type { ReactNode } from 'react'

import { DeleteOutlined, EditOutlined, KeyOutlined, PlusOutlined, SearchOutlined } from '@ant-design/icons'
import { Button, Input, Popconfirm, Space, Table, Tag, Tooltip, Typography } from 'antd'

import type { AdminUserPage } from '../../../client'
import { UserLink } from '../../../components/entity'
import { useLocale } from '../../../locale'
import type { UserRow } from '../types'

export function UsersTab({
  block,
  data,
  search,
  setSearch,
  page,
  pageSize,
  setPage,
  setPageSize,
  roleText,
  onAdd,
  onEdit,
  onResetPassword,
  resetLoadingName,
  onDelete
}: {
  block: ReactNode
  data?: AdminUserPage
  search: string
  setSearch: (value: string) => void
  page: number
  pageSize: number
  setPage: (value: number) => void
  setPageSize: (value: number) => void
  roleText: Record<string, string>
  onAdd: () => void
  onEdit: (row: UserRow) => void
  onResetPassword: (name: string) => void
  resetLoadingName?: string
  onDelete: (name: string) => void
}) {
  const text = useLocale().text
  return block ?? (
    <Space orientation="vertical" size={16} style={{ width: '100%' }}>
      <Space style={{ width: '100%', justifyContent: 'space-between' }} wrap>
        <Input
          allowClear
          prefix={<SearchOutlined />}
          placeholder={text.admin.searchUsers}
          value={search}
          onChange={(event) => {
            setSearch(event.target.value)
            setPage(1)
          }}
        />
        <Button type="primary" icon={<PlusOutlined />} onClick={onAdd}>
          {text.admin.addUser}
        </Button>
      </Space>
      <Table<UserRow>
        rowKey="name"
        scroll={{ x: 760 }}
        pagination={{ current: data?.page ?? page, pageSize: data?.pageSize ?? pageSize, total: data?.total ?? 0, showSizeChanger: true }}
        dataSource={data?.items ?? []}
        onChange={(pagination) => {
          setPage(pagination.current ?? page)
          setPageSize(pagination.pageSize ?? pageSize)
        }}
        columns={[
          { title: text.rank.user, dataIndex: 'name', width: 220, render: (_, row) => <UserLink name={row.name} avatar={row.avatar} /> },
          { title: text.profile.email, dataIndex: 'mail', width: 300, ellipsis: { showTitle: false }, render: (mail: string) => <Typography.Text ellipsis={{ tooltip: mail }}>{mail}</Typography.Text> },
          { title: text.admin.role, dataIndex: 'role', render: (role: string) => <Tag color={role === 'admin' ? 'blue' : undefined}>{roleText[role] ?? role}</Tag> },
          { title: text.admin.groupCount, dataIndex: 'groups', render: (groups: number[] | undefined) => <Typography.Text>{groups?.length ?? 0}</Typography.Text> },
          {
            title: text.common.actions,
            align: 'right',
            render: (_, row) => (
              <Space size={4}>
                <Tooltip title={text.common.edit}>
                  <Button type="text" icon={<EditOutlined />} onClick={() => onEdit(row)} />
                </Tooltip>
                <Popconfirm title={text.admin.resetPassword} okText={text.admin.resetPassword} cancelText={text.common.cancel} onConfirm={() => onResetPassword(row.name)}>
                  <Button type="text" icon={<KeyOutlined />} loading={resetLoadingName === row.name} />
                </Popconfirm>
                <Popconfirm title={text.common.confirmDelete} okText={text.common.delete} cancelText={text.common.cancel} onConfirm={() => onDelete(row.name)}>
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
