import { UnorderedListOutlined } from '@ant-design/icons'
import { App as AntApp, Button, Card, Flex, Form, Table, Tabs, Tag, Tooltip, Typography } from 'antd'
import type { TableProps } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import dayjs from 'dayjs'
import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'

import { api, apiData } from '../../client'
import type { AssignmentProgress, ProblemListItem, ProblemState } from '../../client'
import { ScheduledDetailHeader } from '../../components/scheduled-detail-header'
import { ProblemLink, UserLink } from '../../components/entity'
import { defaultProblemSort } from '../../components/problem-ref'
import { ErrorBlock, LoadingBlock } from '../../components/state'
import { ScheduleTag } from '../../components/time'
import { useLocale } from '../../locale'
import { useSession } from '../../session'
import { problemCode, problemLabel } from '../../utils/format'
import { AssignmentFormFields } from './form'
import type { AssignmentFormValues } from './form'

export function AssignmentDetailPage() {
  const { text } = useLocale()
  const session = useSession()
  const { message, modal } = AntApp.useApp()
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
    mutationFn: (values: AssignmentFormValues) =>
      apiData(api.PATCH('/api/assignments/{id}', { params: { path: { id } }, body: {
        title: values.title,
        description: values.description,
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
  const scopeRisk = assignment.status === 'ended' || assignmentProgress.some((item) => item.submit > 0)
  const recordsUser = session.signedIn && !session.admin ? session.name : ''
  const stateByProblem = new Map((state.data ?? []).map((item) => [item.problemId, item]))
  const problemOptions = problems.map((item) => ({
    value: item.id,
    label: problemLabel(item.id, item.title)
  }))
  const editFormId = `assignment-${assignment.id}-edit-form`
  const save = (values: AssignmentFormValues) => {
    if (!scopeRisk) {
      update.mutate(values)
      return
    }
    modal.confirm({
      title: text.assignments.changeWarning,
      content: text.assignments.changeWarningDescription,
      okText: text.common.save,
      cancelText: text.common.cancel,
      onOk: () => update.mutate(values)
    })
  }
  return (
    <Flex vertical gap={16}>
      <ScheduledDetailHeader
        descriptionId={`assignment-${assignment.id}-description`}
        descriptionValue={query.data.description}
        status={<ScheduleTag kind="assignment" status={assignment.status} target={assignment.endAt} onFinish={() => void query.refetch()} />}
        title={assignment.title}
        recordsHref={`/submissions?assignment=${assignment.id}${recordsUser ? `&user=${encodeURIComponent(recordsUser)}` : ''}`}
        recordsLabel={recordsUser ? text.submissions.myRecords : text.submissions.allRecords}
        admin={session.admin}
        editing={editOpen}
        onStartEdit={() => setEditOpen(true)}
        onCancelEdit={() => setEditOpen(false)}
        saving={update.isPending}
        editFormId={editFormId}
      >
        <Form<AssignmentFormValues>
          id={editFormId}
          key={editFormId}
          preserve={false}
          layout="vertical"
          initialValues={{
            title: assignment.title,
            description: query.data.description,
            endAt: dayjs(assignment.endAt),
            problems: problems.map((problem, index) => ({ id: problem.id, sort: problem.sort || defaultProblemSort(index) })),
            users: assignment.users,
            groups: assignment.groups
          }}
          onFinish={save}
        >
          <AssignmentFormFields editorId={`assignment-${assignment.id}-description-edit`} problemOptions={problemOptions} />
        </Form>
      </ScheduledDetailHeader>
      <Card>
        <Tabs
          items={[
            {
              key: 'problems',
              label: text.assignments.problems,
              children: <Table<ProblemListItem> rowKey="id" columns={problemColumns(text, assignment.id, assignment.status === 'ended', recordsUser, stateByProblem)} dataSource={problems} pagination={false} loading={state.isLoading} scroll={{ x: 640 }} />
            },
            {
              key: 'progress',
              label: text.assignments.completion,
              children: <Table<AssignmentProgress> rowKey="user" columns={progressColumns(text, problems)} dataSource={assignmentProgress} pagination={false} scroll={{ x: 'max-content' }} />
            }
          ]}
        />
      </Card>
    </Flex>
  )
}

function progressColumns(text: ReturnType<typeof useLocale>['text'], problems: ProblemListItem[]): TableProps<AssignmentProgress>['columns'] {
  return [
    {
      title: text.rank.user,
      width: 220,
      render: (_, row) => <UserLink name={row.user} avatar={row.avatar} />
    },
    {
      title: text.rank.ac,
      dataIndex: 'ac',
      width: 80,
      align: 'center'
    },
    {
      title: text.rank.submit,
      dataIndex: 'submit',
      width: 80,
      align: 'center'
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
        const score = item?.score
        if (item?.status === 'pending') return <Tag color="processing">{text.submissions.statuses.pending}</Tag>
        if (score === undefined) return <Typography.Text type="secondary">-</Typography.Text>
        return <Tag color={score >= 100 ? 'success' : score > 0 ? 'warning' : 'error'}>{score}</Tag>
      }
    }))
  ]
}

function problemColumns(text: ReturnType<typeof useLocale>['text'], assignmentID: number, assignmentEnded: boolean, recordsUser: string, state: Map<number, ProblemState>): TableProps<ProblemListItem>['columns'] {
  const columns: TableProps<ProblemListItem>['columns'] = [
    {
      title: text.assignments.status,
      width: 140,
      render: (_, row) => <AssignmentProblemStatus record={state.get(row.id)} text={text} />
    },
    {
      title: text.common.sort,
      dataIndex: 'sort',
      width: 88,
      render: (sort: string | undefined, row) => <Typography.Text>{sort || problemCode(row.id)}</Typography.Text>
    },
    {
      title: text.submissions.problem,
      dataIndex: 'title',
      ellipsis: { showTitle: false },
      render: (title: string, row) => <ProblemLink id={row.id} title={title} search={assignmentEnded ? '' : `?assignment=${assignmentID}`} />
    },
    {
      title: text.common.actions,
      width: 80,
      align: 'right',
      render: (_, row) => (
        <Tooltip title={text.submissions.viewProblemRecords}>
          <Button aria-label={`${text.submissions.viewProblemRecords} ${problemCode(row.id)}`} type="text" icon={<UnorderedListOutlined />} href={`/submissions?assignment=${assignmentID}&problem=${row.id}${recordsUser ? `&user=${encodeURIComponent(recordsUser)}` : ''}`} />
        </Tooltip>
      )
    }
  ]
  return columns
}

function AssignmentProblemStatus({ record, text }: { record?: ProblemState; text: ReturnType<typeof useLocale>['text'] }) {
  const status = record?.status
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
