import { DatePicker, Form, Input } from 'antd'
import type { Dayjs } from 'dayjs'

import type { ProblemRef } from '../../client'
import { IdSelect } from '../../components/id-select'
import { MarkdownEditor } from '../../components/markdown'
import { ProblemRefInput } from '../../components/problem-ref'
import { useLocale } from '../../locale'
import { limits } from '../../utils/limits'

export type AssignmentFormValues = {
  title: string
  description: string
  endAt: Dayjs
  problems?: ProblemRef[]
  users?: number[]
  groups?: number[]
}

export function AssignmentFormFields({
  editorId,
  problemOptions,
  loading = false
}: {
  editorId: string
  problemOptions: { value: number; label: string }[]
  loading?: boolean
}) {
  const { text } = useLocale()
  return (
    <>
      <Form.Item name="title" label={text.assignments.name} rules={[{ required: true, whitespace: true }]}>
        <Input maxLength={limits.title} showCount />
      </Form.Item>
      <Form.Item name="description" label={text.assignments.instructions}>
        <MarkdownEditor id={editorId} height={220} />
      </Form.Item>
      <Form.Item name="endAt" label={text.assignments.deadline} rules={[{ required: true }]}>
        <DatePicker showTime style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item name="problems" label={text.assignments.problems}>
        <ProblemRefInput options={problemOptions} loading={loading} />
      </Form.Item>
      <Form.Item name="users" label={text.assignments.users}>
        <IdSelect kind="users" />
      </Form.Item>
      <Form.Item name="groups" label={text.assignments.groups}>
        <IdSelect kind="groups" />
      </Form.Item>
    </>
  )
}
