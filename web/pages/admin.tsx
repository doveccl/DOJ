import { CloudUploadOutlined, DeleteOutlined, DownloadOutlined, EditOutlined, KeyOutlined, PlusOutlined, SearchOutlined } from '@ant-design/icons'
import { App as AntApp, Button, Card, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Statistic, Switch, Table, Tabs, Tag, Tooltip, Typography } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import {
  createAdminUser,
  createBackup,
  createAdminGroup,
  createAdminJudger,
  createAdminLang,
  deleteBackup,
  deleteAdminGroup,
  deleteAdminJudger,
  deleteAdminLang,
  deleteAdminUser,
  downloadBackup,
  getAdminJudgers,
  getAdminLangs,
  getAdminMembers,
  getAdminSettings,
  getBackups,
  getBackupSettings,
  resetAdminUserPassword,
  updateBackupSettings,
  updateAdminGroup,
  updateAdminJudger,
  updateAdminLang,
  updateAdminUser,
  updateAdminSettings
} from '../client'
import type { AdminGroupUpdate, AdminJudgers, AdminLang, AdminLangCreate, AdminMembers, AdminSettings, AdminUserCreate, BackupItem, BackupSettings } from '../client'
import { UserLink } from '../components/entity'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { IdSelect } from '../components/id-select'
import { useLocale } from '../locale'
import { useSession } from '../session'
import { formatBytes, formatDuration } from '../utils/format'
import { limits } from '../utils/limits'

type UserRow = AdminMembers['users'][number]
type GroupRow = AdminMembers['groups'][number]
type LanguageRow = AdminLang
type JudgerRow = AdminJudgers['judgers'][number]
type JudgerForm = { name: string; auth?: string }
type UserForm = AdminUserCreate
type UserEditForm = Pick<AdminUserCreate, 'role' | 'groups'>
type GroupForm = AdminGroupUpdate
type SettingsForm = Pick<AdminSettings, 'siteName' | 'allowRegistration' | 'allowGuestAccess' | 'defaultSubmissionPublic'>
type BackupSettingsForm = BackupSettings

const defaultLanguage = {
  id: '',
  name: '',
  source: 'main.cc',
  image: 'gcc:14',
  compile: 'g++ -std=c++20 -O2 -pipe -static -s main.cc -o /work/main',
  run: './main'
}

