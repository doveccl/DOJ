import { EditOutlined } from '@ant-design/icons'
import { App as AntApp, Button, Card, DatePicker, Flex, Form, Input, Modal, Table, Tabs, Tag, Tooltip, Typography } from 'antd'
import type { TableProps } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import dayjs from 'dayjs'
import type { Dayjs } from 'dayjs'
import { useState } from 'react'
import { useParams } from 'react-router-dom'

import { api, apiData } from '../client'
import type { AssignmentProgress, ProblemListItem, ProblemRef } from '../client'
import { ProblemLink, UserLink } from '../components/entity'
import { IdSelect } from '../components/id-select'
import { PlagiarismPanel } from '../components/plagiarism'
import { defaultProblemSort, ProblemRefInput } from '../components/problem-ref'
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
  if (!query.data) {
    return <ErrorBlock error={text.common.emptyResponse} />
  }

  const { assignment, problems, progress: assignmentProgress } = query.data
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
              children: <Table<ProblemListItem> rowKey="id" columns={problemColumns(text)} dataSource={problems} pagination={false} scroll={{ x: 560 }} />
            },
            {
              key: 'progress',
              label: text.assignments.completion,
              children: <Table<AssignmentProgress> rowKey="user" columns={progressColumns(text, problems)} dataSource={assignmentProgress} pagination={false} scroll={{ x: 'max-content' }} />
            },
            ...(session.admin
              ? [{
                  key: 'plagiarism',
                  label: text.plagiarism.title,
                  children: <PlagiarismPanel scope="assignment" id={assignment.id} />
                }]
              : [])
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
    ...problems.map((problem) => ({
      key: `problem-${problem.id}`,
      title: (
        <Tooltip title={problemLabel(problem.id, problem.title)}>
          <span>{problem.sort || problemCode(problem.id)}</span>
        </Tooltip>
      ),
      render: (_: unknown, row: AssignmentProgress) => <AssignmentProblemStatus mine={row.problems?.find((item) => item.problemId === problem.id)?.status} text={text} />
    })),
    {
      title: text.rank.ac,
      dataIndex: 'ac'
    },
    {
      title: text.rank.submit,
      dataIndex: 'submit'
    }
  ]
}

function problemColumns(text: ReturnType<typeof useLocale>['text']): TableProps<ProblemListItem>['columns'] {
  return [
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
      dataIndex: 'mine',
      render: (mine: string) => <AssignmentProblemStatus mine={mine} text={text} />
    }
  ]
}

function AssignmentProblemStatus({ mine, text }: { mine?: string; text: ReturnType<typeof useLocale>['text'] }) {
  if (mine === 'pending') {
    return <Tag color="processing">{text.submissions.statuses.pending}</Tag>
  }
  if (mine === 'ac') {
    return <Tag color="success">{text.assignments.completed}</Tag>
  }
  if (mine === 'tried') {
    return <Tag color="warning">{text.assignments.attempted}</Tag>
  }
  return <Tag>{text.assignments.notCompleted}</Tag>
}
