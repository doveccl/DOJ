import { EditOutlined } from '@ant-design/icons'
import { Button, Card, Space } from 'antd'
import { useState } from 'react'
import type { ReactNode } from 'react'

import { useLocale } from '../locale'
import { MarkdownEditor, MarkdownPreview } from './markdown'

export function DescriptionCard({ id, header, value, editable, onSave }: {
  id: string
  header: ReactNode
  value: string
  editable: boolean
  onSave: (value: string) => Promise<void>
}) {
  const { text } = useLocale()
  const [editing, setEditing] = useState(false)
  const [saving, setSaving] = useState(false)
  const [draft, setDraft] = useState('')

  async function save() {
    setSaving(true)
    try {
      await onSave(draft)
      setEditing(false)
    } catch {
      // The page owns error presentation; keep the editor open for retry.
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card
      title={header}
      styles={!editing && !value.trim() ? { body: { display: 'none' } } : undefined}
      extra={editable ? (
        editing ? (
          <Space size={8}>
            <Button size="small" disabled={saving} onClick={() => setEditing(false)}>{text.common.cancel}</Button>
            <Button size="small" type="primary" loading={saving} onClick={() => void save()}>{text.common.save}</Button>
          </Space>
        ) : (
          <Button size="small" icon={<EditOutlined />} onClick={() => { setDraft(value); setEditing(true) }}>{text.common.edit}</Button>
        )
      ) : null}
    >
      {editing ? <MarkdownEditor id={id} value={draft} onChange={setDraft} /> : value.trim() ? <MarkdownPreview id={id} value={value} /> : null}
    </Card>
  )
}
