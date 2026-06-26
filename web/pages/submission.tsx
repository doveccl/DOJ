import { App as AntApp, Card, Col, Flex, Row, Space, Switch, Table, Tooltip, Typography } from 'antd'
import type { TableProps } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useParams } from 'react-router-dom'

import { getLangs, getSubmission, updateSubmission } from '../client'
import type { Case, SubmissionDetail } from '../client'
import { ProblemLink, UserLink } from '../components/entity'
import { MarkdownPreview } from '../components/markdown'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { SubmissionStatus } from '../components/status'
import { useLocale } from '../locale'
import { useSession } from '../session'
import { caseCode, formatTime, memoryText, submissionCode } from '../utils/format'

export function SubmissionDetailPage() {
  const { message } = AntApp.useApp()
  const { lang, text } = useLocale()
  const session = useSession()
  const client = useQueryClient()
  const params = useParams()
  const id = Number(params.id)
  const query = useQuery({
    queryKey: ['submission', id],
    queryFn: () => getSubmission(id),
    enabled: Number.isFinite(id)
  })
  const languages = useQuery({ queryKey: ['languages'], queryFn: getLangs })
  const updatePublic = useMutation({
    mutationFn: (value: boolean) => updateSubmission(id, { public: value }),
    onSuccess: (next) => {
      client.setQueryData<SubmissionDetail>(['submission', id], (current) => (current ? { ...current, submission: next } : current))
      client.invalidateQueries({ queryKey: ['submissions'] })
    },
    onError: (error) => message.error(error.message)
  })

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
  const canUpdatePublic = session.admin || session.name === submission.user

  return (
    <Flex vertical gap={20}>
      <Row gutter={[20, 20]} align="top">
        <Col xs={24} lg={16}>
          <Card
            title={
              <Space size={10}>
                <Typography.Text strong>{submissionCode(submission.id)}</Typography.Text>
                <SubmissionStatus status={submission.status} />
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
            <Flex vertical gap={16}>
              {submission.message ? <MarkdownPreview value={codeMarkdown(submission.message, 'text')} /> : null}
              <Table<Case> rowKey="no" columns={caseColumns(text)} dataSource={cases} pagination={false} size="small" />
            </Flex>
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card>
            <Flex vertical gap={12}>
              <Meta label={text.submissions.problem}>
                <ProblemLink id={submission.problemId} title={submission.problemTitle} />
              </Meta>
              <Meta label={text.submissions.user}>
                <UserLink name={submission.user} />
              </Meta>
              <Meta label={text.submissions.language}>{languageName}</Meta>
              <Meta label={text.submissions.created}>{formatTime(submission.createdAt, lang)}</Meta>
            </Flex>
          </Card>
        </Col>
      </Row>
      {code.trim() ? (
        <Card
          title={text.submissions.source}
          extra={
            canUpdatePublic ? (
              <Space size={8}>
                <Typography.Text type="secondary">{text.problem.publicSource}</Typography.Text>
                <Switch
                  checked={submission.public}
                  loading={updatePublic.isPending}
                  disabled={updatePublic.isPending}
                  onChange={(checked) => updatePublic.mutate(checked)}
                />
              </Space>
            ) : null
          }
        >
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
      <span>{children}</span>
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
    { title: text.submissions.cases, dataIndex: 'no', render: (no: number) => caseCode(no) },
    {
      title: text.submissions.status,
      dataIndex: 'status',
      render: (status: string) => <SubmissionStatus status={status} />
    },
    {
      title: text.submissions.time,
      dataIndex: 'timeMs',
      render: (value?: number) => (value === undefined ? '-' : `${value}ms`)
    },
    {
      title: text.submissions.memory,
      dataIndex: 'memoryKb',
      render: (value?: number) => memoryText(value)
    },
    {
      title: text.submissions.message,
      dataIndex: 'message',
      ellipsis: { showTitle: false },
      render: (value: string) => (value ? <Tooltip title={value}>{value}</Tooltip> : '-')
    }
  ]
}
