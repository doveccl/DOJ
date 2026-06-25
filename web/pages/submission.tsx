import { Card, Col, Flex, Row, Space, Table, Tag, Typography } from 'antd'
import type { TableProps } from 'antd'
import { useQuery } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { Link, useParams } from 'react-router-dom'

import { getLangs, getSubmission } from '../client'
import type { Case } from '../client'
import { MarkdownPreview } from '../components/markdown'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { SubmissionStatus } from '../components/status'
import { useLocale } from '../locale'
import { formatTime, memoryText, problemCode } from '../utils/format'

export function SubmissionDetailPage() {
  const { lang, text } = useLocale()
  const params = useParams()
  const id = Number(params.id)
  const query = useQuery({
    queryKey: ['submission', id],
    queryFn: () => getSubmission(id),
    enabled: Number.isFinite(id)
  })
  const languages = useQuery({ queryKey: ['languages'], queryFn: getLangs })

  if (query.isLoading) {
    return <LoadingBlock />
  }
  if (query.isError) {
    return <ErrorBlock error={query.error} />
  }
  if (!query.data) {
    return <ErrorBlock error={text.common.emptyResponse} />
  }

  const { submission, code, cases } = query.data
  const languageName = (languages.data ?? []).find((item) => item.id === submission.language)?.name ?? submission.language

  return (
    <Flex vertical gap={20}>
      <Row gutter={[20, 20]} align="top">
        <Col xs={24} lg={16}>
          <Card
            title={
              <Space size={10}>
                <Typography.Text strong>#{submission.id}</Typography.Text>
                <SubmissionStatus status={submission.status} />
                {submission.public ? <Tag>{text.submissions.public}</Tag> : null}
              </Space>
            }
            extra={
              <Space size={12} wrap>
                <MetaInline label={text.submissions.score}>{submission.score}</MetaInline>
                <MetaInline label={text.submissions.time}>{submission.timeMs === undefined ? '-' : `${submission.timeMs}ms`}</MetaInline>
                <MetaInline label={text.submissions.memory}>{memoryText(submission.memoryKb)}</MetaInline>
              </Space>
            }
          >
            <Table<Case> rowKey="no" columns={caseColumns(text)} dataSource={cases} pagination={false} size="small" />
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card>
            <Flex vertical gap={12}>
              <Meta label={text.submissions.problem}>
                <Link to={`/problems/${submission.problemId}`}>
                  {problemCode(submission.problemId)} {submission.problemTitle}
                </Link>
              </Meta>
              <Meta label={text.submissions.user}>
                <Link to={`/users/${submission.user}`}>{submission.user}</Link>
              </Meta>
              <Meta label={text.submissions.language}>{languageName}</Meta>
              <Meta label={text.submissions.created}>{formatTime(submission.createdAt, lang)}</Meta>
            </Flex>
          </Card>
        </Col>
      </Row>
      {code.trim() ? (
        <Card title={text.submissions.source}>
          <MarkdownPreview value={codeMarkdown(code, submission.language)} />
        </Card>
      ) : null}
    </Flex>
  )
}

function codeMarkdown(source: string, syntax: string) {
  return `\`\`\`${fenceLanguage(syntax)}\n${source.replaceAll('```', '`\\`\\`')}\n\`\`\``
}

function fenceLanguage(syntax: string) {
  const first = syntax.trim().split(/[\s,;|]+/)[0] ?? ''
  return first.replace(/[^a-zA-Z0-9_+#.-]/g, '')
}

function Meta({ label, children }: { label: string; children: ReactNode }) {
  return (
    <Flex justify="space-between" gap={12}>
      <Typography.Text type="secondary">{label}</Typography.Text>
      <Typography.Text>{children}</Typography.Text>
    </Flex>
  )
}

function MetaInline({ label, children }: { label: string; children: ReactNode }) {
  return (
    <Space size={4}>
      <Typography.Text type="secondary">{label}</Typography.Text>
      <Typography.Text strong>{children}</Typography.Text>
    </Space>
  )
}

function caseColumns(text: ReturnType<typeof useLocale>['text']): TableProps<Case>['columns'] {
  return [
    { title: '#', dataIndex: 'no', width: 56 },
    {
      title: text.submissions.status,
      dataIndex: 'status',
      render: (status: string) => <SubmissionStatus status={status} />
    },
    {
      title: text.submissions.time,
      dataIndex: 'timeMs',
      width: 90,
      render: (value?: number) => (value === undefined ? '-' : `${value}ms`)
    },
    {
      title: text.submissions.memory,
      dataIndex: 'memoryKb',
      width: 110,
      render: (value?: number) => memoryText(value)
    },
    {
      title: text.submissions.message,
      dataIndex: 'message',
      ellipsis: true,
      render: (value: string) => value || '-'
    }
  ]
}
