import { DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons'
import {
  App as AntApp,
  Button,
  Card,
  DatePicker,
  Flex,
  Form,
  Input,
  Modal,
  Popconfirm,
  Progress,
  Space,
  Table,
  Tooltip,
  Typography
} from 'antd'
import type { TableProps } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import dayjs from 'dayjs'
import type { Dayjs } from 'dayjs'
import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import { createAssignment, deleteAssignment, getAssignment, getAssignments, updateAssignment } from '../client'
import type { AssignmentListItem, ProblemRef } from '../client'
import { IdSelect } from '../components/id-select'
import { defaultProblemSort, ProblemRefInput } from '../components/problem-ref'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { ScheduleTag } from '../components/time'
import { useLocale } from '../locale'
import { useSession } from '../session'
import { problemLabel, progress } from '../utils/format'
import { limits } from '../utils/limits'

type AssignmentForm = {
  title: string
  endAt: Dayjs
  problems?: ProblemRef[]
  users?: number[]
  groups?: number[]
}

export function AssignmentsPage() {
  const { text } = useLocale()
  const session = useSession()
  const { message } = AntApp.useApp()
  const client = useQueryClient()
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)
  const query = useQuery({ queryKey: ['assignments'], queryFn: () => getAssignments() })
  const showError = (error: unknown) => {
    message.error(error instanceof Error ? error.message : text.common.loadingFailed)
  }
  const create = useMutation({
    mutationFn: (values: AssignmentForm) =>
      createAssignment({
        title: values.title,
        endAt: values.endAt.toISOString(),
        problems: values.problems ?? [],
        users: values.users ?? [],
        groups: values.groups ?? []
      }),
    onSuccess: (item) => {
      void client.invalidateQueries({ queryKey: ['assignments'] })
      void client.invalidateQueries({ queryKey: ['home'] })
      message.success(text.common.saved)
      closeModal()
      navigate(`/assignments/${item.id}`)
    },
    onError: showError
  })
  const update = useMutation({
    mutationFn: (values: AssignmentForm) => {
      if (!editingId) {
        throw new Error(text.common.emptyResponse)
      }
      return updateAssignment(editingId, {
        title: values.title,
        endAt: values.endAt.toISOString(),
        problems: values.problems ?? [],
        users: values.users ?? [],
        groups: values.groups ?? []
      })
    },
    onSuccess: (item) => {
      void client.invalidateQueries({ queryKey: ['assignments'] })
      void client.invalidateQueries({ queryKey: ['assignment', item.id] })
      void client.invalidateQueries({ queryKey: ['home'] })
      message.success(text.common.saved)
      closeModal()
    },
    onError: showError
  })
  const remove = useMutation({
    mutationFn: deleteAssignment,
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['assignments'] })
      void client.invalidateQueries({ queryKey: ['home'] })
      message.success(text.common.saved)
    },
    onError: showError
  })
  function openCreate() {
    setEditingId(null)
    setOpen(true)
  }

  function openEdit(item: AssignmentListItem) {
    setEditingId(item.id)
    setOpen(true)
  }

  function closeModal() {
    setOpen(false)
    setEditingId(null)
  }

  function save(values: AssignmentForm) {
    if (editingId) {
      update.mutate(values)
      return
    }
    create.mutate(values)
  }

  return (
    <Card>
      <Flex justify="flex-end" style={{ marginBottom: 18 }}>
        {session.admin ? (
          <Button icon={<PlusOutlined />} onClick={openCreate}>
            {text.assignments.create}
          </Button>
        ) : null}
      </Flex>
      {query.isError ? (
        <ErrorBlock error={query.error} />
      ) : query.isLoading ? (
        <LoadingBlock />
      ) : (
        <Table<AssignmentListItem>
          rowKey="id"
          columns={assignmentColumns(
            text,
            {
              edit: openEdit,
              remove: (id) => remove.mutate(id),
              refresh: () => void query.refetch()
            },
            session.admin
          )}
          dataSource={query.data}
          pagination={{ pageSize: 20, showSizeChanger: true }}
        />
      )}
      {session.admin && open ? (
        <AssignmentModal
	          editingId={editingId}
	          loading={create.isPending || update.isPending}
	          onCancel={closeModal}
          onSave={save}
        />
      ) : null}
    </Card>
  )
}

