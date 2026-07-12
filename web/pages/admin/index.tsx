import { Card, Form, Tabs } from 'antd'
import type { ReactNode } from 'react'
import { useSearchParams } from 'react-router-dom'

import type { UseQueryResult } from '@tanstack/react-query'
import { ErrorBlock, LoadingBlock } from '../../components/state'
import { useLocale } from '../../locale'
import { useSession } from '../../session'
import {
  GroupModal,
  JudgerModal,
  LangModal,
  UserEditModal,
  UserModal
} from './modals'
import {
  useAdminBackups,
  useAdminGroups,
  useAdminJudgers,
  useAdminLanguages,
  useAdminMembers,
  useAdminSettings,
  useAdminUsers
} from './hooks'
import { BackupsTab, GroupsTab, JudgersTab, LanguagesTab, SettingsTab, UsersTab } from './tabs'
import type { SettingsForm } from './types'

const defaultAdminTab = 'settings'
const adminTabs = new Set(['settings', 'users', 'groups', 'languages', 'judgers', 'backups'])

// Renders a loading/error placeholder for a tab query, or null when data is ready.
function queryBlock(query: UseQueryResult<unknown>, enabled: boolean, empty?: ReactNode): ReactNode {
  if (query.isLoading || (enabled && query.data === undefined && !query.isError)) {
    return <LoadingBlock />
  }
  if (query.isError) {
    return <ErrorBlock error={query.error} />
  }
  return empty ?? null
}

