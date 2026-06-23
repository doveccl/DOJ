import { DeleteOutlined, EditOutlined, KeyOutlined, PlusOutlined } from '@ant-design/icons'
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
import { ErrorBlock, LoadingBlock } from '../components/state'
import { useLocale } from '../locale'
import { useSession } from '../session'
import { formatDuration } from '../utils/format'

type UserRow = AdminOverview['users'][number]
type GroupRow = AdminOverview['groups'][number]
type LanguageRow = AdminOverview['languages'][number]
type JudgerRow = AdminOverview['judgers'][number]
type JudgerForm = { name: string; auth?: string }
type UserForm = AdminUserCreate
type SettingsForm = Pick<AdminSettings, 'siteName' | 'registration' | 'guest' | 'defaultPublicSource'>

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
  const [editingLang, setEditingLang] = useState<LanguageRow | null>(null)
  const [editingJudger, setEditingJudger] = useState<JudgerRow | null>(null)
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
    onSuccess: saveOverview,
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
            <Space direction="vertical" style={{ width: '100%' }}>
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
  const groupOptions = data.groups.map((group) => ({ value: group.id, label: group.name }))
  const userGroupIds = (row: UserRow) => row.groups ?? []

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
                  key={`${data.settings.siteName}:${data.settings.registration}:${data.settings.guest}:${data.settings.defaultPublicSource}`}
                  onFinish={(values) => settings.mutate(values)}
                >
                  <Form.Item name="siteName" label={text.admin.siteName} rules={[{ required: true }]}>
                    <Input />
                  </Form.Item>
                  <Form.Item name="registration" label={text.admin.registration} valuePropName="checked">
                    <Switch />
                  </Form.Item>
                  <Form.Item name="guest" label={text.admin.guest} valuePropName="checked">
                    <Switch />
                  </Form.Item>
                  <Form.Item name="defaultPublicSource" label={text.admin.defaultPublicSource} valuePropName="checked">
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
                  <Button type="primary" icon={<PlusOutlined />} onClick={() => setUserOpen(true)}>
                    {text.admin.addUser}
                  </Button>
                  <Table<UserRow>
                    rowKey="name"
                    pagination={false}
                    dataSource={data.users}
                    columns={[
                      { title: text.rank.user, dataIndex: 'name' },
                      { title: text.profile.email, dataIndex: 'mail' },
                      {
                        title: text.admin.role,
                        dataIndex: 'role',
                        render: (role: string, row) => (
                          <Select
                            value={role}
                            options={[
                              { value: 'admin', label: roleText.admin },
                              { value: 'user', label: roleText.user }
                            ]}
                            onChange={(next) => userSave.mutate({ name: row.name, role: next, groups: userGroupIds(row) })}
                          />
                        )
                      },
                      {
                        title: text.admin.userGroups,
                        dataIndex: 'groups',
                        width: 260,
                        render: (groups: number[] | undefined, row) => (
                          <Select
                            mode="multiple"
                            value={groups ?? []}
                            options={groupOptions}
                            maxTagCount="responsive"
                            placeholder={text.admin.groups}
                            style={{ width: '100%' }}
                            onChange={(next) => userSave.mutate({ name: row.name, role: row.role, groups: next })}
                          />
                        )
                      },
                      {
                        title: text.common.edit,
                        width: 112,
                        render: (_, row) => (
                          <Space size={4}>
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
                  <Button type="primary" icon={<PlusOutlined />} onClick={() => openGroup()}>
                    {text.admin.addGroup}
                  </Button>
                  <Table<GroupRow>
                    rowKey="id"
                    pagination={false}
                    dataSource={data.groups}
                    columns={[
                      { title: text.admin.name, dataIndex: 'name' },
                      {
                        title: text.admin.users,
                        width: 120,
                        render: (_, row) => data.users.filter((user) => userGroupIds(user).includes(row.id)).length
                      },
                      {
                        title: text.common.edit,
                        width: 112,
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
                      { title: text.problems.title, dataIndex: 'name', width: 180 },
                      { title: text.admin.source, dataIndex: 'source', width: 140 },
                      {
                        title: text.admin.dockerfile,
                        dataIndex: 'dockerfile',
                        ellipsis: true,
                        render: (value: string) => <Typography.Text code>{value.split('\n')[0]}</Typography.Text>
                      },
                      {
                        title: text.common.edit,
                        width: 112,
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
                        width: 120,
                        render: (online: boolean) => <Tag color={online ? 'green' : 'default'}>{online ? text.admin.online : text.admin.offline}</Tag>
                      },
                      {
                        title: text.admin.uptime,
                        dataIndex: 'uptimeSeconds',
                        width: 160,
                        render: (value: number, row) => (row.online ? formatDuration(value, lang) : '-')
                      },
                      {
                        title: text.common.edit,
                        width: 112,
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
          groupOptions={groupOptions}
          roleOptions={[
            { value: 'user', label: roleText.user },
            { value: 'admin', label: roleText.admin }
          ]}
          onCancel={() => setUserOpen(false)}
          onSave={(values) => userCreate.mutate(values)}
        />
      ) : null}
      {groupOpen ? <GroupModal editingGroup={editingGroup} loading={groupSave.isPending} onCancel={closeGroup} onSave={(values) => groupSave.mutate(values)} /> : null}
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
  groupOptions,
  roleOptions,
  onCancel,
  onSave
}: {
  loading: boolean
  groupOptions: { value: number; label: string }[]
  roleOptions: { value: string; label: string }[]
  onCancel: () => void
  onSave: (values: UserForm) => void
}) {
  const { text } = useLocale()
  const [form] = Form.useForm<UserForm>()
  return (
    <Modal open destroyOnHidden title={text.admin.addUser} okText={text.common.create} cancelText={text.common.cancel} confirmLoading={loading} onCancel={onCancel} onOk={() => form.submit()}>
      <Form<UserForm> form={form} layout="vertical" initialValues={{ name: '', mail: '', password: '', role: 'user', groups: [] }} onFinish={onSave}>
        <Form.Item name="name" label={text.prefs.username} rules={[{ required: true, whitespace: true }, { min: 3 }, { max: 32 }]}>
          <Input autoComplete="off" />
        </Form.Item>
        <Form.Item name="mail" label={text.profile.email} rules={[{ required: true }, { type: 'email' }]}>
          <Input autoComplete="off" />
        </Form.Item>
        <Form.Item name="password" label={text.admin.initialPassword} rules={[{ required: true }, { min: 8 }]}>
          <Input.Password autoComplete="new-password" />
        </Form.Item>
        <Form.Item name="role" label={text.admin.role} rules={[{ required: true }]}>
          <Select options={roleOptions} />
        </Form.Item>
        <Form.Item name="groups" label={text.admin.userGroups}>
          <Select mode="multiple" options={groupOptions} maxTagCount="responsive" />
        </Form.Item>
      </Form>
    </Modal>
  )
}

function GroupModal({
  editingGroup,
  loading,
  onCancel,
  onSave
}: {
  editingGroup: GroupRow | null
  loading: boolean
  onCancel: () => void
  onSave: (values: AdminGroupUpdate) => void
}) {
  const { text } = useLocale()
  const [form] = Form.useForm<AdminGroupUpdate>()
  return (
    <Modal open destroyOnHidden title={editingGroup ? text.admin.editGroup : text.admin.addGroup} okText={text.common.save} cancelText={text.common.cancel} confirmLoading={loading} onCancel={onCancel} onOk={() => form.submit()}>
      <Form<AdminGroupUpdate> form={form} layout="vertical" initialValues={editingGroup ?? { name: '' }} onFinish={onSave}>
        <Form.Item name="name" label={text.admin.name} rules={[{ required: true, whitespace: true }]}>
          <Input />
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
          <Input />
        </Form.Item>
        <Form.Item name="name" label={text.problems.title} rules={[{ required: true, whitespace: true }]}>
          <Input />
        </Form.Item>
        <Form.Item name="source" label={text.admin.source} rules={[{ required: true, whitespace: true }]}>
          <Input placeholder="main.cc" />
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
          <Input />
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
