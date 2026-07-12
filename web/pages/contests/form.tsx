import { Col, DatePicker, Form, Input, Row, Select } from 'antd'
import type { FormInstance } from 'antd'
import type { Dayjs } from 'dayjs'

import type { ProblemRef } from '../../client'
import { MarkdownEditor } from '../../components/markdown'
import { ProblemRefInput } from '../../components/problem-ref'
import { useLocale } from '../../locale'
import { limits } from '../../utils/limits'

export type ContestFormValues = {
  title: string
  description: string
  kind: string
  startAt: Dayjs
  endAt: Dayjs
  freezeAt?: Dayjs | null
  problems?: ProblemRef[]
}

export function ContestFormFields({
  form,
  editorId,
  problemOptions,
  loading = false
}: {
  form: FormInstance<ContestFormValues>
  editorId: string
  problemOptions: { value: number; label: string }[]
  loading?: boolean
}) {
  const { text } = useLocale()
  const kind = Form.useWatch('kind', form) ?? 'OI'
  const dateSpan = kind === 'ICPC' ? 8 : 12
  return (
    <>
      <Form.Item name="title" label={text.contests.name} rules={[{ required: true, whitespace: true }]}>
        <Input maxLength={limits.title} showCount />
      </Form.Item>
      <Form.Item name="description" label={text.contests.description}>
        <MarkdownEditor id={editorId} height={220} />
      </Form.Item>
      <Form.Item name="kind" label={text.contests.kind}>
        <Select options={[{ value: 'OI', label: 'OI' }, { value: 'ICPC', label: 'ICPC' }]} />
      </Form.Item>
      <Row gutter={12}>
        <Col xs={24} md={dateSpan}>
          <Form.Item name="startAt" label={text.contests.start} rules={[{ required: true }]}>
            <DatePicker showTime style={{ width: '100%' }} />
          </Form.Item>
        </Col>
        <Col xs={24} md={dateSpan}>
          <Form.Item name="endAt" label={text.contests.end} rules={[{ required: true }]}>
            <DatePicker showTime style={{ width: '100%' }} />
          </Form.Item>
        </Col>
        {kind === 'ICPC' ? (
          <Col xs={24} md={8}>
            <Form.Item name="freezeAt" label={text.contests.freeze}>
              <DatePicker showTime style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        ) : null}
      </Row>
      <Form.Item name="problems" label={text.contests.problems}>
        <ProblemRefInput options={problemOptions} loading={loading} />
      </Form.Item>
    </>
  )
}
