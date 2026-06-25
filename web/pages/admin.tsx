import { DeleteOutlined, EditOutlined, KeyOutlined, PlusOutlined, SearchOutlined } from '@ant-design/icons'
import { App as AntApp, Button, Card, Form, Input, Modal, Popconfirm, Select, Space, Statistic, Switch, Table, Tabs, Tag, Tooltip, Typography } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import {
  createAdminUser,
  createAdminGroup,
  createAdminJudger,
  createAdminLang,
  deleteAdminGroup,
  deleteAdminJudger,
  deleteAdminLang,
  deleteAdminUser,
  getAdmin,
  resetAdminUserPassword,
  updateAdminGroup,
  updateAdminJudger,
  updateAdminLang,
  updateAdminUser,
  updateAdminSettings
} from '../client'
import type { AdminGroupUpdate, AdminLangCreate, AdminOverview, AdminSettings, AdminUserCreate } from '../client'
import { UserLink } from '../components/entity'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { IdSelect } from '../components/id-select'
import { useLocale } from '../locale'
import { useSession } from '../session'
import { formatDuration } from '../utils/format'
import { limits } from '../utils/limits'

type UserRow = AdminOverview['users'][number]
type GroupRow = AdminOverview['groups'][number]
type LanguageRow = AdminOverview['languages'][number]
type JudgerRow = AdminOverview['judgers'][number]
type JudgerForm = { name: string; auth?: string }
type UserForm = AdminUserCreate
type UserEditForm = Pick<AdminUserCreate, 'role' | 'groups'>
type GroupForm = AdminGroupUpdate
type SettingsForm = Pick<AdminSettings, 'siteName' | 'allowRegistration' | 'allowGuestAccess' | 'defaultSubmissionPublic'>

