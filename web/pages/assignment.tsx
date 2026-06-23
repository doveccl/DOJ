import { EditOutlined } from '@ant-design/icons'
import { App as AntApp, Button, Card, DatePicker, Flex, Form, Input, Modal, Progress, Select, Space, Table, Tabs, Tag, Tooltip, Typography } from 'antd'
import type { TableProps } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import dayjs from 'dayjs'
import type { Dayjs } from 'dayjs'
import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'

import { getAssignment, getProblems, updateAssignment } from '../client'
import type { Problem, Submission } from '../client'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { AssignmentStatus, SubmissionStatus } from '../components/status'
import { useLocale } from '../locale'
import { useSession } from '../session'
import { formatLimit, formatPass, formatTime, problemCode, progress } from '../utils/format'

type AssignmentForm = {
  title: string
  desc: string
  endAt: Dayjs
  problems?: number[]
}

export function AssignmentDetailPage() {
  const { lang, text } = useLocale()
  const session = useSession()
  const { message } = AntApp.useApp()
  const client = useQueryClient()
  const params = useParams()
  const id = Number(params.id)
  const [editOpen, setEditOpen] = useState(false)
  const query = useQuery({
    queryKey: ['assignment', id],
    queryFn: () => getAssignment(id),
    enabled: Number.isFinite(id)
  })
  const problemsQuery = useQuery({ queryKey: ['problems', '', ''], queryFn: () => getProblems(), enabled: editOpen })
  const showError = (error: unknown) => {
    message.error(error instanceof Error ? error.message : text.common.loadingFailed)
  }
  const update = useMutation({
    mutationFn: (values: AssignmentForm) =>
      updateAssignment(id, {
        title: values.title,
        desc: values.desc ?? '',
        endAt: values.endAt.toISOString(),
        problems: values.problems ?? []
      }),
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

  const { assignment, problems, submissions } = query.data
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
                {assignment.title}
              </Typography.Title>
              <AssignmentStatus status={assignment.status} />
              {session.admin ? (
                <Tooltip title={text.common.edit}>
                  <Button aria-label={`${text.common.edit} #${assignment.id}`} type="text" size="small" icon={<EditOutlined />} onClick={openEdit} />
                </Tooltip>
              ) : null}
            </Flex>
            <Typography.Text type="secondary">{assignment.desc}</Typography.Text>
          </Flex>
          <Flex align="center" justify="flex-end" gap={16} style={{ minWidth: 'min(420px, 100%)' }}>
            <Typography.Text>{formatTime(assignment.endAt, lang)}</Typography.Text>
            <Flex align="center" gap={10} style={{ width: 180 }}>
              <Progress percent={progress(assignment)} size="small" showInfo={false} />
              <Typography.Text className="nowrap">{text.assignments.done(assignment.done, assignment.total)}</Typography.Text>
            </Flex>
          </Flex>
        </Flex>
      </Card>
      <Card>
        <Tabs
          items={[
            {
              key: 'problems',
              label: text.assignments.problems,
              children: <Table<Problem> rowKey="id" columns={problemColumns(text, assignment.id)} dataSource={problems} pagination={false} />
            },
            {
              key: 'submissions',
              label: text.assignments.submissions,
              children: <Table<Submission> rowKey="id" columns={submissionColumns(text, lang)} dataSource={submissions} pagination={false} />
            }
          ]}
        />
      </Card>
      {editOpen ? (
        <AssignmentEditModal
          assignment={assignment}
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

function AssignmentEditModal({
  assignment,
  problems,
  problemOptions,
  problemLoading,
  loading,
  onCancel,
  onSave
}: {
  assignment: { title: string; desc: string; endAt: string }
  problems: Problem[]
  problemOptions: { value: number; label: string }[]
  problemLoading: boolean
  loading: boolean
  onCancel: () => void
  onSave: (values: AssignmentForm) => void
}) {
  const { text } = useLocale()
  const [form] = Form.useForm<AssignmentForm>()
  const initialValues = {
    title: assignment.title,
    desc: assignment.desc,
    endAt: dayjs(assignment.endAt),
    problems: problems.map((problem) => problem.id)
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
      <Form<AssignmentForm> form={form} preserve={false} layout="vertical" initialValues={initialValues} onFinish={onSave}>
        <Form.Item name="title" label={text.assignments.name} rules={[{ required: true, whitespace: true }]}>
          <Input maxLength={120} showCount />
        </Form.Item>
        <Form.Item name="desc" label={text.assignments.desc}>
          <Input.TextArea rows={3} maxLength={500} showCount />
        </Form.Item>
        <Form.Item name="endAt" label={text.assignments.deadline} rules={[{ required: true }]}>
          <DatePicker showTime style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item name="problems" label={text.assignments.problems}>
          <Select mode="multiple" options={problemOptions} loading={problemLoading} />
        </Form.Item>
      </Form>
    </Modal>
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

function problemColumns(text: ReturnType<typeof useLocale>['text'], assignmentId: number): TableProps<Problem>['columns'] {
  return [
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
          <Link to={`/problems/${row.id}?context=assignment&contextId=${assignmentId}`}>{title}</Link>
        </Typography.Text>
      )
    },
    {
      title: text.problems.tag,
      dataIndex: 'tags',
      width: 220,
      render: (tags: string[]) => (
        <Space size={[0, 4]} wrap>
          {tags.map((tag) => (
            <Tag key={tag}>{tag}</Tag>
          ))}
        </Space>
      )
    },
    {
      title: text.problems.limit,
      render: (_, row) => <Typography.Text type="secondary">{formatLimit(row)}</Typography.Text>
    },
    {
      title: text.problems.pass,
      render: (_, row) => <Typography.Text>{formatPass(row)}</Typography.Text>
    }
  ]
}
