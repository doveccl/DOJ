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
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography
} from 'antd'
import type { TableProps } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import dayjs from 'dayjs'
import type { Dayjs } from 'dayjs'
import { useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'

import { createContest, deleteContest, getContest, getContestPage, updateContest } from '../client'
import type { Contest, ProblemRef } from '../client'
import { defaultProblemSort, ProblemRefInput } from '../components/problem-ref'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { contestTarget, ScheduleTag } from '../components/time'
import { useLocale } from '../locale'
import { useSession } from '../session'
import { formatTime, problemLabel } from '../utils/format'
import { limits } from '../utils/limits'
import { pageFromParams, pageSizeFromParams, setPageParams } from '../utils/pagination'

type ContestForm = {
  title: string
  kind: string
  startAt: Dayjs
  endAt: Dayjs
  freezeAt?: Dayjs | null
  problems?: ProblemRef[]
}

export function ContestsPage() {
  const { lang, text } = useLocale()
  const session = useSession()
  const { message } = AntApp.useApp()
  const client = useQueryClient()
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const [open, setOpen] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)
  const page = pageFromParams(params)
  const pageSize = pageSizeFromParams(params)
  const query = useQuery({ queryKey: ['contests', page, pageSize], queryFn: () => getContestPage({ page, pageSize }) })
  const showError = (error: unknown) => {
    message.error(error instanceof Error ? error.message : text.common.loadingFailed)
  }
  const payload = (values: ContestForm) => ({
    title: values.title,
    kind: values.kind,
    startAt: values.startAt.toISOString(),
    endAt: values.endAt.toISOString(),
    freezeAt: values.kind === 'ICPC' ? (values.freezeAt?.toISOString() ?? '') : '',
    problems: values.problems ?? []
  })
  const create = useMutation({
    mutationFn: (values: ContestForm) => createContest(payload(values)),
    onSuccess: (item) => {
      void client.invalidateQueries({ queryKey: ['contests'] })
      void client.invalidateQueries({ queryKey: ['home'] })
      message.success(text.common.saved)
      closeModal()
      navigate(`/contests/${item.id}`)
    },
    onError: showError
  })
  const update = useMutation({
    mutationFn: (values: ContestForm) => {
      if (!editingId) {
        throw new Error(text.common.emptyResponse)
      }
      return updateContest(editingId, payload(values))
    },
    onSuccess: (item) => {
      void client.invalidateQueries({ queryKey: ['contests'] })
      void client.invalidateQueries({ queryKey: ['contest', item.id] })
      void client.invalidateQueries({ queryKey: ['home'] })
      message.success(text.common.saved)
      closeModal()
    },
    onError: showError
  })
  const remove = useMutation({
    mutationFn: deleteContest,
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['contests'] })
      void client.invalidateQueries({ queryKey: ['home'] })
      message.success(text.common.saved)
    },
    onError: showError
  })
  function openCreate() {
    setEditingId(null)
    setOpen(true)
  }

  function openEdit(item: Contest) {
    setEditingId(item.id)
    setOpen(true)
  }

  function closeModal() {
    setOpen(false)
    setEditingId(null)
  }

  function save(values: ContestForm) {
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
            {text.contests.create}
          </Button>
        ) : null}
      </Flex>
      {query.isError ? (
        <ErrorBlock error={query.error} />
      ) : query.isLoading ? (
        <LoadingBlock />
      ) : (
        <Table<Contest>
          rowKey="id"
          columns={contestColumns(
            text,
            lang,
            {
              edit: openEdit,
              remove: (id) => remove.mutate(id),
              refresh: () => void query.refetch()
            },
            session.admin
          )}
          dataSource={query.data?.items ?? []}
          pagination={{ current: query.data?.page ?? page, pageSize: query.data?.pageSize ?? pageSize, total: query.data?.total ?? 0, showSizeChanger: true }}
          onChange={(pagination) => setParams(setPageParams(params, pagination.current ?? page, pagination.pageSize ?? pageSize))}
        />
      )}
      {session.admin && open ? (
        <ContestModal
          editingId={editingId}
          loading={create.isPending || update.isPending}
          onCancel={closeModal}
          onSave={save}
        />
      ) : null}
    </Card>
  )
}

