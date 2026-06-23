import { UploadOutlined } from '@ant-design/icons'
import { App as AntApp, Avatar, Button, Card, Col, Flex, Form, Input, Row, Space, Table, Tabs, Tag, Typography, Upload } from 'antd'
import type { TableProps, UploadProps } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'

import { getMe, getUser, updateMe, updatePassword, uploadImage } from '../client'
import type { MeUpdate, PasswordUpdate, Problem, Submission } from '../client'
import { YearHeatmap } from '../components/heatmap'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { SubmissionStatus } from '../components/status'
import { useLocale } from '../locale'
import { useSession } from '../session'
import { formatPass, formatTime, problemCode } from '../utils/format'

export function ProfilePage() {
  const { lang, text } = useLocale()
  const session = useSession()
  const { message } = AntApp.useApp()
  const client = useQueryClient()
  const query = useQuery({ queryKey: ['me'], queryFn: getMe })
  const activity = useQuery({
    queryKey: ['user', session.name],
    queryFn: () => getUser(session.name),
    enabled: session.signedIn && session.name !== ''
  })
  const account = useMutation({
    mutationFn: updateMe,
    onSuccess: (data) => {
      client.setQueryData(['me'], data)
      void session.refresh()
      message.success(text.common.saved)
    },
    onError: (error) => {
      message.error(error instanceof Error ? error.message : text.common.loadingFailed)
    }
  })
  if (!session.signedIn) {
    return <ErrorBlock error={text.common.forbidden} />
  }

  if (query.isLoading) {
    return <LoadingBlock />
  }
  if (query.isError) {
    return <ErrorBlock error={query.error} />
  }
  if (!query.data) {
    return <ErrorBlock error={text.common.emptyResponse} />
  }

  const me = query.data
  const avatarText = me.name.slice(0, 1).toUpperCase()
  const uploadAvatar: UploadProps['beforeUpload'] = (file) => {
    if (!file.type.startsWith('image/')) {
      message.error(text.profile.avatarImageOnly)
      return Upload.LIST_IGNORE
    }
    void uploadImage(file)
      .then((src) => {
        account.mutate({ mail: me.mail, bio: me.bio, avatar: src })
      })
      .catch((error: unknown) => {
        message.error(error instanceof Error ? error.message : text.common.loadingFailed)
      })
    return false
  }

  return (
    <Flex vertical gap={20} className="pageStack">
      <Card
        title={
          <Space align="center" size={12}>
            <Avatar size={40} src={me.avatar || session.avatar || undefined}>
              {avatarText}
            </Avatar>
            <span>{me.name}</span>
          </Space>
        }
        extra={
          <Upload accept="image/*" beforeUpload={uploadAvatar} showUploadList={false}>
            <Button icon={<UploadOutlined />}>{text.profile.avatar}</Button>
          </Upload>
        }
      >
        <Tabs
          destroyOnHidden
          items={[
            {
              key: 'account',
              label: text.profile.account,
              children: (
                <Form<MeUpdate>
                  layout="vertical"
                  style={{ maxWidth: 560 }}
                  initialValues={{ mail: me.mail, bio: me.bio }}
                  key={`${me.name}:${me.mail}:${me.bio}:${me.avatar}`}
                  onFinish={(values) => account.mutate({ ...values, avatar: me.avatar })}
                >
                  <Form.Item name="mail" label={text.profile.email} rules={[{ type: 'email' }]}>
                    <Input />
                  </Form.Item>
                  <Form.Item name="bio" label={text.profile.bio}>
                    <Input.TextArea maxLength={280} showCount rows={4} />
                  </Form.Item>
                  <Button type="primary" htmlType="submit" loading={account.isPending}>
                    {text.profile.save}
                  </Button>
                </Form>
              )
            },
            {
              key: 'password',
              label: text.profile.password,
              children: <PasswordPane />
            }
          ]}
        />
      </Card>
      {activity.isLoading ? (
        <LoadingBlock />
      ) : activity.isError ? (
        <ErrorBlock error={activity.error} />
      ) : activity.data ? (
        <>
          <Card title={text.home.heatmap}>
            <YearHeatmap cells={activity.data.heatmap} />
          </Card>
          <Row gutter={[20, 20]}>
            <Col xs={24} lg={10}>
              <Card title={text.user.solved}>
                <Table<Problem> rowKey="id" size="small" pagination={false} columns={problemColumns(text)} dataSource={activity.data.solved} />
              </Card>
            </Col>
            <Col xs={24} lg={14}>
              <Card title={text.user.recent}>
                <Table<Submission> rowKey="id" size="small" pagination={false} columns={submissionColumns(text, lang)} dataSource={activity.data.submissions} />
              </Card>
            </Col>
          </Row>
        </>
      ) : null}
    </Flex>
  )
}

function PasswordPane() {
  const { text } = useLocale()
  const { message } = AntApp.useApp()
  const [form] = Form.useForm<PasswordUpdate>()
  const password = useMutation({
    mutationFn: updatePassword,
    onSuccess: () => {
      form.resetFields()
      message.success(text.common.saved)
    },
    onError: (error) => {
      message.error(error instanceof Error ? error.message : text.common.loadingFailed)
    }
  })

  return (
    <Form<PasswordUpdate> form={form} preserve={false} layout="vertical" style={{ maxWidth: 560 }} onFinish={(values) => password.mutate(values)}>
      <Form.Item name="oldPassword" label={text.profile.oldPassword}>
        <Input.Password />
      </Form.Item>
      <Form.Item name="newPassword" label={text.profile.newPassword}>
        <Input.Password />
      </Form.Item>
      <Button type="primary" htmlType="submit" loading={password.isPending}>
        {text.profile.save}
      </Button>
    </Form>
  )
}

function problemColumns(text: ReturnType<typeof useLocale>['text']): TableProps<Problem>['columns'] {
  return [
    {
      title: text.problems.title,
      render: (_, row) => (
        <Typography.Text ellipsis className="lineText">
          <Link to={`/problems/${row.id}`}>
            {problemCode(row.id)} {row.title}
          </Link>
        </Typography.Text>
      )
    },
    {
      title: text.problems.tag,
      dataIndex: 'tags',
      width: 180,
      render: (tags: string[]) => (
        <Space size={[0, 4]} wrap>
          {tags.slice(0, 2).map((tag) => (
            <Tag key={tag}>{tag}</Tag>
          ))}
        </Space>
      )
    },
    {
      title: text.problems.pass,
      width: 120,
      render: (_, row) => formatPass(row)
    }
  ]
}

function submissionColumns(text: ReturnType<typeof useLocale>['text'], lang: string): TableProps<Submission>['columns'] {
  return [
    {
      title: text.submissions.problem,
      render: (_, row) => <Link to={`/problems/${row.problemId}`}>{row.problemTitle}</Link>
    },
    {
      title: text.submissions.status,
      width: 90,
      render: (_, row) => <SubmissionStatus status={row.status} />
    },
    {
      title: text.submissions.created,
      width: 140,
      render: (_, row) => <Typography.Text className="nowrap">{formatTime(row.createdAt, lang)}</Typography.Text>
    }
  ]
}
