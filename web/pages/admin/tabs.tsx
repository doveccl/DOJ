import { CloudUploadOutlined, DeleteOutlined, DownloadOutlined, EditOutlined, KeyOutlined, PlusOutlined, SearchOutlined } from '@ant-design/icons'
import { Button, Form, Input, InputNumber, Popconfirm, Select, Space, Statistic, Switch, Table, Tag, Tooltip, Typography } from 'antd'
import type { FormInstance } from 'antd'
import type { ReactNode } from 'react'

import type { AdminGroupPage, AdminJudgers, AdminLang, AdminMembers, AdminSettings, AdminUserPage, BackupItem, BackupList, BackupSettings } from '../../client'
import { UserLink } from '../../components/entity'
import { ErrorBlock, LoadingBlock } from '../../components/state'
import { useLocale } from '../../locale'
import { formatBytes, formatDuration } from '../../utils/format'
import { limits } from '../../utils/limits'
import type { BackupSettingsForm, GroupRow, JudgerRow, LanguageRow, SettingsForm, UserRow } from './types'

type Option<T extends string | number> = { value: T; label: string }

export function SettingsTab({
  block,
  form,
  data,
  pending,
  saveSiteName,
  savePatch
}: {
  block: ReactNode
  form: FormInstance<SettingsForm>
  data?: AdminSettings
  pending: boolean
  saveSiteName: (value: string) => void
  savePatch: (patch: Partial<SettingsForm>) => void
}) {
  const text = useAdminText()
  return block ?? (
    <Form<SettingsForm>
      form={form}
      layout="vertical"
      style={{ maxWidth: 680 }}
      initialValues={data}
      key={`${data?.siteName}:${data?.allowRegistration}:${data?.allowGuestAccess}:${data?.defaultSubmissionPublic}`}
    >
      <Form.Item name="siteName" label={text.admin.siteName} rules={[{ required: true }]}>
        <Input maxLength={limits.name} showCount disabled={pending} onBlur={(event) => saveSiteName(event.target.value)} onPressEnter={(event) => event.currentTarget.blur()} />
      </Form.Item>
      <Form.Item name="allowRegistration" label={text.admin.allowRegistration} valuePropName="checked">
        <Switch loading={pending} onChange={(checked) => savePatch({ allowRegistration: checked })} />
      </Form.Item>
      <Form.Item name="allowGuestAccess" label={text.admin.allowGuestAccess} valuePropName="checked">
        <Switch loading={pending} onChange={(checked) => savePatch({ allowGuestAccess: checked })} />
      </Form.Item>
      <Form.Item name="defaultSubmissionPublic" label={text.admin.defaultSubmissionPublic} valuePropName="checked">
        <Switch loading={pending} onChange={(checked) => savePatch({ defaultSubmissionPublic: checked })} />
      </Form.Item>
    </Form>
  )
}

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
  const text = useAdminText()
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
        pagination={{ current: data?.page ?? page, pageSize: data?.pageSize ?? pageSize, total: data?.total ?? 0, showSizeChanger: true }}
        dataSource={data?.items ?? []}
        onChange={(pagination) => {
          setPage(pagination.current ?? page)
          setPageSize(pagination.pageSize ?? pageSize)
        }}
        columns={[
          { title: text.rank.user, dataIndex: 'name', render: (name: string) => <UserLink name={name} /> },
          { title: text.profile.email, dataIndex: 'mail' },
          { title: text.admin.role, dataIndex: 'role', render: (role: string) => <Tag color={role === 'admin' ? 'blue' : undefined}>{roleText[role] ?? role}</Tag> },
          { title: text.admin.groupCount, dataIndex: 'groups', render: (groups: number[] | undefined) => <Typography.Text>{groups?.length ?? 0}</Typography.Text> },
          {
            title: text.common.actions,
            render: (_, row) => (
              <Space size={4}>
                <Tooltip title={text.common.edit}>
                  <Button type="text" icon={<EditOutlined />} onClick={() => onEdit(row)} />
                </Tooltip>
                <Tooltip title={text.admin.resetPassword}>
                  <Button type="text" icon={<KeyOutlined />} loading={resetLoadingName === row.name} onClick={() => onResetPassword(row.name)} />
                </Tooltip>
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
  block: ReactNode
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
        pagination={{ current: data?.page ?? page, pageSize: data?.pageSize ?? pageSize, total: data?.total ?? 0, showSizeChanger: true }}
        dataSource={data?.items ?? []}
        onChange={(pagination) => {
          setPage(pagination.current ?? page)
          setPageSize(pagination.pageSize ?? pageSize)
        }}
        columns={[
          { title: text.admin.groups, dataIndex: 'name' },
          { title: text.admin.userCount, render: (_, row) => <Typography.Text>{row.users?.length ?? 0}</Typography.Text> },
          {
            title: text.common.actions,
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

export function LanguagesTab({
  block,
  data,
  onAdd,
  onEdit,
  onDelete
}: {
  block: ReactNode
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
        pagination={false}
        dataSource={data ?? []}
        columns={[
          { title: text.admin.name, dataIndex: 'name' },
          { title: text.admin.source, dataIndex: 'source' },
          { title: text.admin.image, dataIndex: 'image', ellipsis: true },
          {
            title: text.admin.run,
            dataIndex: 'run',
            ellipsis: true,
            render: (value: string) => {
              const firstLine = value.split('\n')[0]
              return <Typography.Text ellipsis={{ tooltip: firstLine }} className="lineText">{firstLine}</Typography.Text>
            }
          },
          {
            title: text.common.actions,
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
  lang: string
  onAdd: () => void
  onEdit: (row: JudgerRow) => void
  onDelete: (id: number) => void
}) {
  const text = useAdminText()
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
        pagination={false}
        dataSource={data?.judgers ?? []}
        columns={[
          { title: text.admin.name, dataIndex: 'name' },
          { title: text.admin.status, dataIndex: 'online', render: (online: boolean) => (online ? <Tag color="success">{text.admin.online}</Tag> : <Tag>{text.admin.offline}</Tag>) },
          { title: text.admin.uptime, dataIndex: 'uptimeSeconds', render: (value: number, row) => (row.online ? formatDuration(value, lang) : '-') },
          {
            title: text.common.actions,
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

export function BackupsTab({
  settings,
  backups,
  frequencyOptions,
  frequencyText,
  createLoading,
  settingsSaveLoading,
  downloadName,
  deleteName,
  onSaveSettings,
  onCreate,
  onDownload,
  onDelete,
  lang
}: {
  settings: { isLoading: boolean; isError: boolean; error: unknown; data?: BackupSettings }
  backups: { isLoading: boolean; isError: boolean; error: unknown; data?: BackupList }
  frequencyOptions: Option<string>[]
  frequencyText: string
  createLoading: boolean
  settingsSaveLoading: boolean
  downloadName?: string
  deleteName?: string
  onSaveSettings: (values: BackupSettingsForm) => void
  onCreate: () => void
  onDownload: (name: string) => void
  onDelete: (name: string) => void
  lang: string
}) {
  const text = useAdminText()
  return (
    <div className="adminBackupPage">
      <div className="adminBackupToolbar">
        <Space className="adminBackupStatus" wrap>
          <Typography.Text strong>{text.admin.backupSchedule}</Typography.Text>
          {settings.data?.enabled ? <Tag color="success">{frequencyText}</Tag> : <Tag>{text.admin.backupDisabled}</Tag>}
          {settings.data?.enabled ? <Typography.Text type="secondary">{settings.data.time}</Typography.Text> : null}
          {backups.data?.running ? <Tag color={backups.data.running.stale ? 'warning' : 'processing'}>{backups.data.running.stale ? text.admin.backupStale : text.admin.backupRunning}</Tag> : <Tag>{text.admin.backupReady}</Tag>}
        </Space>
        <Button type="primary" icon={<CloudUploadOutlined />} loading={createLoading || !!backups.data?.running} onClick={onCreate}>
          {text.admin.backupNow}
        </Button>
      </div>
      <div className="adminBackupSettings">
        {settings.isLoading ? <LoadingBlock /> : settings.isError ? <ErrorBlock error={settings.error} /> : settings.data ? (
          <Form<BackupSettingsForm>
            className="adminBackupForm"
            layout="vertical"
            initialValues={settings.data}
            key={`${settings.data.enabled}:${settings.data.frequency}:${settings.data.keep}:${settings.data.time}`}
            onFinish={onSaveSettings}
          >
            <div className="adminBackupFormGrid">
              <Form.Item name="enabled" label={text.admin.backupEnabled} valuePropName="checked">
                <Switch />
              </Form.Item>
              <Form.Item name="frequency" label={text.admin.backupFrequency} rules={[{ required: true }]}>
                <Select options={frequencyOptions} />
              </Form.Item>
              <Form.Item name="time" label={text.admin.backupTime} rules={[{ required: true }]}>
                <Input placeholder="03:00" maxLength={5} />
              </Form.Item>
              <Form.Item name="keep" label={text.admin.backupKeep} rules={[{ required: true }]}>
                <InputNumber min={1} max={100} style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item className="adminBackupSubmit">
                <Button type="primary" htmlType="submit" loading={settingsSaveLoading}>{text.common.save}</Button>
              </Form.Item>
            </div>
          </Form>
        ) : null}
      </div>
      <div className="adminBackupTableHead">
        <Typography.Title level={5}>{text.admin.backupFiles}</Typography.Title>
        <Typography.Text type="secondary">{text.admin.backupCount(backups.data?.items.length ?? 0)}</Typography.Text>
      </div>
      {backups.isLoading ? <LoadingBlock /> : backups.isError ? <ErrorBlock error={backups.error} /> : (
        <Table<BackupItem>
          rowKey="name"
          pagination={{ defaultPageSize: 10, hideOnSinglePage: true }}
          scroll={{ x: 760 }}
          dataSource={backups.data?.items ?? []}
          columns={[
            { title: text.admin.backupFile, dataIndex: 'name', width: 320, render: (name: string) => <Typography.Text code ellipsis={{ tooltip: name }} className="backupFileName">{name}</Typography.Text> },
            { title: text.admin.backupDatabase, dataIndex: 'database', width: 120 },
            { title: text.admin.createdAt, dataIndex: 'createdAt', width: 220, render: (value: string) => new Date(value).toLocaleString(lang === 'zh' ? 'zh-CN' : 'en-US') },
            { title: text.admin.backupSize, dataIndex: 'size', width: 100, render: (value: number) => formatBytes(value) },
            {
              title: text.common.actions,
              width: 120,
              render: (_, row) => (
                <Space size={4}>
                  <Tooltip title={text.common.download}>
                    <Button type="text" icon={<DownloadOutlined />} loading={downloadName === row.name} onClick={() => onDownload(row.name)} />
                  </Tooltip>
                  <Popconfirm title={text.common.confirmDelete} okText={text.common.delete} cancelText={text.common.cancel} onConfirm={() => onDelete(row.name)}>
                    <Button type="text" danger icon={<DeleteOutlined />} loading={deleteName === row.name} />
                  </Popconfirm>
                </Space>
              )
            }
          ]}
        />
      )}
    </div>
  )
}

function useAdminText() {
  return useLocale().text
}
