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
  Select,
  Space,
  Table,
  Tooltip,
  Typography
} from 'antd'
import type { TableProps } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import dayjs from 'dayjs'
import type { Dayjs } from 'dayjs'
import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import { createAssignment, deleteAssignment, getAssignment, getAssignments, getProblems, updateAssignment } from '../client'
import type { Assignment } from '../client'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { AssignmentStatus } from '../components/status'
import { useLocale } from '../locale'
import { useSession } from '../session'
import { formatTime, progress } from '../utils/format'

type AssignmentForm = {
  title: string
  desc: string
  endAt: Dayjs
  problems?: number[]
}

export function AssignmentsPage() {
  const { lang, text } = useLocale()
  const session = useSession()
  const { message } = AntApp.useApp()
  const client = useQueryClient()
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<Assignment | null>(null)
  const query = useQuery({ queryKey: ['assignments'], queryFn: getAssignments })
  const problems = useQuery({ queryKey: ['problems', '', ''], queryFn: () => getProblems() })
  const showError = (error: unknown) => {
    message.error(error instanceof Error ? error.message : text.common.loadingFailed)
  }
  const create = useMutation({
    mutationFn: (values: AssignmentForm) =>
      createAssignment({
        title: values.title,
        desc: values.desc ?? '',
        endAt: values.endAt.toISOString(),
        problems: values.problems ?? []
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
      if (!editing) {
        throw new Error(text.common.emptyResponse)
      }
      return updateAssignment(editing.id, {
        title: values.title,
        desc: values.desc ?? '',
        endAt: values.endAt.toISOString(),
        problems: values.problems ?? []
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
  const problemOptions = (problems.data ?? []).map((item) => ({
    value: item.id,
    label: `${item.id} ${item.title}`
  }))

  function openCreate() {
    setEditing(null)
    setOpen(true)
  }

  function openEdit(item: Assignment) {
    setEditing(item)
    setOpen(true)
  }

  function closeModal() {
    setOpen(false)
    setEditing(null)
  }

  function save(values: AssignmentForm) {
    if (editing) {
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
        <Table<Assignment>
          rowKey="id"
          columns={assignmentColumns(text, lang, {
            edit: openEdit,
            remove: (id) => remove.mutate(id)
          }, session.admin)}
          dataSource={query.data}
          pagination={{ pageSize: 20, showSizeChanger: true }}
        />
      )}
      {session.admin && open ? (
        <AssignmentModal
          editing={editing}
          loading={create.isPending || update.isPending}
          problemOptions={problemOptions}
          problemLoading={problems.isLoading}
          onCancel={closeModal}
          onSave={save}
        />
      ) : null}
    </Card>
  )
}

function AssignmentModal({
  editing,
  loading,
  problemOptions,
  problemLoading,
  onCancel,
  onSave
}: {
  editing: Assignment | null
  loading: boolean
  problemOptions: { value: number; label: string }[]
  problemLoading: boolean
  onCancel: () => void
  onSave: (values: AssignmentForm) => void
}) {
  const { text } = useLocale()
  const [form] = Form.useForm<AssignmentForm>()
  const detail = useQuery({
    queryKey: ['assignment', editing?.id],
    queryFn: () => getAssignment(editing?.id ?? 0),
    enabled: !!editing
  })

  useEffect(() => {
    if (editing) {
      form.setFieldsValue({
        title: editing.title,
        desc: editing.desc,
        endAt: dayjs(editing.endAt),
        problems: detail.data?.problems.map((problem) => problem.id)
      })
      return
    }
    form.setFieldsValue({ title: '', desc: '', problems: [] })
  }, [detail.data, editing, form])

  return (
    <Modal
      open
      destroyOnHidden
      title={editing ? text.common.edit : text.assignments.create}
      okText={editing ? text.common.save : text.common.create}
      cancelText={text.common.cancel}
      confirmLoading={loading}
      onCancel={onCancel}
      onOk={() => form.submit()}
    >
      <Form<AssignmentForm> form={form} preserve={false} layout="vertical" onFinish={onSave}>
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
          <Select mode="multiple" options={problemOptions} loading={problemLoading || detail.isLoading} />
        </Form.Item>
      </Form>
    </Modal>
  )
}

function assignmentColumns(
  text: ReturnType<typeof useLocale>['text'],
  lang: string,
  actions: {
    edit: (item: Assignment) => void
    remove: (id: number) => void
  },
  admin: boolean
): TableProps<Assignment>['columns'] {
  const columns: TableProps<Assignment>['columns'] = [
    {
      title: text.assignments.name,
      dataIndex: 'title',
      render: (title: string, row) => (
        <Flex align="center" gap={8} className="tableTitleLine">
          <Typography.Text ellipsis className="lineText">
            <Link to={`/assignments/${row.id}`}>{title}</Link>
          </Typography.Text>
          <AssignmentStatus status={row.status} />
        </Flex>
      )
    },
    {
      title: text.assignments.desc,
      dataIndex: 'desc',
      render: (desc: string) => (
        <Typography.Text type="secondary" ellipsis className="lineText">
          {desc}
        </Typography.Text>
      )
    },
    {
      title: text.assignments.progress,
      width: 220,
      render: (_, row) => (
        <Flex align="center" gap={10} style={{ minWidth: 160 }}>
          <Progress percent={progress(row)} size="small" showInfo={false} style={{ minWidth: 96 }} />
          <Typography.Text className="nowrap">{text.assignments.done(row.done, row.total)}</Typography.Text>
        </Flex>
      )
    },
    {
      title: text.assignments.deadline,
      dataIndex: 'endAt',
      width: 220,
      render: (endAt: string) => <Typography.Text>{formatTime(endAt, lang)}</Typography.Text>
    }
  ]
  if (admin) {
    columns.push({
      title: '',
      width: 96,
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