export function AdminPage() {
  const { lang, text } = useLocale()
  const session = useSession()
  const { message, modal } = AntApp.useApp()
  const client = useQueryClient()
  const [groupOpen, setGroupOpen] = useState(false)
  const [userOpen, setUserOpen] = useState(false)
  const [langOpen, setLangOpen] = useState(false)
  const [judgerOpen, setJudgerOpen] = useState(false)
  const [editingGroup, setEditingGroup] = useState<GroupRow | null>(null)
  const [editingUser, setEditingUser] = useState<UserRow | null>(null)
  const [editingLang, setEditingLang] = useState<LanguageRow | null>(null)
  const [editingJudger, setEditingJudger] = useState<JudgerRow | null>(null)
  const [userSearch, setUserSearch] = useState('')
  const [groupSearch, setGroupSearch] = useState('')
  const [activeTab, setActiveTab] = useState('settings')
  const [settingsForm] = Form.useForm<SettingsForm>()
  const membersEnabled = session.admin && (activeTab === 'users' || activeTab === 'groups')
  const languagesEnabled = session.admin && activeTab === 'languages'
  const judgersEnabled = session.admin && activeTab === 'judgers'
  const backupEnabled = session.admin && activeTab === 'backups'
  const membersQuery = useQuery({ queryKey: ['admin-members'], queryFn: () => getAdminMembers(), enabled: membersEnabled })
  const languagesQuery = useQuery({ queryKey: ['admin-languages'], queryFn: getAdminLangs, enabled: languagesEnabled })
  const judgersQuery = useQuery({ queryKey: ['admin-judgers'], queryFn: getAdminJudgers, enabled: judgersEnabled })
  const settingsQuery = useQuery({ queryKey: ['admin-settings'], queryFn: getAdminSettings, enabled: session.admin })
  const backupSettings = useQuery({ queryKey: ['backup-settings'], queryFn: getBackupSettings, enabled: backupEnabled })
  const backups = useQuery({ queryKey: ['backups'], queryFn: getBackups, enabled: backupEnabled, refetchInterval: (query) => (query.state.data?.running ? 5000 : false) })
  const showError = (error: unknown) => {
    message.error(error instanceof Error ? error.message : text.common.loadingFailed)
  }
  const saveMembers = (data: AdminMembers) => {
    client.setQueryData<AdminMembers>(['admin-members'], data)
    message.success(text.common.saved)
  }
  const saveLanguages = (data: AdminLang[]) => {
    client.setQueryData<AdminLang[]>(['admin-languages'], data)
    message.success(text.common.saved)
  }
  const saveJudgers = (data: AdminJudgers) => {
    client.setQueryData<AdminJudgers>(['admin-judgers'], data)
    message.success(text.common.saved)
  }
  const settings = useMutation({
    mutationFn: updateAdminSettings,
    onSuccess: (data) => {
      client.setQueryData(['admin-settings'], data)
      client.setQueryData(['site'], data)
      message.success(text.common.saved)
    },
    onError: showError
  })
  const userSave = useMutation({
    mutationFn: ({ name, role, groups }: { name: string; role: string; groups: number[] }) => updateAdminUser(name, { role, groups }),
    onSuccess: (data) => {
      saveMembers(data)
      setEditingUser(null)
    },
    onError: showError
  })
  const userCreate = useMutation({
    mutationFn: createAdminUser,
    onSuccess: (data) => {
      saveMembers(data)
      setUserOpen(false)
    },
    onError: showError
  })
  const userDelete = useMutation({
    mutationFn: deleteAdminUser,
    onSuccess: saveMembers,
    onError: showError
  })
  const userPassword = useMutation({
    mutationFn: resetAdminUserPassword,
    onSuccess: (data) => {
      modal.info({
        title: text.admin.resetPassword,
        content: <Typography.Paragraph copyable>{data.password}</Typography.Paragraph>
      })
    },
    onError: showError
  })
  const groupSave = useMutation({
    mutationFn: (values: AdminGroupUpdate) => (editingGroup ? updateAdminGroup(editingGroup.id, values) : createAdminGroup(values)),
    onSuccess: (data) => {
      saveMembers(data)
      closeGroup()
    },
    onError: showError
  })
  const groupDelete = useMutation({
    mutationFn: deleteAdminGroup,
    onSuccess: saveMembers,
    onError: showError
  })
  const langSave = useMutation({
    mutationFn: (values: AdminLangCreate) => (editingLang ? updateAdminLang(editingLang.id, values) : createAdminLang(values)),
    onSuccess: (data) => {
      saveLanguages(data)
      closeLang()
    },
    onError: showError
  })
  const langDelete = useMutation({
    mutationFn: deleteAdminLang,
    onSuccess: saveLanguages,
    onError: showError
  })
  const judgerSave = useMutation({
    mutationFn: (values: JudgerForm) =>
      editingJudger ? updateAdminJudger(editingJudger.id, { name: values.name, auth: values.auth || undefined }) : createAdminJudger({ name: values.name }),
    onSuccess: (data) => {
      const created = data.judgers.find((row) => row.token)
      saveJudgers(data)
      closeJudger()
      if (created?.token) {
        const command = `SERVER=http://server:7974 TOKEN=${created.token} doj-judger`
        modal.info({
          title: text.admin.judgerTokenCreated,
          content: (
            <Space orientation="vertical" style={{ width: '100%' }}>
              <Typography.Text type="secondary">{text.admin.judgerTokenHelp}</Typography.Text>
              <Typography.Paragraph copyable code>
                {created.token}
              </Typography.Paragraph>
              <Typography.Paragraph copyable={{ text: command }} code>
                {command}
              </Typography.Paragraph>
            </Space>
          )
        })
      }
    },
    onError: showError
  })
  const judgerDelete = useMutation({
    mutationFn: deleteAdminJudger,
    onSuccess: saveJudgers,
    onError: showError
  })
  const backupSettingsSave = useMutation({
    mutationFn: updateBackupSettings,
    onSuccess: (data) => {
      client.setQueryData(['backup-settings'], data)
      message.success(text.common.saved)
    },
    onError: showError
  })
  const backupCreate = useMutation({
    mutationFn: createBackup,
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['backups'] })
      message.success(text.admin.backupManualDone)
    },
    onError: showError
  })
  const backupDelete = useMutation({
    mutationFn: deleteBackup,
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['backups'] })
    },
    onError: showError
  })
  const backupDownload = useMutation({
    mutationFn: downloadBackup,
    onSuccess: (blob, name) => saveBlob(blob, name),
    onError: showError
  })

  if (!session.admin) {
    return <ErrorBlock error={text.common.forbidden} />
  }

  const membersData = membersQuery.data
  const languagesData = languagesQuery.data
  const judgersData = judgersQuery.data
  const settingsData = settingsQuery.data
  const roleText: Record<string, string> = text.admin.roles
  const roleOptions = [
    { value: 'user', label: roleText.user },
    { value: 'admin', label: roleText.admin }
  ]
  const groupOptions = (membersData?.groups ?? []).map((group) => ({ value: group.id, label: group.name }))
  const userOptions = (membersData?.users ?? []).map((user) => ({ value: user.id, label: user.name }))
  const userGroupIds = (row: UserRow) => row.groups ?? []
  const groupUserIds = (row: GroupRow) => (membersData?.users ?? []).filter((user) => userGroupIds(user).includes(row.id)).map((user) => user.id)
  const userQuery = userSearch.trim().toLowerCase()
  const groupQuery = groupSearch.trim().toLowerCase()
  const groupName = (id: number) => membersData?.groups.find((group) => group.id === id)?.name ?? ''
  const backupFrequencyOptions = [
    { value: 'hourly', label: text.admin.backupFrequencies.hourly },
    { value: 'daily', label: text.admin.backupFrequencies.daily },
    { value: 'weekly', label: text.admin.backupFrequencies.weekly }
  ]
  const backupFrequencyText = backupSettings.data
    ? (backupFrequencyOptions.find((option) => option.value === backupSettings.data.frequency)?.label ?? backupSettings.data.frequency)
    : ''
  const saveSettingsPatch = (patch: Partial<SettingsForm>) => {
    settings.mutate(patch)
  }
  const saveSiteName = (value: string) => {
    const current = settingsData
    if (!current) {
      return
    }
    const siteName = value.trim()
    if (!siteName) {
      settingsForm.setFieldValue('siteName', current.siteName)
      return
    }
    if (siteName === current.siteName) {
      settingsForm.setFieldValue('siteName', current.siteName)
      return
    }
    saveSettingsPatch({ siteName })
  }
  const filteredUsers = userQuery
    ? (membersData?.users ?? []).filter((user) =>
        [user.name, user.mail, roleText[user.role] ?? user.role, ...userGroupIds(user).map(groupName)].some((value) => value.toLowerCase().includes(userQuery))
      )
    : (membersData?.users ?? [])
  const filteredGroups = groupQuery
    ? (membersData?.groups ?? []).filter((group) =>
        [group.name, ...groupUserIds(group).map((id) => membersData?.users.find((user) => user.id === id)?.name ?? '')].some((value) =>
          value.toLowerCase().includes(groupQuery)
        )
      )
    : (membersData?.groups ?? [])
  const membersBlock = membersQuery.isLoading ? (
    <LoadingBlock />
  ) : membersQuery.isError ? (
    <ErrorBlock error={membersQuery.error} />
  ) : membersEnabled && !membersData ? (
    <LoadingBlock />
  ) : null
  const languagesBlock = languagesQuery.isLoading ? (
    <LoadingBlock />
  ) : languagesQuery.isError ? (
    <ErrorBlock error={languagesQuery.error} />
  ) : languagesEnabled && !languagesData ? (
    <LoadingBlock />
  ) : null
  const judgersBlock = judgersQuery.isLoading ? (
    <LoadingBlock />
  ) : judgersQuery.isError ? (
    <ErrorBlock error={judgersQuery.error} />
  ) : judgersEnabled && !judgersData ? (
    <LoadingBlock />
  ) : null
  const settingsBlock = settingsQuery.isLoading ? (
    <LoadingBlock />
  ) : settingsQuery.isError ? (
    <ErrorBlock error={settingsQuery.error} />
  ) : !settingsData ? (
    <ErrorBlock error={text.common.emptyResponse} />
  ) : null

  function openGroup(row?: GroupRow) {
    setEditingGroup(row ?? null)
    setGroupOpen(true)
  }

  function closeGroup() {
    setGroupOpen(false)
    setEditingGroup(null)
  }

  function openLang(row?: LanguageRow) {
    setEditingLang(row ?? null)
    setLangOpen(true)
  }

  function closeLang() {
    setLangOpen(false)
    setEditingLang(null)
  }

  function openJudger(row?: JudgerRow) {
    setEditingJudger(row ?? null)
    setJudgerOpen(true)
  }

  function closeJudger() {
    setJudgerOpen(false)
    setEditingJudger(null)
  }

  return (
    <>
      <Card>
        <Tabs
          activeKey={activeTab}
          destroyOnHidden
          onChange={setActiveTab}
          items={[
            {
              key: 'settings',
              label: text.admin.settings,
              children: settingsBlock ?? (
                <Form<SettingsForm>
                  form={settingsForm}
                  layout="vertical"
                  style={{ maxWidth: 680 }}
                  initialValues={settingsData}
                  key={`${settingsData?.siteName}:${settingsData?.allowRegistration}:${settingsData?.allowGuestAccess}:${settingsData?.defaultSubmissionPublic}`}
                >
                  <Form.Item name="siteName" label={text.admin.siteName} rules={[{ required: true }]}>
                    <Input
                      maxLength={limits.name}
                      showCount
                      disabled={settings.isPending}
                      onBlur={(event) => saveSiteName(event.target.value)}
                      onPressEnter={(event) => event.currentTarget.blur()}
                    />
                  </Form.Item>
                  <Form.Item name="allowRegistration" label={text.admin.allowRegistration} valuePropName="checked">
                    <Switch loading={settings.isPending} onChange={(checked) => saveSettingsPatch({ allowRegistration: checked })} />
                  </Form.Item>
                  <Form.Item name="allowGuestAccess" label={text.admin.allowGuestAccess} valuePropName="checked">
                    <Switch loading={settings.isPending} onChange={(checked) => saveSettingsPatch({ allowGuestAccess: checked })} />
                  </Form.Item>
                  <Form.Item name="defaultSubmissionPublic" label={text.admin.defaultSubmissionPublic} valuePropName="checked">
                    <Switch loading={settings.isPending} onChange={(checked) => saveSettingsPatch({ defaultSubmissionPublic: checked })} />
                  </Form.Item>
                </Form>
              )
            },
            {
              key: 'users',
              label: text.admin.users,
              children: membersBlock ?? (
                <Space orientation="vertical" size={16} style={{ width: '100%' }}>
                  <Space style={{ width: '100%', justifyContent: 'space-between' }} wrap>
                    <Input allowClear prefix={<SearchOutlined />} placeholder={text.admin.searchUsers} value={userSearch} onChange={(event) => setUserSearch(event.target.value)} />
                    <Button type="primary" icon={<PlusOutlined />} onClick={() => setUserOpen(true)}>
                      {text.admin.addUser}
                    </Button>
                  </Space>
                  <Table<UserRow>
                    rowKey="name"
                    pagination={{ defaultPageSize: 10, hideOnSinglePage: true, showSizeChanger: true }}
                    dataSource={filteredUsers}
                    columns={[
                      { title: text.rank.user, dataIndex: 'name', render: (name: string) => <UserLink name={name} /> },
                      { title: text.profile.email, dataIndex: 'mail' },
                      {
                        title: text.admin.role,
                        dataIndex: 'role',
                        render: (role: string) => <Tag color={role === 'admin' ? 'blue' : undefined}>{roleText[role] ?? role}</Tag>
                      },
                      {
                        title: text.admin.groupCount,
                        dataIndex: 'groups',
                        render: (groups: number[] | undefined) => <Typography.Text>{groups?.length ?? 0}</Typography.Text>
                      },
                      {
                        title: text.common.actions,
                        render: (_, row) => (
                          <Space size={4}>
                            <Tooltip title={text.common.edit}>
                              <Button type="text" icon={<EditOutlined />} onClick={() => setEditingUser(row)} />
                            </Tooltip>
                            <Tooltip title={text.admin.resetPassword}>
                              <Button
                                type="text"
                                icon={<KeyOutlined />}
                                loading={userPassword.isPending && userPassword.variables === row.name}
                                onClick={() => userPassword.mutate(row.name)}
                              />
                            </Tooltip>
                            <Popconfirm
                              title={text.common.confirmDelete}
                              okText={text.common.delete}
                              cancelText={text.common.cancel}
                              onConfirm={() => userDelete.mutate(row.name)}
                            >
                              <Button type="text" danger icon={<DeleteOutlined />} />
                            </Popconfirm>
                          </Space>
                        )
                      }
                    ]}
                  />
                </Space>
              )
            },
            {
              key: 'groups',
              label: text.admin.groups,
              children: membersBlock ?? (
                <Space orientation="vertical" size={16} style={{ width: '100%' }}>
                  <Space style={{ width: '100%', justifyContent: 'space-between' }} wrap>
                    <Input allowClear prefix={<SearchOutlined />} placeholder={text.admin.searchGroups} value={groupSearch} onChange={(event) => setGroupSearch(event.target.value)} />
                    <Button type="primary" icon={<PlusOutlined />} onClick={() => openGroup()}>
                      {text.admin.addGroup}
                    </Button>
                  </Space>
                  <Table<GroupRow>
                    rowKey="id"
                    pagination={{ defaultPageSize: 10, hideOnSinglePage: true, showSizeChanger: true }}
                    dataSource={filteredGroups}
                    columns={[
                      { title: text.admin.groups, dataIndex: 'name' },
                      {
                        title: text.admin.userCount,
                        render: (_, row) => <Typography.Text>{groupUserIds(row).length}</Typography.Text>
                      },
                      {
                        title: text.common.actions,
                        render: (_, row) => (
                          <Space size={4}>
                            <Tooltip title={text.common.edit}>
                              <Button type="text" icon={<EditOutlined />} onClick={() => openGroup(row)} />
                            </Tooltip>
                            <Popconfirm
                              title={text.common.confirmDelete}
                              okText={text.common.delete}
                              cancelText={text.common.cancel}
                              onConfirm={() => groupDelete.mutate(row.id)}
                            >
                              <Button type="text" danger icon={<DeleteOutlined />} />
                            </Popconfirm>
                          </Space>
                        )
                      }
                    ]}
                  />
                </Space>
              )
            },
            {
              key: 'languages',
              label: text.admin.languages,
              children: languagesBlock ?? (
                <Space orientation="vertical" size={16} style={{ width: '100%' }}>
                  <Button type="primary" icon={<PlusOutlined />} onClick={() => openLang()}>
                    {text.admin.addLang}
                  </Button>
                  <Table<LanguageRow>
                    rowKey="id"
                    pagination={false}
                    dataSource={languagesData ?? []}
                    columns={[
                      { title: text.admin.name, dataIndex: 'name' },
                      { title: text.admin.source, dataIndex: 'source' },
                      { title: text.admin.image, dataIndex: 'image', ellipsis: true },
                      {
                        title: text.admin.run,
                        dataIndex: 'run',
                        width: 280,
                        ellipsis: true,
                        render: (value: string) => {
                          const firstLine = value.split('\n')[0]
                          return (
                            <Typography.Text ellipsis={{ tooltip: firstLine }} className="lineText">
                              {firstLine}
                            </Typography.Text>
                          )
                        }
                      },
                      {
                        title: text.common.actions,
                        render: (_, row) => (
                          <Space size={4}>
                            <Tooltip title={text.common.edit}>
                              <Button type="text" icon={<EditOutlined />} onClick={() => openLang(row)} />
                            </Tooltip>
                            <Popconfirm
                              title={text.common.confirmDelete}
                              okText={text.common.delete}
                              cancelText={text.common.cancel}
                              onConfirm={() => langDelete.mutate(row.id)}
                            >
                              <Button type="text" danger icon={<DeleteOutlined />} />
                            </Popconfirm>
                          </Space>
                        )
                      }
                    ]}
                  />
                </Space>
              )
            },
            {
              key: 'judgers',
              label: text.admin.judgers,
              children: judgersBlock ?? (
                <Space orientation="vertical" size={16} style={{ width: '100%' }}>
                  <Space size={24} wrap>
                    <Statistic title={text.admin.queued} value={judgersData?.queue.queued ?? 0} />
                    <Statistic title={text.admin.running} value={judgersData?.queue.running ?? 0} />
                    <Statistic title={text.admin.done} value={judgersData?.queue.done ?? 0} />
                  </Space>
                  <Button type="primary" icon={<PlusOutlined />} onClick={() => openJudger()}>
                    {text.admin.addJudger}
                  </Button>
                  <Table<JudgerRow>
                    rowKey="id"
                    pagination={false}
                    dataSource={judgersData?.judgers ?? []}
                    columns={[
                      { title: text.admin.name, dataIndex: 'name' },
                      {
                        title: text.admin.status,
                        dataIndex: 'online',
                        render: (online: boolean) => (online ? <Tag color="success">{text.admin.online}</Tag> : <Tag>{text.admin.offline}</Tag>)
                      },
                      {
                        title: text.admin.uptime,
                        dataIndex: 'uptimeSeconds',
                        render: (value: number, row) => (row.online ? formatDuration(value, lang) : '-')
                      },
                      {
                        title: text.common.actions,
                        render: (_, row) => (
                          <Space size={4}>
                            <Tooltip title={text.common.edit}>
                              <Button type="text" icon={<EditOutlined />} onClick={() => openJudger(row)} />
                            </Tooltip>
                            <Popconfirm
                              title={text.common.confirmDelete}
                              okText={text.common.delete}
                              cancelText={text.common.cancel}
                              onConfirm={() => judgerDelete.mutate(row.id)}
                            >
                              <Button type="text" danger icon={<DeleteOutlined />} />
                            </Popconfirm>
                          </Space>
                        )
                      }
                    ]}
                  />
                </Space>
              )
            },
            {
              key: 'backups',
              label: text.admin.backups,
              children: (
                <div className="adminBackupPage">
                  <div className="adminBackupToolbar">
                    <Space className="adminBackupStatus" wrap>
                      <Typography.Text strong>{text.admin.backupSchedule}</Typography.Text>
                      {backupSettings.data?.enabled ? <Tag color="success">{backupFrequencyText}</Tag> : <Tag>{text.admin.backupDisabled}</Tag>}
                      {backupSettings.data?.enabled ? <Typography.Text type="secondary">{backupSettings.data.time}</Typography.Text> : null}
                      {backups.data?.running ? (
                        <Tag color={backups.data.running.stale ? 'warning' : 'processing'}>
                          {backups.data.running.stale ? text.admin.backupStale : text.admin.backupRunning}
                        </Tag>
                      ) : (
                        <Tag>{text.admin.backupReady}</Tag>
                      )}
                    </Space>
                    <Button type="primary" icon={<CloudUploadOutlined />} loading={backupCreate.isPending || !!backups.data?.running} onClick={() => backupCreate.mutate()}>
                      {text.admin.backupNow}
                    </Button>
                  </div>
                  <div className="adminBackupSettings">
                    {backupSettings.isLoading ? (
                      <LoadingBlock />
                    ) : backupSettings.isError ? (
                      <ErrorBlock error={backupSettings.error} />
                    ) : backupSettings.data ? (
                      <Form<BackupSettingsForm>
                        className="adminBackupForm"
                        layout="vertical"
                        initialValues={backupSettings.data}
                        key={`${backupSettings.data.enabled}:${backupSettings.data.frequency}:${backupSettings.data.keep}:${backupSettings.data.time}`}
                        onFinish={(values) => backupSettingsSave.mutate(values)}
                      >
                        <div className="adminBackupFormGrid">
                          <Form.Item name="enabled" label={text.admin.backupEnabled} valuePropName="checked">
                            <Switch />
                          </Form.Item>
                          <Form.Item name="frequency" label={text.admin.backupFrequency} rules={[{ required: true }]}>
                            <Select options={backupFrequencyOptions} />
                          </Form.Item>
                          <Form.Item name="time" label={text.admin.backupTime} rules={[{ required: true }]}>
                            <Input placeholder="03:00" maxLength={5} />
                          </Form.Item>
                          <Form.Item name="keep" label={text.admin.backupKeep} rules={[{ required: true }]}>
                            <InputNumber min={1} max={100} style={{ width: '100%' }} />
                          </Form.Item>
                          <Form.Item className="adminBackupSubmit">
                            <Button type="primary" htmlType="submit" loading={backupSettingsSave.isPending}>
                              {text.common.save}
                            </Button>
                          </Form.Item>
                        </div>
                      </Form>
                    ) : null}
                  </div>
                  <div className="adminBackupTableHead">
                    <Typography.Title level={5}>{text.admin.backupFiles}</Typography.Title>
                    <Typography.Text type="secondary">{text.admin.backupCount(backups.data?.items.length ?? 0)}</Typography.Text>
                  </div>
                  {backups.isLoading ? (
                    <LoadingBlock />
                  ) : backups.isError ? (
                    <ErrorBlock error={backups.error} />
                  ) : (
                    <Table<BackupItem>
                      rowKey="name"
                      pagination={{ defaultPageSize: 10, hideOnSinglePage: true }}
                      scroll={{ x: 760 }}
                      dataSource={backups.data?.items ?? []}
                      columns={[
                        {
                          title: text.admin.backupFile,
                          dataIndex: 'name',
                          width: 320,
                          render: (name: string) => (
                            <Typography.Text code ellipsis={{ tooltip: name }} className="backupFileName">
                              {name}
                            </Typography.Text>
                          )
                        },
                        { title: text.admin.backupDatabase, dataIndex: 'database', width: 120 },
                        { title: text.admin.createdAt, dataIndex: 'createdAt', width: 220, render: (value: string) => new Date(value).toLocaleString(lang === 'zh' ? 'zh-CN' : 'en-US') },
                        { title: text.admin.backupSize, dataIndex: 'size', width: 100, render: (value: number) => formatBytes(value) },
                        {
                          title: text.common.actions,
                          width: 120,
                          render: (_, row) => (
                            <Space size={4}>
                              <Tooltip title={text.common.download}>
                                <Button
                                  type="text"
                                  icon={<DownloadOutlined />}
                                  loading={backupDownload.isPending && backupDownload.variables === row.name}
                                  onClick={() => backupDownload.mutate(row.name)}
                                />
                              </Tooltip>
                              <Popconfirm
                                title={text.common.confirmDelete}
                                okText={text.common.delete}
                                cancelText={text.common.cancel}
                                onConfirm={() => backupDelete.mutate(row.name)}
                              >
                                <Button type="text" danger icon={<DeleteOutlined />} loading={backupDelete.isPending && backupDelete.variables === row.name} />
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
          ]}
        />
      </Card>
      {userOpen ? (
        <UserModal
          loading={userCreate.isPending}
          roleOptions={roleOptions}
          groupOptions={groupOptions}
          onCancel={() => setUserOpen(false)}
          onSave={(values) => userCreate.mutate(values)}
        />
      ) : null}
      {editingUser ? (
        <UserEditModal
          user={editingUser}
          roleOptions={roleOptions}
          groupOptions={groupOptions}
          loading={userSave.isPending}
          onCancel={() => setEditingUser(null)}
          onSave={(values) => userSave.mutate({ name: editingUser.name, role: values.role, groups: values.groups ?? [] })}
        />
      ) : null}
      {groupOpen ? (
        <GroupModal
          editingGroup={editingGroup}
          loading={groupSave.isPending}
          userOptions={userOptions}
          onCancel={closeGroup}
          onSave={(values) => groupSave.mutate(values)}
        />
      ) : null}
      {langOpen ? <LangModal editingLang={editingLang} loading={langSave.isPending} onCancel={closeLang} onSave={(values) => langSave.mutate(values)} /> : null}
      {judgerOpen ? (
        <JudgerModal
          editingJudger={editingJudger}
          loading={judgerSave.isPending}
          onCancel={closeJudger}
          onSave={(values) => judgerSave.mutate(values)}
        />
      ) : null}
    </>
  )
}

function saveBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  anchor.click()
  URL.revokeObjectURL(url)
}

function UserModal({
  loading,
  roleOptions,
  groupOptions,
  onCancel,
  onSave
}: {
  loading: boolean
  roleOptions: { value: string; label: string }[]
  groupOptions: { value: number; label: string }[]
  onCancel: () => void
  onSave: (values: UserForm) => void
}) {
  const { text } = useLocale()
  const [form] = Form.useForm<UserForm>()
  return (
    <Modal
      open
      destroyOnHidden
      width={760}
      title={text.admin.addUser}
      okText={text.common.create}
      cancelText={text.common.cancel}
      confirmLoading={loading}
      onCancel={onCancel}
      onOk={() => form.submit()}
    >
      <Form<UserForm> form={form} layout="vertical" initialValues={{ name: '', mail: '', password: '', role: 'user', groups: [] }} onFinish={onSave}>
        <Form.Item name="name" label={text.prefs.username} rules={[{ required: true, whitespace: true }, { min: limits.usernameMin }, { max: limits.username }]}>
          <Input autoComplete="off" maxLength={limits.username} />
        </Form.Item>
        <Form.Item name="mail" label={text.profile.email} rules={[{ required: true }, { type: 'email' }]}>
          <Input autoComplete="off" maxLength={limits.mail} />
        </Form.Item>
        <Form.Item name="password" label={text.admin.initialPassword} rules={[{ required: true }, { min: 8 }]}>
          <Input.Password autoComplete="new-password" />
        </Form.Item>
        <Form.Item name="role" label={text.admin.role} rules={[{ required: true }]}>
          <Select options={roleOptions} />
	        </Form.Item>
	        <Form.Item name="groups" label={text.admin.userGroups}>
	          <IdSelect kind="groups" options={groupOptions} />
	        </Form.Item>
      </Form>
    </Modal>
  )
}

function UserEditModal({
  user,
  roleOptions,
  groupOptions,
  loading,
  onCancel,
  onSave
}: {
  user: UserRow
  roleOptions: { value: string; label: string }[]
  groupOptions: { value: number; label: string }[]
  loading: boolean
  onCancel: () => void
  onSave: (values: UserEditForm) => void
}) {
  const { text } = useLocale()
  const [form] = Form.useForm<UserEditForm>()
  return (
    <Modal
      open
      destroyOnHidden
      width={760}
      title={`${user.name} · ${text.common.edit}`}
      okText={text.common.save}
      cancelText={text.common.cancel}
      confirmLoading={loading}
      onCancel={onCancel}
      onOk={() => form.submit()}
    >
      <Form<UserEditForm> form={form} layout="vertical" initialValues={{ role: user.role, groups: user.groups ?? [] }} onFinish={onSave}>
        <Form.Item name="role" label={text.admin.role} rules={[{ required: true }]}>
          <Select options={roleOptions} />
	        </Form.Item>
	        <Form.Item name="groups" label={text.admin.userGroups}>
	          <IdSelect kind="groups" options={groupOptions} />
	        </Form.Item>
      </Form>
    </Modal>
  )
}

function GroupModal({
  editingGroup,
  loading,
  userOptions,
  onCancel,
  onSave
}: {
  editingGroup: GroupRow | null
  loading: boolean
  userOptions: { value: number; label: string }[]
  onCancel: () => void
  onSave: (values: GroupForm) => void
}) {
  const { text } = useLocale()
  const [form] = Form.useForm<GroupForm>()
  const initialValues = { name: editingGroup?.name ?? '', users: editingGroup?.users ?? [] }
  return (
    <Modal
      open
      destroyOnHidden
      width={760}
      title={editingGroup ? text.admin.editGroup : text.admin.addGroup}
      okText={text.common.save}
      cancelText={text.common.cancel}
      confirmLoading={loading}
      onCancel={onCancel}
      onOk={() => form.submit()}
    >
      <Form<GroupForm> form={form} layout="vertical" initialValues={initialValues} onFinish={onSave}>
        <Form.Item name="name" label={text.admin.name} rules={[{ required: true, whitespace: true }]}>
          <Input maxLength={limits.name} showCount />
	        </Form.Item>
	        <Form.Item name="users" label={text.admin.users}>
	          <IdSelect kind="users" options={userOptions} />
	        </Form.Item>
      </Form>
    </Modal>
  )
}

function LangModal({
  editingLang,
  loading,
  onCancel,
  onSave
}: {
  editingLang: LanguageRow | null
  loading: boolean
  onCancel: () => void
  onSave: (values: AdminLangCreate) => void
}) {
  const { text } = useLocale()
  const [form] = Form.useForm<AdminLangCreate>()
  const initialValues = editingLang ?? defaultLanguage
  return (
    <Modal open destroyOnHidden title={editingLang ? text.admin.editLang : text.admin.addLang} okText={text.common.save} cancelText={text.common.cancel} confirmLoading={loading} onCancel={onCancel} onOk={() => form.submit()} width={720}>
      <Form<AdminLangCreate> form={form} layout="vertical" initialValues={initialValues} onFinish={onSave}>
        <Form.Item name="id" label="ID" rules={[{ required: true, whitespace: true }]}>
          <Input maxLength={limits.languageId} />
        </Form.Item>
        <Form.Item name="name" label={text.admin.name} rules={[{ required: true, whitespace: true }]}>
          <Input maxLength={limits.name} showCount />
        </Form.Item>
        <Form.Item name="source" label={text.admin.source} rules={[{ required: true, whitespace: true }]}>
          <Input placeholder="main.cc" maxLength={limits.source} />
        </Form.Item>
        <Form.Item name="image" label={text.admin.image} rules={[{ required: true, whitespace: true }]}>
          <Input placeholder="gcc:14" maxLength={256} />
        </Form.Item>
        <Form.Item name="compile" label={text.admin.compile}>
          <Input.TextArea rows={3} />
        </Form.Item>
        <Form.Item name="run" label={text.admin.run} rules={[{ required: true, whitespace: true }]}>
          <Input.TextArea rows={2} />
        </Form.Item>
      </Form>
    </Modal>
  )
}

function JudgerModal({
  editingJudger,
  loading,
  onCancel,
  onSave
}: {
  editingJudger: JudgerRow | null
  loading: boolean
  onCancel: () => void
  onSave: (values: JudgerForm) => void
}) {
  const { text } = useLocale()
  const [form] = Form.useForm<JudgerForm>()
  const initialValues = editingJudger ? { name: editingJudger.name, auth: '' } : { name: '', auth: '' }
  return (
    <Modal open destroyOnHidden title={editingJudger ? text.admin.editJudger : text.admin.addJudger} okText={editingJudger ? text.common.save : text.common.create} cancelText={text.common.cancel} confirmLoading={loading} onCancel={onCancel} onOk={() => form.submit()}>
      <Form<JudgerForm> form={form} layout="vertical" initialValues={initialValues} onFinish={onSave}>
        <Form.Item name="name" label={text.admin.name} rules={[{ required: true, whitespace: true }]}>
          <Input maxLength={limits.name} showCount />
        </Form.Item>
        {editingJudger ? (
          <Form.Item name="auth" label={text.admin.token}>
            <Input.Password placeholder={text.admin.keepAuth} />
          </Form.Item>
        ) : null}
      </Form>
    </Modal>
  )
}
