import { DeleteOutlined, EditOutlined, EyeInvisibleOutlined, EyeOutlined, PlusOutlined, SearchOutlined } from '@ant-design/icons'
import {
  App as AntApp,
  Button,
  Card,
  Flex,
  Form,
  Input,
  Modal,
  Popconfirm,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography
} from 'antd'
import type { TableProps } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'

import { api, apiData, apiEmpty, uploadProblemImage } from '../../client'
import type { ProblemListItem, ProblemListPage, ProblemState } from '../../client'
import { ProblemLink } from '../../components/entity'
import { ErrorBlock, LoadingBlock } from '../../components/state'
import { TagList } from '../../components/tags'
import { TagSelect } from '../../components/tag-select'
import { useLocale } from '../../locale'
import { useSession } from '../../session'
import { formatLimit, formatPass, problemCode } from '../../utils/format'
import { pageFromParams, pageSizeFromParams, setPageParams } from '../../utils/pagination'
import { problemAssetUploadMarkdownURL, problemMarkdownID } from '../../utils/markdown'
import { ProblemFormFields } from './form'
import type { ProblemFormValues } from './form'

export function ProblemsPage() {
  const { text } = useLocale()
  const session = useSession()
  const { message } = AntApp.useApp()
  const client = useQueryClient()
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const [open, setOpen] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)
  const q = params.get('q') ?? ''
  const tag = params.get('tag') ?? ''
  const page = pageFromParams(params)
  const pageSize = pageSizeFromParams(params)
  const query = useQuery({ queryKey: ['problems', q, tag, page, pageSize], queryFn: () => apiData(api.GET('/api/problems', { params: { query: { q, tag, page, pageSize } } })) })
  const problemIds = (query.data?.items ?? []).map((item) => item.id)
  const ids = problemIds.join(',')
  const state = useQuery({
    queryKey: ['problem-state', ids],
    queryFn: () => apiData(api.GET('/api/problem-state', { params: { query: { ids } } })),
    enabled: ids.length > 0
  })
  const showError = (error: unknown) => {
    message.error(error instanceof Error ? error.message : text.common.loadingFailed)
  }
  const create = useMutation({
    mutationFn: (values: ProblemFormValues) =>
      apiData(api.POST('/api/problems', { body: {
        title: values.title,
        tags: values.tags ?? [],
        mode: values.mode,
        timeMs: values.timeMs,
        memoryMb: values.memoryMb
      } })),
    onSuccess: (item) => {
      void client.invalidateQueries({ queryKey: ['problems'] })
      void client.invalidateQueries({ queryKey: ['home'] })
      message.success(text.common.saved)
      closeModal()
      navigate(`/problems/${item.id}`)
    },
    onError: showError
  })
  const update = useMutation({
    mutationFn: (values: ProblemFormValues) => {
      if (!editingId) {
        throw new Error(text.common.emptyResponse)
      }
      return apiData(api.PATCH('/api/problems/{id}', { params: { path: { id: editingId } }, body: {
        title: values.title,
        statement: values.statement ?? '',
        tags: values.tags ?? [],
        mode: values.mode,
        timeMs: values.timeMs,
        memoryMb: values.memoryMb
      } }))
    },
    onSuccess: (item) => {
      void client.invalidateQueries({ queryKey: ['problem', item.id] })
      void client.invalidateQueries({ queryKey: ['problems'] })
      void client.invalidateQueries({ queryKey: ['home'] })
      message.success(text.common.saved)
      closeModal()
    },
    onError: showError
  })
  const remove = useMutation({
    mutationFn: (id: number) => apiEmpty(api.DELETE('/api/problems/{id}', { params: { path: { id } } })),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['problems'] })
      void client.invalidateQueries({ queryKey: ['home'] })
      message.success(text.common.saved)
    },
    onError: showError
  })
  const visibility = useMutation({
    mutationFn: (item: ProblemListItem) =>
      apiData(api.PATCH('/api/problems/{id}/visibility', { params: { path: { id: item.id } }, body: { visible: !item.visible } })),
    onSuccess: (item) => {
      replaceProblemInCaches(client, item)
      void client.invalidateQueries({ queryKey: ['problems'] })
      void client.invalidateQueries({ queryKey: ['home'] })
      message.success(item.visible ? text.problems.shown : text.problems.hiddenDone)
    },
    onError: showError
  })
  const stateByProblem = new Map((state.data ?? []).map((item) => [item.problemId, item]))
  const stateLoading = ids.length > 0 && state.isLoading
  const columns = problemColumns(text, stateByProblem, {
    edit: (item) => openEdit(item),
    remove: (id) => remove.mutate(id),
    toggle: (item) => visibility.mutate(item),
    toggling: (item) => visibility.isPending && visibility.variables?.id === item.id
  }, session.admin)

  function submit(values: { q?: string; tag?: string }) {
    const next = new URLSearchParams()
    if (values.q) {
      next.set('q', values.q)
    }
    if (values.tag) {
      next.set('tag', values.tag)
    }
    setParams(next)
  }

  function clear() {
    setParams(new URLSearchParams())
  }

  function openCreate() {
    setEditingId(null)
    setOpen(true)
  }

  function openEdit(item: ProblemListItem) {
    setEditingId(item.id)
    setOpen(true)
  }

  function closeModal() {
    setOpen(false)
    setEditingId(null)
  }

  function save(values: ProblemFormValues) {
    if (editingId) {
      update.mutate(values)
      return
    }
    create.mutate(values)
  }

  return (
    <Card>
      <Flex vertical gap={16}>
        <Flex className="tableToolbar" justify="space-between" align="center" gap={12} wrap>
          <Form className="tableToolbarForm" layout="inline" initialValues={{ q: q || undefined, tag: tag || undefined }} onFinish={submit} key={`${q}:${tag}`}>
            <Form.Item name="q">
              <Input placeholder={text.problems.q} allowClear style={{ width: 280 }} />
            </Form.Item>
            <Form.Item name="tag">
              <TagSelect kind="problem" placeholder={text.problems.tag} allowClear style={{ width: 220 }} />
            </Form.Item>
            <Form.Item>
              <Button onClick={clear}>{text.common.clear}</Button>
            </Form.Item>
            <Form.Item>
              <Button type="primary" htmlType="submit" icon={<SearchOutlined />}>
                {text.common.search}
              </Button>
            </Form.Item>
          </Form>
          {session.admin ? (
            <Button icon={<PlusOutlined />} onClick={openCreate}>
              {text.common.createProblem}
            </Button>
          ) : null}
        </Flex>
        {query.isError ? (
          <ErrorBlock error={query.error} />
        ) : state.isError ? (
          <ErrorBlock error={state.error} />
        ) : (
          <Table<ProblemListItem>
            rowKey="id"
            scroll={{ x: session.admin ? 1080 : 960 }}
            loading={query.isLoading || stateLoading}
            columns={columns}
            dataSource={query.data?.items ?? []}
            pagination={{ current: query.data?.page ?? page, pageSize: query.data?.pageSize ?? pageSize, total: query.data?.total ?? 0, showSizeChanger: true }}
            onChange={(pagination) => setParams(setPageParams(params, pagination.current ?? page, pagination.pageSize ?? pageSize))}
          />
        )}
      </Flex>
      {session.admin && open ? (
        <ProblemModal editingId={editingId} loading={create.isPending || update.isPending} onCancel={closeModal} onSave={save} />
      ) : null}
    </Card>
  )
}

