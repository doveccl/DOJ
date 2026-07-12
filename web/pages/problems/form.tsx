import { Col, Form, Input, Row } from 'antd'

import { JudgeModeSelect } from '../../components/judge'
import { LimitInput } from '../../components/limit'
import { MarkdownEditor } from '../../components/markdown'
import { TagSelect } from '../../components/tag-select'
import { useLocale } from '../../locale'
import { limits } from '../../utils/limits'

export type ProblemFormValues = {
  title: string
  statement?: string
  tags?: string[]
  mode: string
  timeMs: number
  memoryMb: number
}

export function ProblemFormFields({
  statement,
  showMode = false
}: {
  statement?: { editorId: string; height?: number; upload?: (file: File) => Promise<string> }
  showMode?: boolean
}) {
  const { text } = useLocale()
  return (
    <>
      <Form.Item name="title" label={text.problems.title} rules={[{ required: true, whitespace: true }]}>
        <Input maxLength={limits.title} showCount />
      </Form.Item>
      <Form.Item name="tags" label={text.problems.tag}>
        <TagSelect kind="problem" mode="tags" />
      </Form.Item>
      {showMode ? (
        <Form.Item name="mode" label={text.problem.mode}>
          <JudgeModeSelect />
        </Form.Item>
      ) : null}
      <Row gutter={12}>
        <Col xs={24} md={12}>
          <Form.Item name="timeMs" label={text.problems.time} rules={[{ required: true }]}>
            <LimitInput min={100} step={100} unit="ms" />
          </Form.Item>
        </Col>
        <Col xs={24} md={12}>
          <Form.Item name="memoryMb" label={text.problems.memory} rules={[{ required: true }]}>
            <LimitInput min={16} step={16} unit="MB" />
          </Form.Item>
        </Col>
      </Row>
      {statement ? (
        <Form.Item name="statement" label={text.problem.statement} rules={[{ required: true, whitespace: true }]}>
          <MarkdownEditor id={statement.editorId} height={statement.height} upload={statement.upload} />
        </Form.Item>
      ) : null}
    </>
  )
}
