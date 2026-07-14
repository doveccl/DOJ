import { DeleteOutlined, EditOutlined, PlusOutlined, UnorderedListOutlined } from '@ant-design/icons'
import {
  App as AntApp,
  Button,
  Card,
  Flex,
  Form,
  Modal,
  Popconfirm,
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
import type { Contest } from '../../client'
import { defaultProblemSort } from '../../components/problem-ref'
import { ErrorBlock, LoadingBlock } from '../../components/state'
import { ContestKindTag } from '../../components/status'
import { contestTarget, ScheduleTag } from '../../components/time'
import { useEntityCrud } from '../../components/use-entity-crud'
import { useLocale } from '../../locale'
import type { Lang } from '../../locale'
import { useSession } from '../../session'
import { formatTime, problemLabel } from '../../utils/format'
import { pageFromParams, pageSizeFromParams, setPageParams } from '../../utils/pagination'
import { ContestFormFields } from './form'
import type { ContestFormValues } from './form'

export function ContestsPage() {
  const { lang, text } = useLocale()
  const session = useSession()
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const page = pageFromParams(params)
  const pageSize = pageSizeFromParams(params)
  const query = useQuery({ queryKey: ['contests', page, pageSize], queryFn: () => apiData(api.GET('/api/contests', { params: { query: { page, pageSize } } })) })
  const payload = (values: ContestFormValues) => ({
    title: values.title,
    description: values.description,
    kind: values.kind,
    startAt: values.startAt.toISOString(),
    endAt: values.endAt.toISOString(),
    freezeAt: values.kind === 'ICPC' ? (values.freezeAt?.toISOString() ?? '') : '',
    problems: values.problems ?? []
  })
  const crud = useEntityCrud<ContestFormValues, { id: number }>({
    invalidate: [['contests'], ['contest'], ['home']],
    create: (values) => apiData(api.POST('/api/contests', { body: payload(values) })),
    update: (id, values) => apiData(api.PATCH('/api/contests/{id}', { params: { path: { id } }, body: payload(values) })),
    remove: (id) => apiEmpty(api.DELETE('/api/contests/{id}', { params: { path: { id } } })),
    onCreated: (item) => navigate(`/contests/${item.id}`)
  })

  return (
    <Card>
      <Flex vertical gap={16}>
        <Flex className="tableToolbar" justify="flex-end">
          {session.admin ? (
            <Button icon={<PlusOutlined />} onClick={crud.openCreate}>
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
            scroll={{ x: session.admin ? 800 : 700 }}
            columns={contestColumns(
              text,
              lang,
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
        <ContestModal
          editingId={crud.editingId}
          loading={crud.saving}
          onCancel={crud.closeModal}
          onSave={crud.save}
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
  onSave: (values: ContestFormValues) => void
}) {
  const { text } = useLocale()
  const { modal } = AntApp.useApp()
  const [form] = Form.useForm<ContestFormValues>()
  const isEdit = editingId !== null
  const detail = useQuery({
    queryKey: ['contest', editingId],
    queryFn: () => apiData(api.GET('/api/contests/{id}', { params: { path: { id: editingId ?? 0 } } })),
    enabled: isEdit
  })

  const initialValues: Partial<ContestFormValues> =
    isEdit && detail.data
      ? {
          title: detail.data.contest.title,
          description: detail.data.description,
          kind: detail.data.contest.kind,
          startAt: dayjs(detail.data.contest.startAt),
          endAt: dayjs(detail.data.contest.endAt),
          freezeAt: detail.data.contest.freezeAt ? dayjs(detail.data.contest.freezeAt) : null,
          problems: detail.data.problems.map((problem, index) => ({ id: problem.id, sort: problem.sort || defaultProblemSort(index) }))
        }
      : { title: '', description: '', kind: 'OI', freezeAt: null, problems: [] }
  const problemOptions = (detail.data?.problems ?? []).map((item) => ({
    value: item.id,
    label: problemLabel(item.id, item.title)
  }))
  function submit(values: ContestFormValues) {
    if (!isEdit || detail.data?.contest.status === 'pending') {
      onSave(values)
      return
    }
    modal.confirm({
      title: text.contests.changeWarning,
      content: text.contests.changeWarningDescription,
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
      <Form<ContestFormValues>
        key={isEdit ? `contest-${editingId}` : 'contest-new'}
        form={form}
        preserve={false}
        layout="vertical"
        initialValues={initialValues}
        onFinish={submit}
      >
        <ContestFormFields
          form={form}
          editorId={isEdit ? `contest-${editingId}-description-edit` : 'contest-new-description'}
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
  lang: Lang,
  actions: {
    edit: (item: Contest) => void
    remove: (id: number) => void
    refresh: () => void
  },
  admin: boolean
): TableProps<Contest>['columns'] {
  const columns: TableProps<Contest>['columns'] = [
    {
      title: text.contests.kind,
      dataIndex: 'kind',
      width: 80,
      render: (kind: string) => <ContestKindTag kind={kind} />
    },
    {
      title: text.contests.status,
      width: 160,
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
      title: text.contests.name,
      dataIndex: 'title',
      ellipsis: { showTitle: false },
      render: (title: string, row) => (
        <Flex align="center" gap={8} className="tableTitleLine">
          <Link to={`/contests/${row.id}`} className="entityTextLink">
            <Typography.Text className="ellipsisText" ellipsis={{ tooltip: title }}>{title}</Typography.Text>
          </Link>
        </Flex>
      )
    },
    {
      title: text.contests.problems,
      dataIndex: 'total',
      width: 100,
      render: (total: number) => <Typography.Text>{text.contests.total(total)}</Typography.Text>
    }
  ]
  if (admin) {
    columns.push({
      title: text.common.actions,
      width: 140,
      align: 'right',
      render: (_, row) => (
        <Space size={4}>
          <Tooltip title={text.submissions.viewContestRecords}>
            <Button aria-label={`${text.submissions.viewContestRecords} #${row.id}`} type="text" icon={<UnorderedListOutlined />} href={`/submissions?contest=${row.id}`} />
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