function ContestModal({
  editingId,
  loading,
  onCancel,
  onSave
}: {
  editingId: number | null
  loading: boolean
  onCancel: () => void
  onSave: (values: ContestForm) => void
}) {
  const { text } = useLocale()
  const [form] = Form.useForm<ContestForm>()
  const isEdit = editingId !== null
  const detail = useQuery({
    queryKey: ['contest', editingId],
    queryFn: () => getContest(editingId ?? 0),
    enabled: isEdit
  })

  const initialValues: Partial<ContestForm> =
    isEdit && detail.data
      ? {
          title: detail.data.contest.title,
          kind: detail.data.contest.kind,
          startAt: dayjs(detail.data.contest.startAt),
          endAt: dayjs(detail.data.contest.endAt),
          freezeAt: detail.data.contest.freezeAt ? dayjs(detail.data.contest.freezeAt) : null,
          problems: detail.data.problems.map((problem, index) => ({ id: problem.id, sort: problem.sort || defaultProblemSort(index) }))
        }
      : { title: '', kind: 'OI', freezeAt: null, problems: [] }
  const problemOptions = (detail.data?.problems ?? []).map((item) => ({
    value: item.id,
    label: problemLabel(item.id, item.title)
  }))
  const kind = Form.useWatch('kind', form) ?? initialValues.kind ?? 'OI'

  const body =
    isEdit && detail.isLoading ? (
      <LoadingBlock />
    ) : isEdit && detail.isError ? (
      <ErrorBlock error={detail.error} />
    ) : (
      <Form<ContestForm>
        key={isEdit ? `contest-${editingId}` : 'contest-new'}
        form={form}
        preserve={false}
        layout="vertical"
        initialValues={initialValues}
        onFinish={onSave}
      >
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
          {kind === 'ICPC' ? (
            <Form.Item name="freezeAt" label={text.contests.freeze}>
              <DatePicker showTime />
            </Form.Item>
          ) : null}
        </Space>
        <Form.Item name="problems" label={text.contests.problems}>
          <ProblemRefInput options={problemOptions} loading={detail.isLoading} />
        </Form.Item>
      </Form>
    )

  return (
    <Modal
      open
      destroyOnHidden
      title={isEdit ? text.common.edit : text.contests.create}
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

function contestColumns(
  text: ReturnType<typeof useLocale>['text'],
  lang: string,
  actions: {
    edit: (item: Contest) => void
    remove: (id: number) => void
    refresh: () => void
  },
  admin: boolean
): TableProps<Contest>['columns'] {
  const columns: TableProps<Contest>['columns'] = [
    {
      title: text.contests.name,
      dataIndex: 'title',
      render: (title: string, row) => (
        <Flex align="center" gap={8} className="tableTitleLine">
          <Typography.Text ellipsis className="lineText">
            <Link to={`/contests/${row.id}`}>{title}</Link>
          </Typography.Text>
        </Flex>
      )
    },
    {
      title: text.contests.status,
      render: (_, row) => (
        <ScheduleTag
          kind="contest"
          status={row.status}
          target={contestTarget(row.status, row.startAt, row.endAt)}
          range={`${formatTime(row.startAt, lang)} - ${formatTime(row.endAt, lang)}`}
          onFinish={actions.refresh}
        />
      )
    },
    {
      title: text.contests.kind,
      dataIndex: 'kind',
      render: (kind: string) => <Tag>{kind}</Tag>
    },
    {
      title: text.contests.problems,
      dataIndex: 'total',
      render: (total: number) => <Typography.Text>{text.contests.total(total)}</Typography.Text>
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