export function AdminPage() {
  const { lang, text } = useLocale()
  const session = useSession()
  const [params, setParams] = useSearchParams()
  const activeTab = adminTab(params.get('tab'))
  const [settingsForm] = Form.useForm<SettingsForm>()

  const usersEnabled = session.admin && activeTab === 'users'
  const groupsEnabled = session.admin && activeTab === 'groups'
  const languagesEnabled = session.admin && activeTab === 'languages'
  const judgersEnabled = session.admin && activeTab === 'judgers'
  const backupsEnabled = session.admin && activeTab === 'backups'

  const settings = useAdminSettings()
  const users = useAdminUsers(usersEnabled)
  const groups = useAdminGroups(groupsEnabled)
  const languages = useAdminLanguages(languagesEnabled)
  const judgers = useAdminJudgers(judgersEnabled)
  const backups = useAdminBackups(backupsEnabled)
  const membersEnabled = session.admin && (users.open || Boolean(users.editing) || groups.open)
  const members = useAdminMembers(membersEnabled)

  if (!session.admin) {
    return <ErrorBlock error={text.common.forbidden} />
  }

  const settingsData = settings.query.data
  const membersData = members.data
  const roleText: Record<string, string> = text.admin.roles
  const roleOptions = [
    { value: 'user', label: roleText.user },
    { value: 'admin', label: roleText.admin }
  ]
  const groupOptions = (membersData?.groups ?? []).map((group) => ({ value: group.id, label: group.name }))
  const userOptions = (membersData?.users ?? []).map((user) => ({ value: user.id, label: user.name }))
  const backupCronOptions = [
    { value: '0 3 * * *', label: text.admin.backupCronPresets.daily },
    { value: '0 3 * * 1', label: text.admin.backupCronPresets.weekly }
  ]

  const settingsBlock = settings.query.isLoading ? (
    <LoadingBlock />
  ) : settings.query.isError ? (
    <ErrorBlock error={settings.query.error} />
  ) : !settingsData ? (
    <ErrorBlock error={text.common.emptyResponse} />
  ) : null

  const saveSiteName = (value: string) => {
    if (!settingsData) {
      return
    }
    const siteName = value.trim()
    if (!siteName || siteName === settingsData.siteName) {
      settingsForm.setFieldValue('siteName', settingsData.siteName)
      return
    }
    settings.savePatch({ siteName })
  }

  function changeTab(key: string) {
    const tab = adminTab(key)
    const next = new URLSearchParams(params)
    if (tab === defaultAdminTab) {
      next.delete('tab')
    } else {
      next.set('tab', tab)
    }
    setParams(next)
  }

  return (
    <>
      <Card>
        <Tabs
          activeKey={activeTab}
          destroyOnHidden
          onChange={changeTab}
          items={[
            {
              key: 'settings',
              label: text.admin.settings,
              children: <SettingsTab block={settingsBlock} form={settingsForm} data={settingsData} pending={settings.save.isPending} saveSiteName={saveSiteName} savePatch={settings.savePatch} />
            },
            {
              key: 'users',
              label: text.admin.users,
              children: <UsersTab block={queryBlock(users.query, usersEnabled)} data={users.query.data} search={users.search} setSearch={users.setSearch} page={users.page} pageSize={users.pageSize} setPage={users.setPage} setPageSize={users.setPageSize} roleText={roleText} onAdd={() => users.setOpen(true)} onEdit={users.setEditing} onResetPassword={(name) => users.password.mutate(name)} resetLoadingName={users.password.isPending ? users.password.variables : undefined} onDelete={(name) => users.remove.mutate(name)} />
            },
            {
              key: 'groups',
              label: text.admin.groups,
              children: <GroupsTab block={queryBlock(groups.query, groupsEnabled)} data={groups.query.data} search={groups.search} setSearch={groups.setSearch} page={groups.page} pageSize={groups.pageSize} setPage={groups.setPage} setPageSize={groups.setPageSize} onAdd={() => groups.openModal()} onEdit={groups.openModal} onDelete={(id) => groups.remove.mutate(id)} />
            },
            {
              key: 'languages',
              label: text.admin.languages,
              children: <LanguagesTab block={queryBlock(languages.query, languagesEnabled)} data={languages.query.data} onAdd={() => languages.openModal()} onEdit={languages.openModal} onDelete={(id) => languages.remove.mutate(id)} />
            },
            {
              key: 'judgers',
              label: text.admin.judgers,
              children: <JudgersTab block={queryBlock(judgers.query, judgersEnabled)} data={judgers.query.data} lang={lang} onAdd={() => judgers.openModal()} onEdit={judgers.openModal} onDelete={(id) => judgers.remove.mutate(id)} />
            },
            {
              key: 'backups',
              label: text.admin.backups,
              children: <BackupsTab settings={backups.settings} backups={backups.backups} cronOptions={backupCronOptions} createLoading={backups.create.isPending} settingsSaveLoading={backups.saveSettings.isPending} downloadName={backups.download.isPending ? backups.download.variables : undefined} deleteName={backups.remove.isPending ? backups.remove.variables : undefined} onSaveSettings={(values) => backups.saveSettings.mutate(values)} onCreate={() => backups.create.mutate()} onDownload={(name) => backups.download.mutate(name)} onDelete={(name) => backups.remove.mutate(name)} lang={lang} />
            }
          ]}
        />
      </Card>
      {users.open ? (
        <UserModal
          loading={users.create.isPending}
          roleOptions={roleOptions}
          groupOptions={groupOptions}
          onCancel={() => users.setOpen(false)}
          onSave={(values) => users.create.mutate(values)}
        />
      ) : null}
      {users.editing ? (
        <UserEditModal
          user={users.editing}
          roleOptions={roleOptions}
          groupOptions={groupOptions}
          loading={users.update.isPending}
          onCancel={() => users.setEditing(null)}
          onSave={(values) => users.editing && users.update.mutate({ name: users.editing.name, role: values.role, groups: values.groups ?? [] })}
        />
      ) : null}
      {groups.open ? (
        <GroupModal
          editingGroup={groups.editing}
          loading={groups.save.isPending}
          userOptions={userOptions}
          onCancel={groups.close}
          onSave={(values) => groups.save.mutate(values)}
        />
      ) : null}
      {languages.open ? <LangModal editingLang={languages.editing} loading={languages.save.isPending} onCancel={languages.close} onSave={(values) => languages.save.mutate(values)} /> : null}
      {judgers.open ? (
        <JudgerModal
          editingJudger={judgers.editing}
          loading={judgers.save.isPending}
          onCancel={judgers.close}
          onSave={(values) => judgers.save.mutate(values)}
        />
      ) : null}
    </>
  )
}

function adminTab(value: string | null) {
  return value && adminTabs.has(value) ? value : defaultAdminTab
}