function ProblemModal({
  editingId,
  loading,
  onCancel,
  onSave
}: {
  editingId: number | null
  loading: boolean
  onCancel: () => void
  onSave: (values: ProblemFormValues) => void
}) {
  const { text } = useLocale()
  const [form] = Form.useForm<ProblemFormValues>()
  const isEdit = editingId !== null
  const detail = useQuery({
    queryKey: ['problem', editingId],
    queryFn: () => apiData(api.GET('/api/problems/{id}', { params: { path: { id: editingId ?? 0 } } })),
    enabled: isEdit
  })
  const initialValues: Partial<ProblemFormValues> = isEdit && detail.data
    ? {
        title: detail.data.title,
        statement: detail.data.statement || `# ${detail.data.title}`,
        tags: detail.data.tags,
        mode: detail.data.mode,
        timeMs: detail.data.timeMs,
        memoryMb: detail.data.memoryMb
      }
    : { tags: [], mode: 'default', timeMs: 1000, memoryMb: 256 }
  const body = isEdit && detail.isLoading ? (
    <LoadingBlock />
  ) : isEdit && detail.isError ? (
    <ErrorBlock error={detail.error} />
  ) : (
    <Form<ProblemFormValues>
      key={isEdit ? `problem-${editingId}` : 'problem-new'}
      form={form}
      preserve={false}
      layout="vertical"
      initialValues={initialValues}
      onFinish={onSave}
    >
      <ProblemFormFields
        showMode
        statement={isEdit && editingId ? {
          editorId: problemMarkdownID(editingId),
          height: 260,
          upload: async (file) => problemAssetUploadMarkdownURL(await uploadProblemImage(editingId, file), editingId)
        } : undefined}
      />
    </Form>
  )

  return (
    <Modal
      open
      destroyOnHidden
      width={isEdit ? 780 : undefined}
      title={isEdit ? text.common.edit : text.common.createProblem}
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

function problemColumns(
  text: ReturnType<typeof useLocale>['text'],
  state: Map<number, ProblemState>,
  actions: {
    edit: (item: ProblemListItem) => void
    remove: (id: number) => void
    toggle: (item: ProblemListItem) => void
    toggling: (item: ProblemListItem) => boolean
  },
  admin: boolean
): TableProps<ProblemListItem>['columns'] {
  const columns: TableProps<ProblemListItem>['columns'] = [
    {
      title: text.submissions.problem,
      dataIndex: 'title',
      width: 420,
      ellipsis: { showTitle: false },
      render: (title: string, row) => (
        <Flex align="center" gap={8} wrap={false} className="tableTitleLine">
          <ProblemLink id={row.id} title={title} maxWidth={560} />
          <ProblemRecordTag status={state.get(row.id)?.status} />
          {!row.visible ? <Tag>{text.problems.hidden}</Tag> : null}
        </Flex>
      )
    },
    {
      title: text.problems.tag,
      dataIndex: 'tags',
      width: 280,
      render: (tags: string[]) => <TagList tags={tags} empty={<Typography.Text type="secondary">-</Typography.Text>} />
    },
    {
      title: text.problems.limit,
      render: (_, row) => <Typography.Text type="secondary" className="nowrap">{formatLimit(row)}</Typography.Text>
    },
    {
      title: text.problems.pass,
      render: (_, row) => {
        const item = state.get(row.id)
        return item ? <Typography.Text>{formatPass(item)}</Typography.Text> : <Typography.Text type="secondary">-</Typography.Text>
      }
    }
  ]
  if (admin) {
    columns.push({
      title: text.common.actions,
      align: 'right',
      render: (_, row) => (
        <Space size={4}>
          <Tooltip title={row.visible ? text.problems.hide : text.problems.show}>
            <Button
              aria-label={`${row.visible ? text.problems.hide : text.problems.show} ${problemCode(row.id)}`}
              type="text"
              icon={row.visible ? <EyeOutlined className="okIcon" /> : <EyeInvisibleOutlined className="mutedIcon" />}
              loading={actions.toggling(row)}
              disabled={actions.toggling(row)}
              onClick={(event) => { event.stopPropagation(); actions.toggle(row) }}
            />
          </Tooltip>
          <Tooltip title={text.common.edit}>
            <Button aria-label={`${text.common.edit} ${problemCode(row.id)}`} type="text" icon={<EditOutlined />} onClick={() => actions.edit(row)} />
          </Tooltip>
          <Popconfirm title={text.common.confirmDelete} okText={text.common.delete} cancelText={text.common.cancel} onConfirm={() => actions.remove(row.id)}>
            <Button aria-label={`${text.common.delete} ${problemCode(row.id)}`} type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      )
    })
  }
  return columns
}

function ProblemRecordTag({ status }: { status?: string }) {
  const { text } = useLocale()
  if (status === 'pending') {
    return <Tag color="processing">{text.submissions.statuses.pending}</Tag>
  }
  if (status === 'ac') {
    return <Tag color="success">{text.problem.passed}</Tag>
  }
  if (status === 'tried') {
    return <Tag color="warning">{text.problem.tried}</Tag>
  }
  return null
}

function replaceProblemInCaches(client: ReturnType<typeof useQueryClient>, item: ProblemListItem) {
  client.setQueriesData<ProblemListPage>({ queryKey: ['problems'] }, (old) => {
    if (!old) {
      return old
    }
    return { ...old, items: old.items.map((row) => (row.id === item.id ? item : row)) }
  })
}
