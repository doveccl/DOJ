import { EditOutlined } from '@ant-design/icons'
import { App as AntApp, Button, Card, DatePicker, Flex, Form, Input, Modal, Select, Space, Table, Tabs, Tag, Tooltip, Typography } from 'antd'
import type { TableProps } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import dayjs from 'dayjs'
import type { Dayjs } from 'dayjs'
import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'

import { getContest, getProblems, updateContest } from '../client'
import type { Problem, ProblemRef, RankUser, Submission } from '../client'
import { defaultProblemSort, ProblemRefInput } from '../components/problem-ref'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { ContestStatus, SubmissionStatus } from '../components/status'
import { contestTarget, DeadlineTimer } from '../components/time'
import { useLocale } from '../locale'
import { useSession } from '../session'
import { formatTime, problemCode, submissionCode } from '../utils/format'

type ContestForm = {
  title: string
  kind: string
  startAt: Dayjs
  endAt: Dayjs
  freezeAt?: Dayjs | null
  problems?: ProblemRef[]
}

export function ContestDetailPage() {
  const { lang, text } = useLocale()
  const session = useSession()
  const { message } = AntApp.useApp()
  const client = useQueryClient()
  const params = useParams()
  const id = Number(params.id)
  const [editOpen, setEditOpen] = useState(false)
  const query = useQuery({
    queryKey: ['contest', id],
    queryFn: () => getContest(id),
    enabled: Number.isFinite(id)
  })
  const problemsQuery = useQuery({ queryKey: ['problems', '', ''], queryFn: () => getProblems(), enabled: editOpen })
  const showError = (error: unknown) => {
    message.error(error instanceof Error ? error.message : text.common.loadingFailed)
  }
  const update = useMutation({
    mutationFn: (values: ContestForm) =>
      updateContest(id, {
        title: values.title,
        kind: values.kind,
        startAt: values.startAt.toISOString(),
        endAt: values.endAt.toISOString(),
        freezeAt: values.freezeAt?.toISOString() ?? '',
        problems: values.problems ?? []
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['contests'] })
      void client.invalidateQueries({ queryKey: ['contest', id] })
      void client.invalidateQueries({ queryKey: ['home'] })
      setEditOpen(false)
      message.success(text.common.saved)
    },
    onError: showError
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

  const { contest, problems, rank, submissions } = query.data
  const problemOptions = (problemsQuery.data ?? problems).map((item) => ({
    value: item.id,
    label: `${problemCode(item.id)} ${item.title}`
  }))

  function openEdit() {
    setEditOpen(true)
  }

  return (
    <Flex vertical gap={16}>
      <Card>
        <Flex justify="space-between" align="flex-start" gap={20} wrap>
          <Flex vertical gap={8}>
            <Flex align="center" gap={10}>
              <Typography.Title level={3} style={{ margin: 0 }}>
                {contest.title}
              </Typography.Title>
              <ContestStatus status={contest.status} />
              <Tag>{contest.kind}</Tag>
              {session.admin ? (
                <Tooltip title={text.common.edit}>
                  <Button aria-label={`${text.common.edit} #${contest.id}`} type="text" size="small" icon={<EditOutlined />} onClick={openEdit} />
                </Tooltip>
              ) : null}
            </Flex>
          </Flex>
          <Flex vertical align="flex-end" gap={8}>
            <DeadlineTimer
              kind="contest"
              status={contest.status}
              target={contestTarget(contest.status, contest.startAt, contest.endAt)}
              range={`${formatTime(contest.startAt, lang)} - ${formatTime(contest.endAt, lang)}`}
              strong
              align="flex-end"
              onFinish={() => void query.refetch()}
            />
            <Typography.Text type="secondary">{text.contests.total(contest.total)}</Typography.Text>
          </Flex>
        </Flex>
      </Card>
      <Card>
        <Tabs
          items={[
            {
              key: 'problems',
              label: text.contests.problems,
              children: <Table<Problem> rowKey="id" columns={problemColumns(text)} dataSource={problems} pagination={false} />
            },
            {
              key: 'rank',
              label: text.contests.rank,
              children: <Table<RankUser> rowKey="rank" columns={rankColumns(text, contest.kind)} dataSource={rank} pagination={false} />
            },
            {
              key: 'submissions',
              label: text.contests.submissions,
              children: <Table<Submission> rowKey="id" columns={submissionColumns(text, lang)} dataSource={submissions} pagination={false} />
            }
          ]}
        />
      </Card>
      {editOpen ? (
        <ContestEditModal
          contest={contest}
          problems={problems}
          problemOptions={problemOptions}
          problemLoading={problemsQuery.isLoading}
          loading={update.isPending}
          onCancel={() => setEditOpen(false)}
          onSave={(values) => update.mutate(values)}
        />
      ) : null}
    </Flex>
  )
}

function ContestEditModal({
  contest,
  problems,
  problemOptions,
  problemLoading,
  loading,
  onCancel,
  onSave
}: {
  contest: { title: string; kind: string; startAt: string; endAt: string; freezeAt: string | null }
  problems: Problem[]
  problemOptions: { value: number; label: string }[]
  problemLoading: boolean
  loading: boolean
  onCancel: () => void
  onSave: (values: ContestForm) => void
}) {
  const { text } = useLocale()
  const [form] = Form.useForm<ContestForm>()
  const initialValues = {
    title: contest.title,
    kind: contest.kind,
    startAt: dayjs(contest.startAt),
    endAt: dayjs(contest.endAt),
    freezeAt: contest.freezeAt ? dayjs(contest.freezeAt) : null,
    problems: problems.map((problem, index) => ({ id: problem.id, sort: problem.sort || defaultProblemSort(index) }))
  }

  return (
    <Modal
      open
      destroyOnHidden
      title={text.common.edit}
      okText={text.common.save}
      cancelText={text.common.cancel}
      confirmLoading={loading}
      onCancel={onCancel}
      onOk={() => form.submit()}
    >
      <Form<ContestForm> form={form} preserve={false} layout="vertical" initialValues={initialValues} onFinish={onSave}>
        <Form.Item name="title" label={text.contests.name} rules={[{ required: true, whitespace: true }]}>
          <Input maxLength={120} showCount />
        </Form.Item>
        <Form.Item name="kind" label={text.contests.kind}>
          <Select
            options={[
              { value: 'OI', label: 'OI' },
              { value: 'ICPC', label: 'ICPC' }
            ]}
          />
        </Form.Item>
        <Space size={12} style={{ width: '100%' }} align="start">
          <Form.Item name="startAt" label={text.contests.start} rules={[{ required: true }]}>
            <DatePicker showTime />
          </Form.Item>
          <Form.Item name="endAt" label={text.contests.end} rules={[{ required: true }]}>
            <DatePicker showTime />
          </Form.Item>
          <Form.Item name="freezeAt" label={text.contests.freeze}>
            <DatePicker showTime />
          </Form.Item>
        </Space>
        <Form.Item name="problems" label={text.contests.problems}>
          <ProblemRefInput options={problemOptions} loading={problemLoading} />
        </Form.Item>
      </Form>
    </Modal>
  )
}

function rankColumns(text: ReturnType<typeof useLocale>['text'], kind: string): TableProps<RankUser>['columns'] {
  const columns: TableProps<RankUser>['columns'] = [
    {
      title: text.rank.rank,
      dataIndex: 'rank',
      width: 90
    },
    {
      title: text.rank.user,
      render: (_, row) => (
        <Flex vertical gap={2}>
          <Typography.Text strong>{row.user}</Typography.Text>
          <Typography.Text type="secondary" ellipsis>
            {row.bio}
          </Typography.Text>
        </Flex>
      )
    },
    {
      title: text.rank.submit,
      dataIndex: 'submit',
      width: 100
    }
  ]
  if (kind === 'OI') {
    columns.splice(2, 0, {
      title: text.rank.score,
      dataIndex: 'score',
      width: 100
    })
    return columns
  }
  columns.splice(
    2,
    0,
    {
      title: text.rank.ac,
      dataIndex: 'ac',
      width: 100
    },
    {
      title: text.rank.penalty,
      dataIndex: 'penalty',
      width: 100
    }
  )
  return columns
}

function submissionColumns(text: ReturnType<typeof useLocale>['text'], lang: string): TableProps<Submission>['columns'] {
  return [
    {
      title: text.submissions.id,
      dataIndex: 'id',
      width: 110,
      render: (id: number) => <Link to={`/submissions/${id}`}>{submissionCode(id)}</Link>
    },
    {
      title: text.submissions.problem,
      render: (_, row) => (
        <Typography.Text ellipsis className="lineText">
          <Link to={`/problems/${row.problemId}`}>
            {problemCode(row.problemId)} {row.problemTitle}
          </Link>
        </Typography.Text>
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
      title: text.submissions.created,
      dataIndex: 'createdAt',
      width: 180,
      render: (value: string) => <Typography.Text className="nowrap">{formatTime(value, lang)}</Typography.Text>
    }
  ]
}

function problemColumns(text: ReturnType<typeof useLocale>['text']): TableProps<Problem>['columns'] {
  return [
    {
      title: text.common.sort,
      dataIndex: 'sort',
      width: 90,
      render: (sort: string | undefined, row) => <Typography.Text>{sort || problemCode(row.id)}</Typography.Text>
    },
    {
      title: text.problems.id,
      dataIndex: 'id',
      width: 120,
      render: (id: number) => <Typography.Text>{problemCode(id)}</Typography.Text>
    },
    {
      title: text.problems.title,
      dataIndex: 'title',
      render: (title: string, row) => (
        <Typography.Text ellipsis className="lineText">
          <Link to={`/problems/${row.id}`}>{title}</Link>
        </Typography.Text>
      )
    }
  ]
}
