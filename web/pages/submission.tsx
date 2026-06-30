import { ReloadOutlined } from '@ant-design/icons'
import { App as AntApp, BorderBeam, Button, Card, Col, Flex, Popconfirm, Progress, Row, Space, Spin, Switch, Table, Tooltip, Typography } from 'antd'
import type { TableProps } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useParams } from 'react-router-dom'

import { api, apiData } from '../client'
import type { Case, SubmissionDetail } from '../client'
import { ProblemLink, UserLink } from '../components/entity'
import { MarkdownPreview } from '../components/markdown'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { SubmissionStatus } from '../components/status'
import { useLocale } from '../locale'
import { useSession } from '../session'
import { caseCode, formatBytes, formatTime, memoryText, submissionCode } from '../utils/format'

export function SubmissionDetailPage() {
  const { message } = AntApp.useApp()
  const { lang, text } = useLocale()
  const session = useSession()
  const client = useQueryClient()
  const params = useParams()
  const id = Number(params.id)
  const query = useQuery({
    queryKey: ['submission', id],
    queryFn: () => apiData(api.GET('/api/submissions/{id}', { params: { path: { id } } })),
    enabled: Number.isFinite(id)
  })
  const languages = useQuery({ queryKey: ['languages'], queryFn: () => apiData(api.GET('/api/languages')) })
  const updatePublic = useMutation({
    mutationFn: (value: boolean) => apiData(api.PATCH('/api/submissions/{id}', { params: { path: { id } }, body: { public: value } })),
    onSuccess: (_next, value) => {
      client.setQueryData<SubmissionDetail>(['submission', id], (current) => (current ? { ...current, submission: { ...current.submission, public: value } } : current))
      client.invalidateQueries({ queryKey: ['submissions'] })
    },
    onError: (error) => message.error(error.message)
  })
  const rejudge = useMutation({
    mutationFn: () => apiData(api.POST('/api/submissions/{id}/rejudge', { params: { path: { id } } })),
    onSuccess: () => {
      client.setQueryData<SubmissionDetail>(['submission', id], (current) =>
        current
          ? {
              ...current,
              submission: {
                ...current.submission,
                status: 'queued',
                score: 0,
                message: '',
                timeMs: undefined,
                memoryKb: undefined
              },
              cases: [],
              progress: undefined
            }
          : current
      )
      void client.invalidateQueries({ queryKey: ['submissions'] })
      message.success(text.submissions.rejudged)
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
  const judging = submission.status === 'queued' || submission.status === 'judging'

  return (
    <Flex vertical gap={20}>
      <Row gutter={[20, 20]} align="top">
        <Col xs={24} lg={16}>
          <ResultCard judging={judging}>
            <Card
              style={{ position: 'relative' }}
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
                  {session.admin ? (
                    <Popconfirm title={text.submissions.confirmRejudge} okText={text.submissions.rejudge} cancelText={text.common.cancel} onConfirm={() => rejudge.mutate()}>
                      <Button size="small" icon={<ReloadOutlined />} loading={rejudge.isPending}>{text.submissions.rejudge}</Button>
                    </Popconfirm>
                  ) : null}
                </Space>
              }
            >
              <Flex vertical gap={24}>
                {judging ? <JudgeProgressPanel status={submission.status} progress={query.data.progress} /> : null}
                {submission.message ? (
                  <div className="submissionMessagePreview">
                    <MarkdownPreview value={codeMarkdown(submission.message, 'text')} />
                  </div>
                ) : null}
                <Table<Case> rowKey="no" columns={caseColumns(text)} dataSource={cases} pagination={false} size="small" scroll={{ x: 620 }} />
              </Flex>
            </Card>
          </ResultCard>
        </Col>
        <Col xs={24} lg={8}>
          <Card>
            <Flex vertical gap={12}>
              <Meta label={text.submissions.problem}>
                <ProblemLink id={submission.problemId} title={submission.problemTitle} maxWidth={220} />
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
          <SourceBlock code={code} language={submission.language} />
        </Card>
      ) : null}
    </Flex>
  )
}

function ResultCard({ judging, children }: { judging: boolean; children: ReactNode }) {
  return judging ? <BorderBeam>{children}</BorderBeam> : children
}

function JudgeProgressPanel({ status, progress }: { status: string; progress?: SubmissionDetail['progress'] }) {
  const { text } = useLocale()
  const copy = progressCopy(status, progress, text)
  const percent = progress && progress.total && progress.total > 0 ? clampPercent(progress.done, progress.total) : undefined
  return (
    <Flex vertical gap={8} className="judgeProgressPanel">
      <Flex align="center" justify="space-between" gap={12} wrap>
        <Space size={8}>
          {percent === undefined ? <Spin size="small" /> : null}
          <Typography.Text strong>{copy.title}</Typography.Text>
        </Space>
        {copy.detail ? <Typography.Text type="secondary">{copy.detail}</Typography.Text> : null}
      </Flex>
      {percent === undefined ? null : <Progress percent={percent} size="small" showInfo={false} status="active" />}
    </Flex>
  )
}

function progressCopy(status: string, progress: SubmissionDetail['progress'] | undefined, text: ReturnType<typeof useLocale>['text']) {
  if (!progress) {
    return { title: status === 'queued' ? text.submissions.progress.waiting : text.submissions.progress.judging }
  }
  switch (progress.stage) {
    case 'leased':
      return { title: text.submissions.progress.leased }
    case 'download':
      return {
        title: text.submissions.progress.download,
        detail: progress.total && progress.total > 0 ? `${formatBytes(progress.done)} / ${formatBytes(progress.total)}` : text.submissions.progress.downloaded(formatBytes(progress.done))
      }
    case 'prepare':
      return { title: text.submissions.progress.prepare }
    case 'compile':
      return { title: text.submissions.progress.compile }
    case 'judge':
      return {
        title: text.submissions.progress.judge,
        detail: progress.total && progress.total > 0 ? text.submissions.progress.cases(progress.done, progress.total) : undefined
      }
    case 'upload':
      return { title: text.submissions.progress.upload }
    default:
      return { title: text.submissions.progress.judging }
  }
}

function clampPercent(done: number, total: number) {
  if (total <= 0) {
    return 0
  }
  return Math.max(0, Math.min(99, Math.floor((done / total) * 100)))
}

function codeMarkdown(source: string, syntax: string) {
  return `\`\`\`${fenceLanguage(syntax)}\n${source.replaceAll('```', '`\\`\\`')}\n\`\`\``
}

const maxHighlightedLineLength = 20000

function SourceBlock({ code, language }: { code: string; language: string }) {
  if (!hasLongLine(code, maxHighlightedLineLength)) {
    return <MarkdownPreview value={codeMarkdown(code, language)} />
  }
  return <pre className="sourceCodeBlock"><code>{code}</code></pre>
}

function hasLongLine(value: string, max: number) {
  let line = 0
  for (const char of value) {
    if (char === '\n') {
      line = 0
    } else if (++line > max) {
      return true
    }
  }
  return false
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
    { title: text.submissions.cases, dataIndex: 'no', width: 96, render: (no: number) => caseCode(no) },
    {
      title: text.submissions.status,
      dataIndex: 'status',
      width: 120,
      render: (status: string) => <SubmissionStatus status={status} />
    },
    {
      title: text.submissions.time,
      dataIndex: 'timeMs',
      width: 96,
      render: (value?: number) => (value === undefined ? '-' : `${value}ms`)
    },
    {
      title: text.submissions.memory,
      dataIndex: 'memoryKb',
      width: 96,
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