const defaultDockerfile = `FROM gcc:14
WORKDIR /src
COPY main.cc main.cc
RUN g++ -std=c++20 -O2 -pipe -static -s main.cc -o /main
CMD ["/main"]
`

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
  const query = useQuery({ queryKey: ['admin'], queryFn: getAdmin })
  const showError = (error: unknown) => {
    message.error(error instanceof Error ? error.message : text.common.loadingFailed)
  }
  const saveOverview = (data: AdminOverview) => {
    client.setQueryData<AdminOverview>(['admin'], data)
    message.success(text.common.saved)
  }
  const settings = useMutation({
    mutationFn: (values: SettingsForm) => {
      const current = client.getQueryData<AdminOverview>(['admin'])?.settings
      return updateAdminSettings({ notice: current?.notice ?? '', ...values })
    },
    onSuccess: (data) => {
      client.setQueryData<AdminOverview>(['admin'], (old) => (old ? { ...old, settings: data } : old))
      client.setQueryData(['site'], data)
      message.success(text.common.saved)
    },
    onError: showError
  })
  const userSave = useMutation({
    mutationFn: ({ name, role, groups }: { name: string; role: string; groups: number[] }) => updateAdminUser(name, { role, groups }),
    onSuccess: (data) => {
      saveOverview(data)
      setEditingUser(null)
    },
    onError: showError
  })
  const userCreate = useMutation({
    mutationFn: createAdminUser,
    onSuccess: (data) => {
      saveOverview(data)
      setUserOpen(false)
    },
    onError: showError
  })
  const userDelete = useMutation({
    mutationFn: deleteAdminUser,
    onSuccess: saveOverview,
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
      saveOverview(data)
      closeGroup()
    },
    onError: showError
  })
  const groupDelete = useMutation({
    mutationFn: deleteAdminGroup,
    onSuccess: saveOverview,
    onError: showError
  })
  const langSave = useMutation({
    mutationFn: (values: AdminLangCreate) => (editingLang ? updateAdminLang(editingLang.id, values) : createAdminLang(values)),
    onSuccess: (data) => {
      saveOverview(data)
      closeLang()
    },
    onError: showError
  })
  const langDelete = useMutation({
    mutationFn: deleteAdminLang,
    onSuccess: saveOverview,
    onError: showError
  })
  const judgerSave = useMutation({
    mutationFn: (values: JudgerForm) =>
      editingJudger ? updateAdminJudger(editingJudger.id, { name: values.name, auth: values.auth || undefined }) : createAdminJudger({ name: values.name }),
    onSuccess: (data) => {
      const created = data.judgers.find((row) => row.token)
      saveOverview(data)
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
    onSuccess: saveOverview,
    onError: showError
  })

  if (!session.admin) {
    return <ErrorBlock error={text.common.forbidden} />
  }
  if (query.isLoading) {
    return <LoadingBlock />
  }
  if (query.isError) {
    return <ErrorBlock error={query.error} />
  }
  if (!query.data) {
    return <ErrorBlock error={text.common.emptyResponse} />
  }

  const data = query.data
  const roleText: Record<string, string> = text.admin.roles
  const roleOptions = [
    { value: 'user', label: roleText.user },
    { value: 'admin', label: roleText.admin }
  ]
  const groupOptions = data.groups.map((group) => ({ value: group.id, label: group.name }))
  const userOptions = data.users.map((user) => ({ value: user.id, label: user.name }))
  const userGroupIds = (row: UserRow) => row.groups ?? []
  const groupUserIds = (row: GroupRow) => data.users.filter((user) => userGroupIds(user).includes(row.id)).map((user) => user.id)
  const userQuery = userSearch.trim().toLowerCase()
  const groupQuery = groupSearch.trim().toLowerCase()
  const groupName = (id: number) => data.groups.find((group) => group.id === id)?.name ?? ''
  const filteredUsers = userQuery
    ? data.users.filter((user) =>
        [user.name, user.mail, roleText[user.role] ?? user.role, ...userGroupIds(user).map(groupName)].some((value) => value.toLowerCase().includes(userQuery))
      )
    : data.users
  const filteredGroups = groupQuery
    ? data.groups.filter((group) =>
        [group.name, ...groupUserIds(group).map((id) => data.users.find((user) => user.id === id)?.name ?? '')].some((value) =>
          value.toLowerCase().includes(groupQuery)
        )
      )
    : data.groups

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
          destroyOnHidden
          items={[
            {
              key: 'settings',
              label: text.admin.settings,
              children: (
                <Form<SettingsForm>
                  layout="vertical"
                  style={{ maxWidth: 680 }}
                  initialValues={data.settings}
                  key={`${data.settings.siteName}:${data.settings.allowRegistration}:${data.settings.allowGuestAccess}:${data.settings.defaultSubmissionPublic}`}
                  onFinish={(values) => settings.mutate(values)}
                >
                  <Form.Item name="siteName" label={text.admin.siteName} rules={[{ required: true }]}>
                    <Input maxLength={limits.name} showCount />
                  </Form.Item>
                  <Form.Item name="allowRegistration" label={text.admin.allowRegistration} valuePropName="checked">
                    <Switch />
                  </Form.Item>
                  <Form.Item name="allowGuestAccess" label={text.admin.allowGuestAccess} valuePropName="checked">
                    <Switch />
                  </Form.Item>
                  <Form.Item name="defaultSubmissionPublic" label={text.admin.defaultSubmissionPublic} valuePropName="checked">
                    <Switch />
                  </Form.Item>
                  <Button type="primary" htmlType="submit" loading={settings.isPending}>
                    {text.common.save}
                  </Button>
                </Form>
              )
            },
            {
              key: 'users',
              label: text.admin.users,
              children: (
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
              children: (
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
              children: (
                <Space orientation="vertical" size={16} style={{ width: '100%' }}>
                  <Button type="primary" icon={<PlusOutlined />} onClick={() => openLang()}>
                    {text.admin.addLang}
                  </Button>
                  <Table<LanguageRow>
                    rowKey="id"
                    pagination={false}
                    dataSource={data.languages}
                    columns={[
                      { title: text.admin.name, dataIndex: 'name' },
                      { title: text.admin.source, dataIndex: 'source' },
                      {
                        title: text.admin.dockerfile,
                        dataIndex: 'dockerfile',
                        ellipsis: true,
                        render: (value: string) => <Typography.Text code>{value.split('\n')[0]}</Typography.Text>
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
              children: (
                <Space orientation="vertical" size={16} style={{ width: '100%' }}>
                  <Space size={24} wrap>
                    <Statistic title={text.admin.queued} value={data.queue.queued} />
                    <Statistic title={text.admin.running} value={data.queue.running} />
                    <Statistic title={text.admin.done} value={data.queue.done} />
                  </Space>
                  <Button type="primary" icon={<PlusOutlined />} onClick={() => openJudger()}>
                    {text.admin.addJudger}
                  </Button>
                  <Table<JudgerRow>
                    rowKey="id"
                    pagination={false}
                    dataSource={data.judgers}
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
          <IdSelect options={groupOptions} />
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
          <IdSelect options={groupOptions} />
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
          <IdSelect options={userOptions} />
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
  const initialValues = editingLang ?? { id: '', name: '', source: 'main.cc', dockerfile: defaultDockerfile }
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
        <Form.Item name="dockerfile" label={text.admin.dockerfile} rules={[{ required: true, whitespace: true }]}>
          <Input.TextArea rows={8} />
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
