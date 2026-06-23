import { SearchOutlined } from '@ant-design/icons'
import { Button, Card, Flex, Form, Input, Select, Space, Table, Tag, Typography } from 'antd'
import type { TableProps } from 'antd'
import { useQuery } from '@tanstack/react-query'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'

import { getSubmissions } from '../client'
import type { Submission } from '../client'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { SubmissionStatus } from '../components/status'
import { useLocale } from '../locale'
import { formatTime, problemCode } from '../utils/format'

const statusOptions = ['queued', 'judging', 'AC', 'WA', 'TLE', 'MLE', 'RE', 'CE']

export function SubmissionsPage() {
  const { lang, text } = useLocale()
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const problem = params.get('problem') ?? ''
  const status = params.get('status') ?? ''
  const query = useQuery({
    queryKey: ['submissions', problem, status],
    queryFn: () => getSubmissions({ problem, status })
  })

  function submit(values: { problem?: string; status?: string }) {
    const next = new URLSearchParams()
    if (values.problem) {
      next.set('problem', values.problem)
    }
    if (values.status) {
      next.set('status', values.status)
    }
    setParams(next)
  }

  function clear() {
    setParams(new URLSearchParams())
  }

  return (
    <Card>
      <Form layout="inline" initialValues={{ problem: problem || undefined, status: status || undefined }} onFinish={submit} key={`${problem}:${status}`}>
        <Flex align="center" justify="space-between" gap={16} wrap style={{ width: '100%', marginBottom: 18 }}>
          <Flex align="center" gap={10} wrap style={{ flex: '1 1 520px', minWidth: 0 }}>
          <Form.Item name="problem">
            <Input placeholder={text.submissions.searchProblem} allowClear style={{ minWidth: 240 }} />
          </Form.Item>
          <Form.Item name="status">
            <Select
              placeholder={text.submissions.allStatus}
              allowClear
              options={statusOptions.map((item) => ({ label: item, value: item }))}
              style={{ width: 220 }}
            />
          </Form.Item>
          <Button onClick={clear}>{text.common.clear}</Button>
          </Flex>
          <Flex align="center" gap={10} wrap>
          <Button type="primary" htmlType="submit" icon={<SearchOutlined />}>
            {text.common.search}
          </Button>
          </Flex>
        </Flex>
      </Form>
      {query.isError ? (
        <ErrorBlock error={query.error} />
      ) : query.isLoading ? (
        <LoadingBlock />
      ) : (
        <Table<Submission>
          rowKey="id"
          rowClassName="clickableRow"
          onRow={(row) => ({
            onClick: (event) => {
              const target = event.target as HTMLElement
              if (target.closest('a, button, input, textarea, [role="button"], [role="combobox"]')) {
                return
              }
              navigate(`/submissions/${row.id}`)
            }
          })}
          columns={submissionColumns(text, lang)}
          dataSource={query.data}
          pagination={{ pageSize: 20, showSizeChanger: true }}
        />
      )}
    </Card>
  )
}

function submissionColumns(text: ReturnType<typeof useLocale>['text'], lang: string): TableProps<Submission>['columns'] {
  return [
    {
      title: text.submissions.id,
      dataIndex: 'id',
      width: 110,
      render: (id: number) => <Link to={`/submissions/${id}`}>#{id}</Link>
    },
    {
      title: text.submissions.problem,
      render: (_, row) => (
        <Flex align="center" gap={8} className="tableTitleLine">
          <Typography.Text type="secondary" className="nowrap">
            {problemCode(row.problemId)}
          </Typography.Text>
          <Typography.Text ellipsis className="lineText">
            <Link to={`/problems/${row.problemId}`}>{row.problemTitle}</Link>
          </Typography.Text>
        </Flex>
      )
    },
    {
      title: text.submissions.user,
      dataIndex: 'user',
      width: 120,
      render: (user: string) => <Link to={`/users/${user}`}>{user}</Link>
    },
    {
      title: text.submissions.status,
      dataIndex: 'status',
      width: 110,
      render: (status: string) => <SubmissionStatus status={status} />
    },
    {
      title: text.submissions.score,
      dataIndex: 'score',
      width: 90
    },
    {
      title: text.submissions.language,
      dataIndex: 'language',
      width: 120,
      render: (language: string) => <Tag>{language}</Tag>
    },
    {
      title: text.submissions.created,
      dataIndex: 'createdAt',
      width: 180,
      render: (value: string) => <Typography.Text className="nowrap">{formatTime(value, lang)}</Typography.Text>
    }
  ]
}
