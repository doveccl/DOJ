import { EditOutlined, UnorderedListOutlined } from '@ant-design/icons'
import { App as AntApp, Button, Card, DatePicker, Flex, Form, Input, Modal, Table, Tabs, Tag, Tooltip, Typography } from 'antd'
import type { TableProps } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import dayjs from 'dayjs'
import type { Dayjs } from 'dayjs'
import { useState } from 'react'
import { useParams } from 'react-router-dom'

import { api, apiData } from '../client'
import type { AssignmentProgress, ProblemListItem, ProblemRef, ProblemState } from '../client'
import { ProblemLink, UserLink } from '../components/entity'
import { IdSelect } from '../components/id-select'
import { defaultProblemSort, ProblemRefInput } from '../components/problem-ref'
import { ScoreTag } from '../components/score'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { DeadlineTimer } from '../components/time'
import { useLocale } from '../locale'
import { useSession } from '../session'
import { problemCode, problemLabel } from '../utils/format'
import { limits } from '../utils/limits'

type AssignmentForm = {
  title: string
  endAt: Dayjs
  problems?: ProblemRef[]
  users?: number[]
  groups?: number[]
}

export function AssignmentDetailPage() {
  const { text } = useLocale()
  const session = useSession()
  const { message } = AntApp.useApp()
  const client = useQueryClient()
  const params = useParams()
  const id = Number(params.id)
  const [editOpen, setEditOpen] = useState(false)
  const query = useQuery({
    queryKey: ['assignment', id],
    queryFn: () => apiData(api.GET('/api/assignments/{id}', { params: { path: { id } } })),
    enabled: Number.isFinite(id)
  })
  const problemIds = query.data?.problems.map((item) => item.id).join(',') ?? ''
  const state = useQuery({
    queryKey: ['problem-state', 'assignment', id, problemIds],
    queryFn: () => apiData(api.GET('/api/problem-state', { params: { query: { ids: problemIds, assignment: id } } })),
    enabled: Number.isFinite(id) && problemIds.length > 0
  })
  const showError = (error: unknown) => {
    message.error(error instanceof Error ? error.message : text.common.loadingFailed)
  }
  const update = useMutation({
    mutationFn: (values: AssignmentForm) =>
      apiData(api.PATCH('/api/assignments/{id}', { params: { path: { id } }, body: {
        title: values.title,
        endAt: values.endAt.toISOString(),
        problems: values.problems ?? [],
        users: values.users ?? [],
        groups: values.groups ?? []
      } })),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['assignments'] })
      void client.invalidateQueries({ queryKey: ['assignment', id] })
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

  const { assignment, problems, progress: assignmentProgress } = query.data
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
              {assignment.title}
            </Typography.Title>
            {session.admin ? (
              <Tooltip title={text.common.edit}>
                <Button aria-label={`${text.common.edit} #${assignment.id}`} type="text" size="small" icon={<EditOutlined />} onClick={openEdit} />
              </Tooltip>
            ) : null}
          </Flex>
          <DeadlineTimer kind="assignment" status={assignment.status} target={assignment.endAt} onFinish={() => void query.refetch()} />
        </Flex>
      </Card>
      <Card>
        <Tabs
          items={[
            {
              key: 'problems',
              label: text.assignments.problems,
              children: <Table<ProblemListItem> rowKey="id" columns={problemColumns(text, session.admin, assignment.id, stateByProblem)} dataSource={problems} pagination={false} loading={state.isLoading} scroll={{ x: session.admin ? 640 : 560 }} />
            },
            {
              key: 'progress',
              label: text.assignments.completion,
              children: <Table<AssignmentProgress> rowKey="user" columns={progressColumns(text, problems)} dataSource={assignmentProgress} pagination={false} scroll={{ x: 'max-content' }} />
            }
          ]}
        />
      </Card>
      {editOpen ? (
        <AssignmentEditModal
          assignment={assignment}
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

function AssignmentEditModal({
  assignment,
  problems,
  problemOptions,
  loading,
  onCancel,
  onSave
}: {
  assignment: { title: string; endAt: string; users: number[]; groups: number[] }
  problems: ProblemListItem[]
  problemOptions: { value: number; label: string }[]
  loading: boolean
  onCancel: () => void
  onSave: (values: AssignmentForm) => void
}) {
  const { text } = useLocale()
  const [form] = Form.useForm<AssignmentForm>()
  const initialValues = {
    title: assignment.title,
    endAt: dayjs(assignment.endAt),
    problems: problems.map((problem, index) => ({ id: problem.id, sort: problem.sort || defaultProblemSort(index) })),
    users: assignment.users,
    groups: assignment.groups
  }

  return (
    <Modal
      open
      destroyOnHidden
      width={780}
      title={text.common.edit}
      okText={text.common.save}
      cancelText={text.common.cancel}
      confirmLoading={loading}
      onCancel={onCancel}
      onOk={() => form.submit()}
    >
      <Form<AssignmentForm> form={form} preserve={false} layout="vertical" initialValues={initialValues} onFinish={onSave}>
        <Form.Item name="title" label={text.assignments.name} rules={[{ required: true, whitespace: true }]}>
          <Input maxLength={limits.title} showCount />
        </Form.Item>
        <Form.Item name="endAt" label={text.assignments.deadline} rules={[{ required: true }]}>
          <DatePicker showTime style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item name="problems" label={text.assignments.problems}>
          <ProblemRefInput options={problemOptions} />
        </Form.Item>
        <Form.Item name="users" label={text.assignments.users}>
          <IdSelect kind="users" />
        </Form.Item>
        <Form.Item name="groups" label={text.assignments.groups}>
          <IdSelect kind="groups" />
        </Form.Item>
      </Form>
    </Modal>
  )
}

function progressColumns(text: ReturnType<typeof useLocale>['text'], problems: ProblemListItem[]): TableProps<AssignmentProgress>['columns'] {
  return [
    {
      title: text.rank.user,
      render: (_, row) => <UserLink name={row.user} strong />
    },
    {
      title: text.rank.ac,
      dataIndex: 'ac'
    },
    {
      title: text.rank.submit,
      dataIndex: 'submit'
    },
    ...problems.map((problem) => ({
      key: `problem-${problem.id}`,
      align: 'center' as const,
      title: (
        <Tooltip title={problemLabel(problem.id, problem.title)}>
          <span>{problem.sort || problemCode(problem.id)}</span>
        </Tooltip>
      ),
      render: (_: unknown, row: AssignmentProgress) => {
        const item = row.problems?.find((problemProgress) => problemProgress.problemId === problem.id)
        return item?.status === 'pending' ? <Tag color="processing">{text.submissions.statuses.pending}</Tag> : <ScoreTag score={item?.score} />
      }
    }))
  ]
}

function problemColumns(text: ReturnType<typeof useLocale>['text'], admin: boolean, assignmentID: number, state: Map<number, ProblemState>): TableProps<ProblemListItem>['columns'] {
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
      title: text.assignments.status,
      render: (_, row) => <AssignmentProblemStatus status={state.get(row.id)?.status} text={text} />
    }
  ]
  if (admin) {
    columns.push({
      align: 'right',
      render: (_, row) => (
        <Tooltip title={text.submissions.viewProblemRecords}>
          <Button aria-label={`${text.submissions.viewProblemRecords} ${problemCode(row.id)}`} type="text" icon={<UnorderedListOutlined />} href={`/submissions?assignment=${assignmentID}&problem=${row.id}`} />
        </Tooltip>
      )
    })
  }
  return columns
}

function AssignmentProblemStatus({ status, text }: { status?: string; text: ReturnType<typeof useLocale>['text'] }) {
  if (status === 'pending') {
    return <Tag color="processing">{text.submissions.statuses.pending}</Tag>
  }
  if (status === 'ac') {
    return <Tag color="success">{text.assignments.completed}</Tag>
  }
  if (status === 'tried') {
    return <Tag color="warning">{text.assignments.attempted}</Tag>
  }
  return <Tag>{text.assignments.notCompleted}</Tag>
}

function mapByProblem<T extends { problemId: number }>(items: T[]) {
  return new Map(items.map((item) => [item.problemId, item]))
}
