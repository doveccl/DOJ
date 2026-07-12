import { DeleteOutlined, EditOutlined, PlusOutlined, UnorderedListOutlined } from '@ant-design/icons'
import {
  App as AntApp,
  Button,
  Card,
  Flex,
  Form,
  Modal,
  Popconfirm,
  Progress,
  Space,
  Table,
  Tooltip,
  Typography
} from 'antd'
import type { TableProps } from 'antd'
import { useQuery } from '@tanstack/react-query'
import dayjs from 'dayjs'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'

import { api, apiData, apiEmpty } from '../../client'
import type { AssignmentListItem } from '../../client'
import { defaultProblemSort } from '../../components/problem-ref'
import { ErrorBlock, LoadingBlock } from '../../components/state'
import { ScheduleTag } from '../../components/time'
import { useEntityCrud } from '../../components/use-entity-crud'
import { useLocale } from '../../locale'
import { useSession } from '../../session'
import { problemLabel, progress } from '../../utils/format'
import { pageFromParams, pageSizeFromParams, setPageParams } from '../../utils/pagination'
import { AssignmentFormFields } from './form'
import type { AssignmentFormValues } from './form'

export function AssignmentsPage() {
  const { text } = useLocale()
  const session = useSession()
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const page = pageFromParams(params)
  const pageSize = pageSizeFromParams(params)
  const query = useQuery({ queryKey: ['assignments', page, pageSize], queryFn: () => apiData(api.GET('/api/assignments', { params: { query: { page, pageSize } } })) })
  const payload = (values: AssignmentFormValues) => ({
    title: values.title,
    description: values.description,
    endAt: values.endAt.toISOString(),
    problems: values.problems ?? [],
    users: values.users ?? [],
    groups: values.groups ?? []
  })
  const crud = useEntityCrud<AssignmentFormValues, { id: number }>({
    invalidate: [['assignments'], ['assignment'], ['home']],
    create: (values) => apiData(api.POST('/api/assignments', { body: payload(values) })),
    update: (id, values) => apiData(api.PATCH('/api/assignments/{id}', { params: { path: { id } }, body: payload(values) })),
    remove: (id) => apiEmpty(api.DELETE('/api/assignments/{id}', { params: { path: { id } } })),
    onCreated: (item) => navigate(`/assignments/${item.id}`)
  })

  return (
    <Card>
      <Flex vertical gap={16}>
        <Flex className="tableToolbar" justify="flex-end">
          {session.admin ? (
            <Button icon={<PlusOutlined />} onClick={crud.openCreate}>
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
            scroll={{ x: session.admin ? 720 : 620 }}
            columns={assignmentColumns(
              text,
              {
                edit: (item) => crud.openEdit(item.id),
                remove: (id) => crud.remove.mutate(id),
                refresh: () => void query.refetch()
              },
              session.admin
            )}
            dataSource={query.data?.items ?? []}
            pagination={{ current: query.data?.page ?? page, pageSize: query.data?.pageSize ?? pageSize, total: query.data?.total ?? 0, showSizeChanger: true }}
            onChange={(pagination) => setParams(setPageParams(params, pagination.current ?? page, pagination.pageSize ?? pageSize))}
          />
        )}
      </Flex>
      {session.admin && crud.open ? (
        <AssignmentModal
          editingId={crud.editingId}
          loading={crud.saving}
          onCancel={crud.closeModal}
          onSave={crud.save}
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
  onSave: (values: AssignmentFormValues) => void
}) {
  const { text } = useLocale()
  const { modal } = AntApp.useApp()
  const [form] = Form.useForm<AssignmentFormValues>()
  const isEdit = editingId !== null
  const detail = useQuery({
    queryKey: ['assignment', editingId],
    queryFn: () => apiData(api.GET('/api/assignments/{id}', { params: { path: { id: editingId ?? 0 } } })),
    enabled: isEdit
  })

  const initialValues: Partial<AssignmentFormValues> =
    isEdit && detail.data
      ? {
          title: detail.data.assignment.title,
          description: detail.data.description,
          endAt: dayjs(detail.data.assignment.endAt),
          problems: detail.data.problems.map((problem, index) => ({ id: problem.id, sort: problem.sort || defaultProblemSort(index) })),
          users: detail.data.assignment.users,
          groups: detail.data.assignment.groups
        }
      : { title: '', description: '', problems: [], users: [], groups: [] }
  const problemOptions = (detail.data?.problems ?? []).map((item) => ({
    value: item.id,
    label: problemLabel(item.id, item.title)
  }))
  function submit(values: AssignmentFormValues) {
    const risky = isEdit && detail.data && (detail.data.assignment.status === 'ended' || detail.data.progress.some((item) => item.submit > 0))
    if (!risky) {
      onSave(values)
      return
    }
    modal.confirm({
      title: text.assignments.changeWarning,
      content: text.assignments.changeWarningDescription,
      okText: text.common.save,
      cancelText: text.common.cancel,
      onOk: () => onSave(values)
    })
  }

  const body =
    isEdit && detail.isLoading ? (
      <LoadingBlock />
    ) : isEdit && detail.isError ? (
      <ErrorBlock error={detail.error} />
    ) : (
      <Form<AssignmentFormValues>
        key={isEdit ? `assignment-${editingId}` : 'assignment-new'}
        form={form}
        preserve={false}
        layout="vertical"
        initialValues={initialValues}
        onFinish={submit}
      >
        <AssignmentFormFields
          editorId={isEdit ? `assignment-${editingId}-description-edit` : 'assignment-new-description'}
          problemOptions={problemOptions}
          loading={detail.isLoading}
        />
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
      width: 280,
      ellipsis: { showTitle: false },
      render: (title: string, row) => (
        <Flex align="center" gap={8} className="tableTitleLine">
          <Typography.Text ellipsis={{ tooltip: title }}>
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
      width: 180,
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
          <Tooltip title={text.submissions.viewAssignmentRecords}>
            <Button aria-label={`${text.submissions.viewAssignmentRecords} #${row.id}`} type="text" icon={<UnorderedListOutlined />} href={`/submissions?assignment=${row.id}`} />
          </Tooltip>
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
