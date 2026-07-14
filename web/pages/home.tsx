import { EditOutlined } from '@ant-design/icons'
import { App as AntApp, Button, Card, Col, Flex, Form, Row, Space, Tag, Typography } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link } from 'react-router-dom'

import { api, apiData } from '../client'
import type { HomeAssignment, HomeContest, HomeProblem, NoticeUpdate } from '../client'
import { ProblemLink } from '../components/entity'
import { YearHeatmap } from '../components/heatmap'
import { MarkdownEditor, MarkdownPreview } from '../components/markdown'
import { EmptyBlock, ErrorBlock, LoadingBlock } from '../components/state'
import { useLocale } from '../locale'
import { useSession } from '../session'

type NoticeForm = {
  content: string
}

export function HomePage() {
  const { text } = useLocale()
  const session = useSession()
  const { message } = AntApp.useApp()
  const client = useQueryClient()
  const [noticeEditing, setNoticeEditing] = useState(false)
  const [noticeForm] = Form.useForm<NoticeForm>()
  const query = useQuery({ queryKey: ['home'], queryFn: () => apiData(api.GET('/api/home')) })
  const showError = (error: unknown) => {
    message.error(error instanceof Error ? error.message : text.common.loadingFailed)
  }
  const notice = useMutation({
    mutationFn: (values: NoticeForm) => apiData(api.PATCH('/api/home/notice', { body: { content: values.content } satisfies NoticeUpdate })),
    onSuccess: (home) => {
      client.setQueryData(['home'], home)
      message.success(text.common.saved)
      setNoticeEditing(false)
    },
    onError: showError
  })

  if (query.isLoading) {
    return <LoadingBlock />
  }
  if (query.isError) {
    return <ErrorBlock error={query.error} />
  }

  const home = query.data
  if (!home) {
    return <ErrorBlock error={text.common.emptyResponse} />
  }

  return (
    <Flex vertical gap={20} className="pageStack">
      <Card
        className="noticeCard"
        title={text.home.notice}
        extra={
          session.admin ? (
            noticeEditing ? (
              <Space size={8}>
                <Button onClick={() => setNoticeEditing(false)}>
                  {text.common.cancel}
                </Button>
                <Button type="primary" loading={notice.isPending} onClick={() => noticeForm.submit()}>
                  {text.common.save}
                </Button>
              </Space>
            ) : (
              <Button
                icon={<EditOutlined />}
                onClick={() => {
                  noticeForm.setFieldsValue({ content: home.notice })
                  setNoticeEditing(true)
                }}
              >
                {text.common.edit}
              </Button>
            )
          ) : null
        }
      >
        {noticeEditing ? (
          <Form<NoticeForm> form={noticeForm} layout="vertical" initialValues={{ content: home.notice }} onFinish={(values) => notice.mutate(values)}>
            <Form.Item name="content" rules={[{ required: true, whitespace: true }]} noStyle>
              <MarkdownEditor id="home-notice" />
            </Form.Item>
          </Form>
        ) : home.notice ? (
          <MarkdownPreview id="home-notice" value={home.notice} />
        ) : (
          <EmptyBlock />
        )}
      </Card>
      {home.heatmap.length > 0 ? (
        <Card title={text.home.heatmap}>
          <YearHeatmap cells={home.heatmap} />
        </Card>
      ) : null}
      <Row gutter={[20, 20]}>
        <Col xs={24} lg={8}>
          <ProblemList title={text.home.latestProblems} items={home.problems} />
        </Col>
        <Col xs={24} lg={8}>
          <AssignmentList title={text.home.assignments} items={home.assignments} />
        </Col>
        <Col xs={24} lg={8}>
          <ContestList title={text.home.contests} items={home.contests} />
        </Col>
      </Row>
    </Flex>
  )
}

function ProblemList({ title, items }: { title: string; items: HomeProblem[] }) {
  return (
    <Card title={title}>
      {items.length === 0 ? (
        <EmptyBlock />
      ) : (
        <Flex vertical gap={8}>
          {items.map((item) => (
            <ProblemLink key={item.id} id={item.id} title={item.title} />
          ))}
        </Flex>
      )}
    </Card>
  )
}

function AssignmentList({ title, items }: { title: string; items: HomeAssignment[] }) {
  const { text } = useLocale()
  return (
    <Card title={title}>
      {items.length === 0 ? (
        <EmptyBlock />
      ) : (
        <Flex vertical gap={8}>
          {items.map((item) => (
            <Flex key={item.id} justify="space-between" align="center" gap={12}>
              <Typography.Text strong ellipsis={{ tooltip: item.title }}>
                <Link to={`/assignments/${item.id}`}>{item.title}</Link>
              </Typography.Text>
              <Typography.Text type="secondary" className="nowrap">{text.assignments.done(item.done, item.total)}</Typography.Text>
            </Flex>
          ))}
        </Flex>
      )}
    </Card>
  )
}

function ContestList({ title, items }: { title: string; items: HomeContest[] }) {
  const { text } = useLocale()
  return (
    <Card title={title}>
      {items.length === 0 ? (
        <EmptyBlock />
      ) : (
        <Flex vertical gap={8}>
          {items.map((item) => (
            <Flex key={item.id} justify="space-between" align="center" gap={12}>
              <Typography.Text strong ellipsis={{ tooltip: item.title }}>
                <Link to={`/contests/${item.id}`}>{item.title}</Link>
              </Typography.Text>
              <Tag color={contestStatusColor(item.status)}>{contestStatusText(item.status, text)}</Tag>
            </Flex>
          ))}
        </Flex>
      )}
    </Card>
  )
}

function contestStatusText(status: string, text: ReturnType<typeof useLocale>['text']) {
  switch (status) {
    case 'pending':
      return text.contests.pending
    case 'running':
      return text.contests.running
    case 'frozen':
      return text.contests.frozen
    case 'ended':
      return text.contests.ended
    default:
      return status
  }
}

function contestStatusColor(status: string) {
  switch (status) {
    case 'pending':
      return 'blue'
    case 'running':
      return 'green'
    case 'frozen':
      return 'orange'
    default:
      return 'default'
  }
}
