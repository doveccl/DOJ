import type { ReactNode } from 'react'

import { Form, Input, Switch } from 'antd'
import type { FormInstance } from 'antd'

import type { AdminSettings } from '../../../client'
import { useLocale } from '../../../locale'
import { limits } from '../../../utils/limits'
import type { SettingsForm } from '../types'

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
  const text = useLocale().text
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
