import { EditOutlined } from '@ant-design/icons'
import { App as AntApp, Button, Card, Col, Divider, Flex, Form, Row, Space, Tag, Typography } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link } from 'react-router-dom'

import { getHome, updateNotice } from '../client'
import type { Item, Problem } from '../client'
import { ProblemLink } from '../components/entity'
import { YearHeatmap } from '../components/heatmap'
import { MarkdownEditor, MarkdownPreview } from '../components/markdown'
import { EmptyBlock, ErrorBlock, LoadingBlock } from '../components/state'
import { useLocale } from '../locale'
import { useSession } from '../session'
import { formatPass } from '../utils/format'

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
  const query = useQuery({ queryKey: ['home'], queryFn: getHome })
  const showError = (error: unknown) => {
    message.error(error instanceof Error ? error.message : text.common.loadingFailed)
  }
  const notice = useMutation({
    mutationFn: (values: NoticeForm) => updateNotice({ content: values.content }),
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
                <Button size="small" onClick={() => setNoticeEditing(false)}>
                  {text.common.cancel}
                </Button>
                <Button size="small" type="primary" loading={notice.isPending} onClick={() => noticeForm.submit()}>
                  {text.common.save}
                </Button>
              </Space>
            ) : (
              <Button
                size="small"
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
              <MarkdownEditor minHeight={320} trust="trusted" />
            </Form.Item>
          </Form>
        ) : home.notice ? (
          <MarkdownPreview value={home.notice} trust="trusted" />
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
          <ItemList title={text.home.assignments} items={home.assignments} hrefPrefix="/assignments" />
        </Col>
        <Col xs={24} lg={8}>
          <ItemList title={text.home.contests} items={home.contests} hrefPrefix="/contests" />
        </Col>
      </Row>
    </Flex>
  )
}

function ProblemList({ title, items }: { title: string; items: Problem[] }) {
  return (
    <Card title={title}>
      {items.length === 0 ? (
        <EmptyBlock />
      ) : (
        <Flex vertical>
          {items.map((item, index) => (
            <Flex vertical gap={8} key={item.id}>
              {index > 0 ? <Divider style={{ margin: 0 }} /> : null}
              <Flex vertical gap={8}>
                <ProblemLink id={item.id} title={item.title} strong />
                <Flex gap={8} wrap>
                  {item.tags.slice(0, 2).map((tag) => (
                    <Tag key={tag}>{tag}</Tag>
                  ))}
                  <Typography.Text type="secondary">{formatPass(item)}</Typography.Text>
                </Flex>
              </Flex>
            </Flex>
          ))}
        </Flex>
      )}
    </Card>
  )
}

function ItemList({ title, items, hrefPrefix }: { title: string; items: Item[]; hrefPrefix: string }) {
  return (
    <Card title={title}>
      {items.length === 0 ? (
        <EmptyBlock />
      ) : (
        <Flex vertical>
          {items.map((item, index) => (
            <Flex vertical gap={8} key={item.id}>
              {index > 0 ? <Divider style={{ margin: 0 }} /> : null}
              <Typography.Text strong ellipsis className="lineText">
                <Link to={`${hrefPrefix}/${item.id}`}>{item.title}</Link>
              </Typography.Text>
              <Typography.Text type="secondary">{item.meta}</Typography.Text>
            </Flex>
          ))}
        </Flex>
      )}
    </Card>
  )
}
