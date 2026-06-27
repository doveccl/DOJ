import { App as AntApp, Card, Form, Space, Tabs, Typography } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import {
  api,
  apiData,
  apiEmpty,
  downloadBackup
} from '../client'
import type { AdminGroupUpdate, AdminJudgers, AdminLang, AdminLangCreate, AdminMembers } from '../client'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { useDebouncedValue } from '../components/use-debounced-value'
import { useLocale } from '../locale'
import { useSession } from '../session'
import { GroupModal, JudgerModal, LangModal, UserEditModal, UserModal } from './admin/modals'
import { BackupsTab, GroupsTab, JudgersTab, LanguagesTab, SettingsTab, UsersTab } from './admin/tabs'
import type { BackupSettingsForm, GroupRow, JudgerForm, JudgerRow, LanguageRow, SettingsForm, UserForm, UserRow } from './admin/types'

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
  const [userPage, setUserPage] = useState(1)
  const [userPageSize, setUserPageSize] = useState(20)
  const [groupPage, setGroupPage] = useState(1)
  const [groupPageSize, setGroupPageSize] = useState(20)
  const [activeTab, setActiveTab] = useState('settings')
  const [settingsForm] = Form.useForm<SettingsForm>()
  const userQuery = useDebouncedValue(userSearch.trim())
  const groupQuery = useDebouncedValue(groupSearch.trim())
  const membersEnabled = session.admin && (userOpen || Boolean(editingUser) || groupOpen)
  const usersEnabled = session.admin && activeTab === 'users'
  const groupsEnabled = session.admin && activeTab === 'groups'
  const languagesEnabled = session.admin && activeTab === 'languages'
  const judgersEnabled = session.admin && activeTab === 'judgers'
  const backupEnabled = session.admin && activeTab === 'backups'
  const membersQuery = useQuery({ queryKey: ['admin-members'], queryFn: () => apiData(api.GET('/api/admin/members')), enabled: membersEnabled })
  const usersQuery = useQuery({
    queryKey: ['admin-users', userQuery, userPage, userPageSize],
    queryFn: () => apiData(api.GET('/api/admin/users', { params: { query: { q: userQuery, page: userPage, pageSize: userPageSize } } })),
    enabled: usersEnabled
  })
  const groupsQuery = useQuery({
    queryKey: ['admin-groups', groupQuery, groupPage, groupPageSize],
    queryFn: () => apiData(api.GET('/api/admin/groups', { params: { query: { q: groupQuery, page: groupPage, pageSize: groupPageSize } } })),
    enabled: groupsEnabled
  })
  const languagesQuery = useQuery({ queryKey: ['admin-languages'], queryFn: () => apiData(api.GET('/api/admin/languages')), enabled: languagesEnabled })
  const judgersQuery = useQuery({ queryKey: ['admin-judgers'], queryFn: () => apiData(api.GET('/api/admin/judgers')), enabled: judgersEnabled })
  const settingsQuery = useQuery({ queryKey: ['admin-settings'], queryFn: () => apiData(api.GET('/api/admin/settings')), enabled: session.admin })
  const backupSettings = useQuery({ queryKey: ['backup-settings'], queryFn: () => apiData(api.GET('/api/admin/backups/settings')), enabled: backupEnabled })
  const backups = useQuery({ queryKey: ['backups'], queryFn: () => apiData(api.GET('/api/admin/backups')), enabled: backupEnabled, refetchInterval: (query) => (query.state.data?.running ? 5000 : false) })
  const showError = (error: unknown) => {
    message.error(error instanceof Error ? error.message : text.common.loadingFailed)
  }
  const saveMembers = (data?: AdminMembers) => {
    if (data) {
      client.setQueryData<AdminMembers>(['admin-members'], data)
    }
    void client.invalidateQueries({ queryKey: ['admin-members'] })
    void client.invalidateQueries({ queryKey: ['admin-users'] })
    void client.invalidateQueries({ queryKey: ['admin-groups'] })
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
    mutationFn: (body: Partial<SettingsForm>) => apiData(api.PATCH('/api/admin/settings', { body })),
    onSuccess: (data) => {
      client.setQueryData(['admin-settings'], data)
      client.setQueryData(['site'], data)
      message.success(text.common.saved)
    },
    onError: showError
  })
  const userSave = useMutation({
    mutationFn: ({ name, role, groups }: { name: string; role: string; groups: number[] }) => apiData(api.PATCH('/api/admin/users/{name}', { params: { path: { name } }, body: { role, groups } })),
    onSuccess: (data) => {
      saveMembers(data)
      setEditingUser(null)
    },
    onError: showError
  })
  const userCreate = useMutation({
    mutationFn: (body: UserForm) => apiData(api.POST('/api/admin/users', { body })),
    onSuccess: (data) => {
      saveMembers(data)
      setUserOpen(false)
    },
    onError: showError
  })
  const userDelete = useMutation({
    mutationFn: (name: string) => apiData(api.DELETE('/api/admin/users/{name}', { params: { path: { name } } })),
    onSuccess: saveMembers,
    onError: showError
  })
  const userPassword = useMutation({
    mutationFn: (name: string) => apiData(api.POST('/api/admin/users/{name}/password', { params: { path: { name } } })),
    onSuccess: (data) => {
      modal.info({
        title: text.admin.resetPassword,
        content: <Typography.Paragraph copyable>{data.password}</Typography.Paragraph>
      })
    },
    onError: showError
  })
  const groupSave = useMutation({
    mutationFn: (values: AdminGroupUpdate) =>
      editingGroup
        ? apiData(api.PATCH('/api/admin/groups/{id}', { params: { path: { id: editingGroup.id } }, body: values }))
        : apiData(api.POST('/api/admin/groups', { body: values })),
    onSuccess: (data) => {
      saveMembers(data)
      closeGroup()
    },
    onError: showError
  })
  const groupDelete = useMutation({
    mutationFn: (id: number) => apiData(api.DELETE('/api/admin/groups/{id}', { params: { path: { id } } })),
    onSuccess: saveMembers,
    onError: showError
  })
  const langSave = useMutation({
    mutationFn: (values: AdminLangCreate) =>
      editingLang
        ? apiData(api.PATCH('/api/admin/languages/{id}', { params: { path: { id: editingLang.id } }, body: values }))
        : apiData(api.POST('/api/admin/languages', { body: values })),
    onSuccess: (data) => {
      saveLanguages(data)
      closeLang()
    },
    onError: showError
  })
  const langDelete = useMutation({
    mutationFn: (id: string) => apiData(api.DELETE('/api/admin/languages/{id}', { params: { path: { id } } })),
    onSuccess: saveLanguages,
    onError: showError
  })
  const judgerSave = useMutation({
    mutationFn: (values: JudgerForm) =>
      editingJudger
        ? apiData(api.PATCH('/api/admin/judgers/{id}', { params: { path: { id: editingJudger.id } }, body: { name: values.name, auth: values.auth || undefined } }))
        : apiData(api.POST('/api/admin/judgers', { body: { name: values.name } })),
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
    mutationFn: (id: number) => apiData(api.DELETE('/api/admin/judgers/{id}', { params: { path: { id } } })),
    onSuccess: saveJudgers,
    onError: showError
  })
  const backupSettingsSave = useMutation({
    mutationFn: (body: BackupSettingsForm) => apiData(api.PATCH('/api/admin/backups/settings', { body })),
    onSuccess: (data) => {
      client.setQueryData(['backup-settings'], data)
      message.success(text.common.saved)
    },
    onError: showError
  })
  const backupCreate = useMutation({
    mutationFn: () => apiData(api.POST('/api/admin/backups')),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['backups'] })
      message.success(text.admin.backupManualDone)
    },
    onError: showError
  })
  const backupDelete = useMutation({
    mutationFn: (name: string) => apiEmpty(api.DELETE('/api/admin/backups/{name}', { params: { path: { name } } })),
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
  const usersData = usersQuery.data
  const groupsData = groupsQuery.data
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
  const usersBlock = usersQuery.isLoading ? (
    <LoadingBlock />
  ) : usersQuery.isError ? (
    <ErrorBlock error={usersQuery.error} />
  ) : usersEnabled && !usersData ? (
    <LoadingBlock />
  ) : null
  const groupsBlock = groupsQuery.isLoading ? (
    <LoadingBlock />
  ) : groupsQuery.isError ? (
    <ErrorBlock error={groupsQuery.error} />
  ) : groupsEnabled && !groupsData ? (
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
              children: <SettingsTab block={settingsBlock} form={settingsForm} data={settingsData} pending={settings.isPending} saveSiteName={saveSiteName} savePatch={saveSettingsPatch} />
            },
            {
              key: 'users',
              label: text.admin.users,
              children: <UsersTab block={usersBlock} data={usersData} search={userSearch} setSearch={setUserSearch} page={userPage} pageSize={userPageSize} setPage={setUserPage} setPageSize={setUserPageSize} roleText={roleText} onAdd={() => setUserOpen(true)} onEdit={setEditingUser} onResetPassword={(name) => userPassword.mutate(name)} resetLoadingName={userPassword.isPending ? userPassword.variables : undefined} onDelete={(name) => userDelete.mutate(name)} />
            },
            {
              key: 'groups',
              label: text.admin.groups,
              children: <GroupsTab block={groupsBlock} data={groupsData} search={groupSearch} setSearch={setGroupSearch} page={groupPage} pageSize={groupPageSize} setPage={setGroupPage} setPageSize={setGroupPageSize} onAdd={() => openGroup()} onEdit={openGroup} onDelete={(id) => groupDelete.mutate(id)} />
            },
            {
              key: 'languages',
              label: text.admin.languages,
              children: <LanguagesTab block={languagesBlock} data={languagesData} onAdd={() => openLang()} onEdit={openLang} onDelete={(id) => langDelete.mutate(id)} />
            },
            {
              key: 'judgers',
              label: text.admin.judgers,
              children: <JudgersTab block={judgersBlock} data={judgersData} lang={lang} onAdd={() => openJudger()} onEdit={openJudger} onDelete={(id) => judgerDelete.mutate(id)} />
            },
            {
              key: 'backups',
              label: text.admin.backups,
              children: <BackupsTab settings={backupSettings} backups={backups} frequencyOptions={backupFrequencyOptions} frequencyText={backupFrequencyText} createLoading={backupCreate.isPending} settingsSaveLoading={backupSettingsSave.isPending} downloadName={backupDownload.isPending ? backupDownload.variables : undefined} deleteName={backupDelete.isPending ? backupDelete.variables : undefined} onSaveSettings={(values) => backupSettingsSave.mutate(values)} onCreate={() => backupCreate.mutate()} onDownload={(name) => backupDownload.mutate(name)} onDelete={(name) => backupDelete.mutate(name)} lang={lang} />
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
