import { Avatar, Card, Col, Flex, Row, Space, Table, Tag, Typography } from 'antd'
import type { TableProps } from 'antd'
import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'

import { getUser } from '../client'
import type { Problem, Submission } from '../client'
import { YearHeatmap } from '../components/heatmap'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { SubmissionStatus } from '../components/status'
import { useLocale } from '../locale'
import { formatPass, formatTime, problemCode } from '../utils/format'

export function UserPage() {
  const { lang, text } = useLocale()
  const params = useParams()
  const name = params.name ?? ''
  const query = useQuery({
    queryKey: ['user', name],
    queryFn: () => getUser(name),
    enabled: name !== ''
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

  const profile = query.data
  const user = profile.user

  return (
    <Flex vertical gap={20} className="pageStack">
      <Card>
        <Flex align="center" justify="space-between" gap={20} wrap>
          <Flex align="center" gap={14}>
            <Avatar size={56} src={user.avatar || undefined}>
              {user.name.slice(0, 1).toUpperCase()}
            </Avatar>
            <Flex vertical gap={4}>
              <Space size={8}>
                <Typography.Title level={3} style={{ margin: 0 }}>
                  {user.name}
                </Typography.Title>
                {user.admin ? <Tag color="cyan">{text.admin.roles.admin}</Tag> : null}
              </Space>
              <Typography.Text>{user.bio || text.user.noBio}</Typography.Text>
            </Flex>
          </Flex>
          <Space size={24}>
            <Stat label={text.rank.ac} value={user.ac} />
            <Stat label={text.rank.submit} value={user.submit} />
          </Space>
        </Flex>
      </Card>
      <Card title={text.home.heatmap}>
        <YearHeatmap cells={profile.heatmap} />
      </Card>
      <Row gutter={[20, 20]}>
        <Col xs={24} lg={10}>
          <Card title={text.user.solved}>
            <Table<Problem>
              rowKey="id"
              size="small"
              pagination={false}
              columns={problemColumns(text)}
              dataSource={profile.solved}
            />
          </Card>
        </Col>
        <Col xs={24} lg={14}>
          <Card title={text.user.recent}>
            <Table<Submission>
              rowKey="id"
              size="small"
              pagination={false}
              columns={submissionColumns(text, lang)}
              dataSource={profile.submissions}
            />
          </Card>
        </Col>
      </Row>
    </Flex>
  )
}

function Stat({ label, value }: { label: string; value: number }) {
  return (
    <Flex vertical align="end">
      <Typography.Text type="secondary">{label}</Typography.Text>
      <Typography.Title level={3} style={{ margin: 0 }}>
        {value}
      </Typography.Title>
    </Flex>
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
      title: text.submissions.id,
      width: 90,
      render: (_, row) => <Link to={`/submissions/${row.id}`}>#{row.id}</Link>
    },
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
