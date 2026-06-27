import { Form, Input, Modal, Select } from 'antd'

import { IdSelect } from '../../components/id-select'
import { useLocale } from '../../locale'
import { limits } from '../../utils/limits'
import type { GroupForm, GroupRow, JudgerForm, JudgerRow, LanguageRow, UserEditForm, UserForm } from './types'
import { defaultLanguage } from './types'
import type { AdminLangCreate } from '../../client'

type Option<T extends string | number> = { value: T; label: string }

export function UserModal({
  loading,
  roleOptions,
  groupOptions,
  onCancel,
  onSave
}: {
  loading: boolean
  roleOptions: Option<string>[]
  groupOptions: Option<number>[]
  onCancel: () => void
  onSave: (values: UserForm) => void
}) {
  const { text } = useLocale()
  const [form] = Form.useForm<UserForm>()
  return (
    <Modal open destroyOnHidden width={760} title={text.admin.addUser} okText={text.common.create} cancelText={text.common.cancel} confirmLoading={loading} onCancel={onCancel} onOk={() => form.submit()}>
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

export function UserEditModal({
  user,
  roleOptions,
  groupOptions,
  loading,
  onCancel,
  onSave
}: {
  user: { name: string; role: string; groups?: number[] }
  roleOptions: Option<string>[]
  groupOptions: Option<number>[]
  loading: boolean
  onCancel: () => void
  onSave: (values: UserEditForm) => void
}) {
  const { text } = useLocale()
  const [form] = Form.useForm<UserEditForm>()
  return (
    <Modal open destroyOnHidden width={760} title={`${user.name} · ${text.common.edit}`} okText={text.common.save} cancelText={text.common.cancel} confirmLoading={loading} onCancel={onCancel} onOk={() => form.submit()}>
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

export function GroupModal({
  editingGroup,
  loading,
  userOptions,
  onCancel,
  onSave
}: {
  editingGroup: GroupRow | null
  loading: boolean
  userOptions: Option<number>[]
  onCancel: () => void
  onSave: (values: GroupForm) => void
}) {
  const { text } = useLocale()
  const [form] = Form.useForm<GroupForm>()
  const initialValues = { name: editingGroup?.name ?? '', users: editingGroup?.users ?? [] }
  return (
    <Modal open destroyOnHidden width={760} title={editingGroup ? text.admin.editGroup : text.admin.addGroup} okText={text.common.save} cancelText={text.common.cancel} confirmLoading={loading} onCancel={onCancel} onOk={() => form.submit()}>
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

export function LangModal({
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

export function JudgerModal({
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
