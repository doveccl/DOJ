import type { AdminGroupUpdate, AdminJudgers, AdminLang, AdminMembers, AdminSettings, AdminUserCreate, BackupSettings } from '../../client'

export type UserRow = AdminMembers['users'][number]
export type GroupRow = AdminMembers['groups'][number]
export type LanguageRow = AdminLang
export type JudgerRow = AdminJudgers['judgers'][number]
export type JudgerForm = { name: string; auth?: string }
export type UserForm = AdminUserCreate
export type UserEditForm = Pick<AdminUserCreate, 'role' | 'groups'>
export type GroupForm = AdminGroupUpdate
export type SettingsForm = Pick<AdminSettings, 'siteName' | 'allowRegistration' | 'allowGuestAccess' | 'defaultSubmissionPublic'>
export type BackupSettingsForm = BackupSettings

export const defaultLanguage = {
  id: '',
  name: '',
  source: 'main.cc',
  image: 'gcc',
  compile: 'g++ main.cc -o main',
  run: './main'
}
