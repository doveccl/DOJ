import { App as AntApp, Space, Typography } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { api, apiData, apiEmpty, downloadBackup } from '../../client'
import type {
  AdminGroupUpdate,
  AdminJudgers,
  AdminLang,
  AdminLangCreate,
  AdminMembers,
  BackupSettings as BackupSettingsData
} from '../../client'
import { MarkdownPreview } from '../../components/markdown'
import { useApiMessage } from '../../components/use-api-message'
import { useDebouncedValue } from '../../components/use-debounced-value'
import { useLocale } from '../../locale'
import { useSession } from '../../session'
import { codeBlock } from '../../utils/markdown'
import { saveBlob } from '../../utils/download'
import type {
  BackupSettingsForm,
  GroupRow,
  JudgerForm,
  JudgerRow,
  LanguageRow,
  SettingsForm,
  UserForm,
  UserRow
} from './types'

function useMembersCache() {
  const client = useQueryClient()
  const { text } = useLocale()
  const { message } = AntApp.useApp()
  return (data?: AdminMembers) => {
    if (data) {
      client.setQueryData<AdminMembers>(['admin-members'], data)
    }
    void client.invalidateQueries({ queryKey: ['admin-members'] })
    void client.invalidateQueries({ queryKey: ['admin-users'] })
    void client.invalidateQueries({ queryKey: ['admin-groups'] })
    message.success(text.common.saved)
  }
}

export function useAdminSettings() {
  const { text } = useLocale()
  const session = useSession()
  const client = useQueryClient()
  const { message, showError } = useApiMessage()
  const query = useQuery({ queryKey: ['admin-settings'], queryFn: () => apiData(api.GET('/api/admin/settings')), enabled: session.admin })
  const save = useMutation({
    mutationFn: (body: Partial<SettingsForm>) => apiData(api.PATCH('/api/admin/settings', { body })),
    onSuccess: (data) => {
      client.setQueryData(['admin-settings'], data)
      client.setQueryData(['site'], data)
      message.success(text.common.saved)
    },
    onError: showError
  })
  return { query, save, savePatch: (patch: Partial<SettingsForm>) => save.mutate(patch) }
}

// Membership options (users + groups) used by the user/group modals.
export function useAdminMembers(enabled: boolean) {
  return useQuery({ queryKey: ['admin-members'], queryFn: () => apiData(api.GET('/api/admin/members')), enabled })
}

export function useAdminUsers(enabled: boolean) {
  const { text } = useLocale()
  const { modal } = AntApp.useApp()
  const { showError } = useApiMessage()
  const saveMembers = useMembersCache()
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<UserRow | null>(null)
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const searchKey = useDebouncedValue(search.trim())
  const query = useQuery({
    queryKey: ['admin-users', searchKey, page, pageSize],
    queryFn: () => apiData(api.GET('/api/admin/users', { params: { query: { q: searchKey, page, pageSize } } })),
    enabled
  })
  const create = useMutation({
    mutationFn: (body: UserForm) => apiData(api.POST('/api/admin/users', { body })),
    onSuccess: (data) => {
      saveMembers(data)
      setOpen(false)
    },
    onError: showError
  })
  const update = useMutation({
    mutationFn: ({ name, role, groups }: { name: string; role: string; groups: number[] }) => apiData(api.PATCH('/api/admin/users/{name}', { params: { path: { name } }, body: { role, groups } })),
    onSuccess: (data) => {
      saveMembers(data)
      setEditing(null)
    },
    onError: showError
  })
  const remove = useMutation({
    mutationFn: (name: string) => apiData(api.DELETE('/api/admin/users/{name}', { params: { path: { name } } })),
    onSuccess: saveMembers,
    onError: showError
  })
  const password = useMutation({
    mutationFn: (name: string) => apiData(api.POST('/api/admin/users/{name}/password', { params: { path: { name } } })),
    onSuccess: (data) => {
      modal.info({
        title: text.admin.resetPassword,
        content: <Typography.Paragraph copyable>{data.password}</Typography.Paragraph>
      })
    },
    onError: showError
  })
  return { open, setOpen, editing, setEditing, search, setSearch, page, setPage, pageSize, setPageSize, query, create, update, remove, password }
}

export function useAdminGroups(enabled: boolean) {
  const { showError } = useApiMessage()
  const saveMembers = useMembersCache()
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<GroupRow | null>(null)
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const searchKey = useDebouncedValue(search.trim())
  const query = useQuery({
    queryKey: ['admin-groups', searchKey, page, pageSize],
    queryFn: () => apiData(api.GET('/api/admin/groups', { params: { query: { q: searchKey, page, pageSize } } })),
    enabled
  })
  function close() {
    setOpen(false)
    setEditing(null)
  }
  const save = useMutation({
    mutationFn: (values: AdminGroupUpdate) =>
      editing
        ? apiData(api.PATCH('/api/admin/groups/{id}', { params: { path: { id: editing.id } }, body: values }))
        : apiData(api.POST('/api/admin/groups', { body: values })),
    onSuccess: (data) => {
      saveMembers(data)
      close()
    },
    onError: showError
  })
  const remove = useMutation({
    mutationFn: (id: number) => apiData(api.DELETE('/api/admin/groups/{id}', { params: { path: { id } } })),
    onSuccess: saveMembers,
    onError: showError
  })
  function openModal(row?: GroupRow) {
    setEditing(row ?? null)
    setOpen(true)
  }
  return { open, editing, search, setSearch, page, setPage, pageSize, setPageSize, query, save, remove, openModal, close }
}

