import { DeleteOutlined, EditOutlined, EyeInvisibleOutlined, EyeOutlined, PlusOutlined, SearchOutlined } from '@ant-design/icons'
import {
  Button,
  Card,
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
import { useNavigate, useSearchParams } from 'react-router-dom'

import { api, apiData, apiEmpty, uploadProblemImage } from '../../client'
import type { ProblemListItem, ProblemListPage, ProblemState } from '../../client'
import { ProblemLink } from '../../components/entity'
import { ErrorBlock, LoadingBlock } from '../../components/state'
import { ProblemStatus } from '../../components/status'
import { TagList } from '../../components/tags'
import { useEntityCrud } from '../../components/use-entity-crud'
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
  const client = useQueryClient()
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const q = params.get('q') ?? ''
  const tag = params.get('tag') ?? ''
  const status = params.get('status') ?? ''
  const showStatusFilter = session.signedIn
  const activeStatus = showStatusFilter && isProblemStatus(status) ? status : ''
  const page = pageFromParams(params)
  const pageSize = pageSizeFromParams(params)
  const query = useQuery({ queryKey: ['problems', q, tag, activeStatus, page, pageSize], queryFn: () => apiData(api.GET('/api/problems', { params: { query: { q, tag, status: activeStatus || undefined, page, pageSize } } })) })
  const problemIds = (query.data?.items ?? []).map((item) => item.id)
  const ids = problemIds.join(',')
  const state = useQuery({
    queryKey: ['problem-state', ids],
    queryFn: () => apiData(api.GET('/api/problem-state', { params: { query: { ids } } })),
    enabled: ids.length > 0
  })
  const crud = useEntityCrud<ProblemFormValues, { id: number }>({
    invalidate: [['problems'], ['problem'], ['home']],
    create: (values) => apiData(api.POST('/api/problems', { body: {
      title: values.title,
      tags: values.tags ?? [],
      mode: values.mode,
      timeMs: values.timeMs,
      memoryMb: values.memoryMb
    } })),
    update: (id, values) => apiData(api.PATCH('/api/problems/{id}', { params: { path: { id } }, body: {
      title: values.title,
      statement: values.statement ?? '',
      tags: values.tags ?? [],
      mode: values.mode,
      timeMs: values.timeMs,
      memoryMb: values.memoryMb
    } })),
    remove: (id) => apiEmpty(api.DELETE('/api/problems/{id}', { params: { path: { id } } })),
    onCreated: (item) => navigate(`/problems/${item.id}`)
  })
  const visibility = useMutation({
    mutationFn: (item: ProblemListItem) =>
      apiData(api.PATCH('/api/problems/{id}/visibility', { params: { path: { id: item.id } }, body: { visible: !item.visible } })),
    onSuccess: (item) => {
      replaceProblemInCaches(client, item)
      void client.invalidateQueries({ queryKey: ['problems'] })
      void client.invalidateQueries({ queryKey: ['home'] })
      crud.message.success(item.visible ? text.problems.shown : text.problems.hiddenDone)
    },
    onError: crud.showError
  })
  const stateByProblem = new Map((state.data ?? []).map((item) => [item.problemId, item]))
  const stateLoading = ids.length > 0 && state.isLoading
  const columns = problemColumns(text, stateByProblem, {
    edit: (item) => crud.openEdit(item.id),
    remove: (id) => crud.remove.mutate(id),
    toggle: (item) => visibility.mutate(item),
    toggling: (item) => visibility.isPending && visibility.variables?.id === item.id
  }, session.admin)

  function submit(values: { q?: string; tag?: string; status?: string }) {
    const next = new URLSearchParams()
    if (values.q) {
      next.set('q', values.q)
    }
    if (values.tag) {
      next.set('tag', values.tag)
    }
    if (showStatusFilter && values.status) {
      next.set('status', values.status)
    }
    setParams(next)
  }

  function clear() {
    setParams(new URLSearchParams())
  }

  return (
    <Card>
      <Flex vertical gap={16}>
        <Flex className="tableToolbar" justify="space-between" align="center" gap={12} wrap>
          <Form className="tableToolbarForm" layout="inline" initialValues={{ q: q || undefined, tag: tag || undefined, status: activeStatus || undefined }} onFinish={submit} key={`${q}:${tag}:${activeStatus}`}>
            <Form.Item name="q">
              <Input placeholder={text.problems.q} allowClear />
            </Form.Item>
            <Form.Item name="tag">
              <TagSelect kind="problem" placeholder={text.problems.tag} allowClear />
            </Form.Item>
            {showStatusFilter ? (
              <Form.Item name="status">
                <Select
                  allowClear
                  placeholder={text.problems.statusFilter}
                  options={[
                    { value: 'none', label: text.problems.statusNone },
                    { value: 'pending', label: text.problems.statusPending },
                    { value: 'tried', label: text.problems.statusTried },
                    { value: 'ac', label: text.problems.statusAc }
                  ]}
                />
              </Form.Item>
            ) : null}
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
            <Button icon={<PlusOutlined />} onClick={crud.openCreate}>
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
      {session.admin && crud.open ? (
        <ProblemModal editingId={crud.editingId} loading={crud.saving} onCancel={crud.closeModal} onSave={crud.save} />
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
      title: text.problems.statusFilter,
      width: 88,
      align: 'center',
      render: (_, row) => <ProblemStatus status={state.get(row.id)?.status} />
    },
    {
      title: text.submissions.problem,
      dataIndex: 'title',
      ellipsis: { showTitle: false },
      render: (title: string, row) => (
        <Flex align="center" gap={8} wrap={false} className="tableTitleLine">
          <ProblemLink id={row.id} title={title} />
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
      width: 160,
      render: (_, row) => <Typography.Text type="secondary" className="nowrap">{formatLimit(row)}</Typography.Text>
    },
    {
      title: text.problems.pass,
      width: 128,
      render: (_, row) => {
		const item = state.get(row.id)
		return item ? <Typography.Text className="nowrap">{formatPass(item)}</Typography.Text> : <Typography.Text type="secondary">-</Typography.Text>
      }
    }
  ]
  if (admin) {
    columns.push({
      title: text.common.actions,
      width: 140,
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

function isProblemStatus(value: string): value is 'none' | 'pending' | 'tried' | 'ac' {
  return value === 'none' || value === 'pending' || value === 'tried' || value === 'ac'
}

function replaceProblemInCaches(client: ReturnType<typeof useQueryClient>, item: ProblemListItem) {
  client.setQueriesData<ProblemListPage>({ queryKey: ['problems'] }, (old) => {
    if (!old) {
      return old
    }
    return { ...old, items: old.items.map((row) => (row.id === item.id ? item : row)) }
  })
}
