import { EditOutlined, UnorderedListOutlined } from '@ant-design/icons'
import { Alert, App as AntApp, Button, Card, DatePicker, Flex, Form, Input, Modal, Select, Space, Table, Tabs, Tag, Tooltip, Typography } from 'antd'
import type { TableProps } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import dayjs from 'dayjs'
import type { Dayjs } from 'dayjs'
import { useState } from 'react'
import type { ReactNode } from 'react'
import { Link, useParams } from 'react-router-dom'

import { api, apiData } from '../client'
import type { ProblemListItem, ProblemRef, ProblemState, RankUser } from '../client'
import { ProblemLink, UserLink } from '../components/entity'
import { defaultProblemSort, ProblemRefInput } from '../components/problem-ref'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { contestTarget, DeadlineTimer } from '../components/time'
import { useLocale } from '../locale'
import { useSession } from '../session'
import { formatTime, problemCode, problemLabel } from '../utils/format'
import { limits } from '../utils/limits'

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
    queryFn: () => apiData(api.GET('/api/contests/{id}', { params: { path: { id } } })),
    enabled: Number.isFinite(id)
  })
  const problemIds = query.data?.problems.map((item) => item.id).join(',') ?? ''
  const state = useQuery({
    queryKey: ['problem-state', 'contest', id, problemIds],
    queryFn: () => apiData(api.GET('/api/problem-state', { params: { query: { ids: problemIds, contest: id } } })),
    enabled: Number.isFinite(id) && problemIds.length > 0
  })
  const showError = (error: unknown) => {
    message.error(error instanceof Error ? error.message : text.common.loadingFailed)
  }
  const update = useMutation({
    mutationFn: (values: ContestForm) =>
      apiData(api.PATCH('/api/contests/{id}', { params: { path: { id } }, body: {
        title: values.title,
        kind: values.kind,
        startAt: values.startAt.toISOString(),
        endAt: values.endAt.toISOString(),
        freezeAt: values.freezeAt?.toISOString() ?? '',
        problems: values.problems ?? []
      } })),
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
  if (state.isError) {
    return <ErrorBlock error={state.error} />
  }
  if (!query.data) {
    return <ErrorBlock error={text.common.emptyResponse} />
  }

  const { contest, problems, rank } = query.data
  const stateByProblem = mapByProblem(state.data ?? [])
  const problemOptions = problems.map((item) => ({
    value: item.id,
    label: problemLabel(item.id, item.title)
  }))

  function openEdit() {
    setEditOpen(true)
  }

  return (
    <Flex vertical gap={16}>
      <Card>
        <Flex justify="space-between" align="center" gap={20} wrap>
          <Flex align="center" gap={10}>
            <Typography.Title level={3} style={{ margin: 0 }}>
              {contest.title}
            </Typography.Title>
            <Tag>{contest.kind}</Tag>
            {session.admin ? (
              <Tooltip title={text.common.edit}>
                <Button aria-label={`${text.common.edit} #${contest.id}`} type="text" size="small" icon={<EditOutlined />} onClick={openEdit} />
              </Tooltip>
            ) : null}
          </Flex>
          <DeadlineTimer
            kind="contest"
            status={contest.status}
            target={contestTarget(contest.status, contest.startAt, contest.endAt)}
            range={`${formatTime(contest.startAt, lang)} - ${formatTime(contest.endAt, lang)}`}
            onFinish={() => void query.refetch()}
          />
        </Flex>
      </Card>
      <Card>
        <Tabs
          items={[
            {
              key: 'problems',
              label: text.contests.problems,
              children: <Table<ProblemListItem> rowKey="id" columns={problemColumns(text, session.admin, contest.id, stateByProblem)} dataSource={problems} pagination={false} loading={state.isLoading} scroll={{ x: session.admin ? 640 : 560 }} />
            },
            {
              key: 'rank',
              label: text.contests.rank,
              children: (
                <Flex vertical gap={12}>
                  {contest.status === 'frozen' ? <Alert type={session.admin ? 'info' : 'warning'} showIcon title={session.admin ? text.contests.realtimeRank : text.contests.frozenRank} /> : null}
                  <Table<RankUser> rowKey="rank" columns={rankColumns(text, contest.kind, problems)} dataSource={rank} pagination={false} scroll={{ x: 'max-content' }} />
                </Flex>
              )
            }
          ]}
        />
      </Card>
      {editOpen ? (
        <ContestEditModal
          contest={contest}
          problems={problems}
          problemOptions={problemOptions}
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
  loading,
  onCancel,
  onSave
}: {
  contest: { title: string; kind: string; startAt: string; endAt: string; freezeAt: string | null }
  problems: ProblemListItem[]
  problemOptions: { value: number; label: string }[]
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
          <Input maxLength={limits.title} showCount />
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
          <ProblemRefInput options={problemOptions} />
        </Form.Item>
      </Form>
    </Modal>
  )
}

function rankColumns(text: ReturnType<typeof useLocale>['text'], kind: string, problems: ProblemListItem[]): TableProps<RankUser>['columns'] {
  const columns: NonNullable<TableProps<RankUser>['columns']> = [
    {
      title: text.rank.rank,
      dataIndex: 'rank'
    },
    {
      title: text.rank.user,
      render: (_, row) => <UserLink name={row.user} strong />
    }
  ]
  if (kind === 'OI') {
    columns.splice(2, 0, {
      title: text.rank.score,
      dataIndex: 'score'
    })
    return [...columns, ...rankProblemColumns(kind, problems)]
  }
  columns.splice(
    2,
    0,
    {
      title: text.rank.ac,
      dataIndex: 'ac'
    },
    {
      title: text.rank.penalty,
      dataIndex: 'penalty'
    }
  )
  return [...columns, ...rankProblemColumns(kind, problems)]
}

function rankProblemColumns(kind: string, problems: ProblemListItem[]): NonNullable<TableProps<RankUser>['columns']> {
  return problems.map((problem) => ({
    key: `problem-${problem.id}`,
    align: 'center',
    title: (
      <Tooltip title={problemLabel(problem.id, problem.title)}>
        <span>{problem.sort || problemCode(problem.id)}</span>
      </Tooltip>
    ),
    render: (_, row) => <RankProblemCell kind={kind} item={row.problems.find((item) => item.problemId === problem.id)} />
  }))
}

function RankProblemCell({ kind, item }: { kind: string; item?: RankUser['problems'][number] }) {
  if (!item || item.status === 'none') {
    return <Typography.Text type="secondary">-</Typography.Text>
  }
  if (kind === 'ICPC') {
    if (item.status === 'pending') {
      return <Tag color="processing">{item.submit > 0 ? `?${item.submit}` : '?'}</Tag>
    }
    if (item.status === 'ac') {
      const wrong = Math.max(0, item.submit - 1)
      return (
        <Space orientation="vertical" size={0} align="center">
          <Tag color="success">{wrong > 0 ? `+${wrong}` : '+'}</Tag>
          <Typography.Text type="secondary">{item.penalty}</Typography.Text>
        </Space>
      )
    }
    return <Tag color="error">-{item.submit}</Tag>
  }
  if (item.status === 'ac') {
    return <Tag color="success">{item.score}</Tag>
  }
  return <Tag color={item.score > 0 ? 'warning' : 'error'}>{item.score}</Tag>
}

function problemColumns(text: ReturnType<typeof useLocale>['text'], admin: boolean, contestID: number, state: Map<number, ProblemState>): TableProps<ProblemListItem>['columns'] {
  const columns: TableProps<ProblemListItem>['columns'] = [
    {
      title: text.common.sort,
      dataIndex: 'sort',
      render: (sort: string | undefined, row) => <Typography.Text>{sort || problemCode(row.id)}</Typography.Text>
    },
    {
      title: text.submissions.problem,
      dataIndex: 'title',
      width: 320,
      ellipsis: { showTitle: false },
      render: (title: string, row) => <ProblemLink id={row.id} title={title} />
    },
    {
      title: text.contests.status,
      render: (_, row) => <ContestProblemStatus record={state.get(row.id)} text={text} />
    }
  ]
  if (admin) {
    columns.push({
      align: 'right',
      render: (_, row) => (
        <Tooltip title={text.submissions.viewProblemRecords}>
          <Button aria-label={`${text.submissions.viewProblemRecords} ${problemCode(row.id)}`} type="text" icon={<UnorderedListOutlined />} href={`/submissions?contest=${contestID}&problem=${row.id}`} />
        </Tooltip>
      )
    })
  }
  return columns
}

function ContestProblemStatus({ record, text }: { record?: ProblemState; text: ReturnType<typeof useLocale>['text'] }) {
  const status = record?.status
  const wrap = (node: ReactNode) => record?.submission?.id ? <Link to={`/submissions/${record.submission.id}`}>{node}</Link> : node
  if (status === 'pending') {
    return wrap(<Tag color="processing">{text.submissions.statuses.pending}</Tag>)
  }
  if (status === 'ac') {
    return wrap(<Tag color="success">{text.contests.completed}</Tag>)
  }
  if (status === 'tried') {
    return wrap(<Tag color="warning">{text.contests.attempted}</Tag>)
  }
  return <Tag>{text.contests.notCompleted}</Tag>
}

function mapByProblem<T extends { problemId: number }>(items: T[]) {
  return new Map(items.map((item) => [item.problemId, item]))
}