export function useAdminLanguages(enabled: boolean) {
  const { text } = useLocale()
  const client = useQueryClient()
  const { message, showError } = useApiMessage()
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<LanguageRow | null>(null)
  const query = useQuery({ queryKey: ['admin-languages'], queryFn: () => apiData(api.GET('/api/admin/languages')), enabled })
  const saveCache = (data: AdminLang[]) => {
    client.setQueryData<AdminLang[]>(['admin-languages'], data)
    message.success(text.common.saved)
  }
  function close() {
    setOpen(false)
    setEditing(null)
  }
  const save = useMutation({
    mutationFn: (values: AdminLangCreate) =>
      editing
        ? apiData(api.PATCH('/api/admin/languages/{id}', { params: { path: { id: editing.id } }, body: values }))
        : apiData(api.POST('/api/admin/languages', { body: values })),
    onSuccess: (data) => {
      saveCache(data)
      close()
    },
    onError: showError
  })
  const remove = useMutation({
    mutationFn: (id: string) => apiData(api.DELETE('/api/admin/languages/{id}', { params: { path: { id } } })),
    onSuccess: saveCache,
    onError: showError
  })
  function openModal(row?: LanguageRow) {
    setEditing(row ?? null)
    setOpen(true)
  }
  return { open, editing, query, save, remove, openModal, close }
}

export function useAdminJudgers(enabled: boolean) {
  const { text } = useLocale()
  const client = useQueryClient()
  const { message, showError } = useApiMessage()
  const { modal } = AntApp.useApp()
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<JudgerRow | null>(null)
  const query = useQuery({ queryKey: ['admin-judgers'], queryFn: () => apiData(api.GET('/api/admin/judgers')), enabled })
  const saveCache = (data: AdminJudgers) => {
    client.setQueryData<AdminJudgers>(['admin-judgers'], data)
    message.success(text.common.saved)
  }
  function close() {
    setOpen(false)
    setEditing(null)
  }
  const save = useMutation({
    mutationFn: (values: JudgerForm) =>
      editing
        ? apiData(api.PATCH('/api/admin/judgers/{id}', { params: { path: { id: editing.id } }, body: { name: values.name, auth: values.auth || undefined } }))
        : apiData(api.POST('/api/admin/judgers', { body: { name: values.name } })),
    onSuccess: (data) => {
      const created = data.judgers.find((row) => row.token)
      saveCache(data)
      close()
      if (created?.token) {
        const server = typeof window === 'undefined' ? 'http://localhost:7974' : window.location.origin
        const composeExample = `services:
  judger:
    image: doveccl/doj:4
    command: ["doj", "judger"]
    restart: unless-stopped
    privileged: true
    network_mode: host
    pid: host
    cgroup: host
    environment:
      SERVER: ${server}
      TOKEN: ${created.token}
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /var/lib/doj:/var/lib/doj`
        modal.info({
          title: text.admin.judgerTokenCreated,
          width: 760,
          content: (
            <Space orientation="vertical" style={{ width: '100%' }}>
              <Typography.Text type="secondary">{text.admin.judgerTokenHelp}</Typography.Text>
              <MarkdownPreview value={codeBlock(composeExample, 'yaml')} />
            </Space>
          )
        })
      }
    },
    onError: showError
  })
  const remove = useMutation({
    mutationFn: (id: number) => apiData(api.DELETE('/api/admin/judgers/{id}', { params: { path: { id } } })),
    onSuccess: saveCache,
    onError: showError
  })
  function openModal(row?: JudgerRow) {
    setEditing(row ?? null)
    setOpen(true)
  }
  return { open, editing, query, save, remove, openModal, close }
}

export function useAdminBackups(enabled: boolean) {
  const { text } = useLocale()
  const client = useQueryClient()
  const { message, showError } = useApiMessage()
  const settings = useQuery({ queryKey: ['backup-settings'], queryFn: () => apiData(api.GET('/api/admin/backups/settings')), enabled })
  const backups = useQuery({ queryKey: ['backups'], queryFn: () => apiData(api.GET('/api/admin/backups')), enabled, refetchInterval: (query) => (query.state.data?.running ? 5000 : false) })
  const saveSettings = useMutation({
    mutationFn: (body: BackupSettingsForm) => apiData(api.PATCH('/api/admin/backups/settings', { body })),
    onSuccess: (data: BackupSettingsData) => {
      client.setQueryData(['backup-settings'], data)
      message.success(text.common.saved)
    },
    onError: showError
  })
  const create = useMutation({
    mutationFn: () => apiData(api.POST('/api/admin/backups')),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['backups'] })
      message.success(text.admin.backupManualDone)
    },
    onError: showError
  })
  const remove = useMutation({
    mutationFn: (name: string) => apiEmpty(api.DELETE('/api/admin/backups/{name}', { params: { path: { name } } })),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['backups'] })
    },
    onError: showError
  })
  const download = useMutation({
    mutationFn: downloadBackup,
    onSuccess: (blob, name) => saveBlob(blob, name),
    onError: showError
  })
  return { settings, backups, saveSettings, create, remove, download }
}
