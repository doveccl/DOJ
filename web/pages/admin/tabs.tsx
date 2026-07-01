import { CloudUploadOutlined, DeleteOutlined, DownloadOutlined, EditOutlined, KeyOutlined, PlusOutlined, SearchOutlined } from '@ant-design/icons'
import { AutoComplete, Button, Flex, Form, Input, InputNumber, Popconfirm, Space, Statistic, Switch, Table, Tag, Tooltip, Typography } from 'antd'
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
        scroll={{ x: 760 }}
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
        scroll={{ x: 520 }}
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
        scroll={{ x: 680 }}
        pagination={false}
        dataSource={data?.judgers ?? []}
        columns={[
          { title: text.admin.name, dataIndex: 'name' },
          { title: text.admin.status, dataIndex: 'online', render: (online: boolean) => (online ? <Tag color="success">{text.admin.online}</Tag> : <Tag>{text.admin.offline}</Tag>) },
          { title: text.admin.uptime, dataIndex: 'uptimeSeconds', render: (value: number, row) => (row.online ? formatDuration(value, lang) : '-') },
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

export function BackupsTab({
  settings,
  backups,
  cronOptions,
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
  cronOptions: Option<string>[]
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
  const [form] = Form.useForm<BackupSettingsForm>()
  const saveSettings = (patch?: Partial<BackupSettingsForm>) => {
    if (!settings.data) {
      return
    }
    const next = { ...settings.data, ...form.getFieldsValue(true), ...patch }
    if (next.enabled === settings.data.enabled && next.cron === settings.data.cron && next.keep === settings.data.keep) {
      return
    }
    onSaveSettings(next)
  }
  return (
    <Flex vertical gap={16} className="adminBackupPage">
      <Flex className="tableToolbar" justify="space-between" align="center" gap={12} wrap>
        {settings.isLoading ? <LoadingBlock /> : settings.isError ? <ErrorBlock error={settings.error} /> : settings.data ? (
          <Form<BackupSettingsForm>
            form={form}
            className="tableToolbarForm"
            layout="inline"
            initialValues={settings.data}
            key={`${settings.data.enabled}:${settings.data.cron}:${settings.data.keep}`}
          >
            <Form.Item name="enabled" label={text.admin.backupEnabled} valuePropName="checked">
              <Switch loading={settingsSaveLoading} onChange={(enabled) => saveSettings({ enabled })} />
            </Form.Item>
            <Form.Item name="cron">
              <AutoComplete options={cronOptions} placeholder={text.admin.backupCron} disabled={settingsSaveLoading} style={{ width: 220 }} onBlur={() => saveSettings()} />
            </Form.Item>
            <Form.Item name="keep">
              <InputNumber min={1} max={100} prefix={text.admin.backupKeep} disabled={settingsSaveLoading} style={{ width: 180 }} onBlur={() => saveSettings()} />
            </Form.Item>
          </Form>
        ) : null}
        <Button type="primary" icon={<CloudUploadOutlined />} loading={createLoading || !!backups.data?.running} onClick={onCreate}>
          {text.admin.backupNow}
        </Button>
      </Flex>
      {backups.isLoading ? <LoadingBlock /> : backups.isError ? <ErrorBlock error={backups.error} /> : (
        <Table<BackupItem>
          rowKey="name"
          pagination={{ defaultPageSize: 10, hideOnSinglePage: true }}
          scroll={{ x: 760 }}
          dataSource={backups.data?.items ?? []}
          columns={[
            { title: text.admin.backupFile, dataIndex: 'name', width: 360, render: (name: string) => <Typography.Text ellipsis={{ tooltip: name }}>{name}</Typography.Text> },
            { title: text.admin.createdAt, dataIndex: 'createdAt', render: (value: string) => new Date(value).toLocaleString(lang === 'zh' ? 'zh-CN' : 'en-US') },
            { title: text.admin.backupSize, dataIndex: 'size', render: (value: number) => formatBytes(value) },
            {
              align: 'right',
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
    </Flex>
  )
}

function useAdminText() {
  return useLocale().text
}