function AssignmentModal({
  editingId,
  loading,
  onCancel,
  onSave
}: {
  editingId: number | null
  loading: boolean
  onCancel: () => void
  onSave: (values: AssignmentForm) => void
}) {
  const { text } = useLocale()
  const [form] = Form.useForm<AssignmentForm>()
  const isEdit = editingId !== null
  const detail = useQuery({
    queryKey: ['assignment', editingId],
    queryFn: () => getAssignment(editingId ?? 0),
    enabled: isEdit
  })

  const initialValues: Partial<AssignmentForm> =
    isEdit && detail.data
      ? {
          title: detail.data.assignment.title,
          endAt: dayjs(detail.data.assignment.endAt),
          problems: detail.data.problems.map((problem, index) => ({ id: problem.id, sort: problem.sort || defaultProblemSort(index) })),
          users: detail.data.assignment.users,
          groups: detail.data.assignment.groups
        }
      : { title: '', problems: [], users: [], groups: [] }
  const problemOptions = (detail.data?.problems ?? []).map((item) => ({
    value: item.id,
    label: problemLabel(item.id, item.title)
  }))

  const body =
    isEdit && detail.isLoading ? (
      <LoadingBlock />
    ) : isEdit && detail.isError ? (
      <ErrorBlock error={detail.error} />
    ) : (
      <Form<AssignmentForm>
        key={isEdit ? `assignment-${editingId}` : 'assignment-new'}
        form={form}
        preserve={false}
        layout="vertical"
        initialValues={initialValues}
        onFinish={onSave}
      >
        <Form.Item name="title" label={text.assignments.name} rules={[{ required: true, whitespace: true }]}>
          <Input maxLength={limits.title} showCount />
        </Form.Item>
        <Form.Item name="endAt" label={text.assignments.deadline} rules={[{ required: true }]}>
          <DatePicker showTime style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item name="problems" label={text.assignments.problems}>
          <ProblemRefInput options={problemOptions} loading={detail.isLoading} />
        </Form.Item>
	        <Form.Item name="users" label={text.assignments.users}>
	          <IdSelect kind="users" />
	        </Form.Item>
	        <Form.Item name="groups" label={text.assignments.groups}>
	          <IdSelect kind="groups" />
	        </Form.Item>
      </Form>
    )

  return (
    <Modal
      open
      destroyOnHidden
      width={780}
      title={isEdit ? text.common.edit : text.assignments.create}
      okText={isEdit ? text.common.save : text.common.create}
      cancelText={text.common.cancel}
      confirmLoading={loading}
      onCancel={onCancel}
      onOk={() => form.submit()}
      okButtonProps={{ disabled: isEdit && !detail.data }}
    >
      {body}
    </Modal>
  )
}

function assignmentColumns(
  text: ReturnType<typeof useLocale>['text'],
  actions: {
    edit: (item: AssignmentListItem) => void
    remove: (id: number) => void
    refresh: () => void
  },
  admin: boolean
): TableProps<AssignmentListItem>['columns'] {
  const columns: TableProps<AssignmentListItem>['columns'] = [
    {
      title: text.assignments.name,
      dataIndex: 'title',
      render: (title: string, row) => (
        <Flex align="center" gap={8} className="tableTitleLine">
          <Typography.Text ellipsis className="lineText">
            <Link to={`/assignments/${row.id}`}>{title}</Link>
          </Typography.Text>
        </Flex>
      )
    },
    {
      title: text.assignments.status,
      render: (_, row) => <ScheduleTag kind="assignment" status={row.status} target={row.endAt} onFinish={actions.refresh} />
    },
    {
      title: text.assignments.progress,
      render: (_, row) => (
        <Flex align="center" gap={10} style={{ minWidth: 160 }}>
          <Progress percent={progress(row)} size="small" showInfo={false} style={{ minWidth: 96 }} />
          <Typography.Text className="nowrap">{text.assignments.done(row.done, row.total)}</Typography.Text>
        </Flex>
      )
    }
  ]
  if (admin) {
    columns.push({
      title: text.common.actions,
      align: 'right',
      render: (_, row) => (
        <Space size={4}>
          <Tooltip title={text.common.edit}>
            <Button aria-label={`${text.common.edit} #${row.id}`} type="text" icon={<EditOutlined />} onClick={() => actions.edit(row)} />
          </Tooltip>
          <Popconfirm title={text.common.confirmDelete} okText={text.common.delete} cancelText={text.common.cancel} onConfirm={() => actions.remove(row.id)}>
            <Button aria-label={`${text.common.delete} #${row.id}`} type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      )
    })
  }
  return columns
}
