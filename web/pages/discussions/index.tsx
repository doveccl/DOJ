import { DeleteOutlined, EditOutlined, LockOutlined, PlusOutlined, PushpinOutlined, SearchOutlined, UnlockOutlined } from '@ant-design/icons'
import { Button, Card, Flex, Form, Input, Modal, Popconfirm, Space, Table, Tag, Tooltip, Typography } from 'antd'
import type { TableProps } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useState } from 'react'

import { api, apiData, apiEmpty } from '../../client'
import type { Discussion, DiscussionCreate } from '../../client'
import { UserLink } from '../../components/entity'
import { MarkdownEditor } from '../../components/markdown'
import { ErrorBlock, LoadingBlock } from '../../components/state'
import { TagList } from '../../components/tags'
import { TagSelect } from '../../components/tag-select'
import { useApiMessage } from '../../components/use-api-message'
import { useLocale } from '../../locale'
import type { Lang } from '../../locale'
import { useSession } from '../../session'
import { formatTime } from '../../utils/format'
import { limits } from '../../utils/limits'
import { pageFromParams, pageSizeFromParams, setPageParams } from '../../utils/pagination'

type DiscussionForm = {
  title: string
  content: string
  tags: string[]
}

export function DiscussionsPage() {
  const { lang, text } = useLocale()
  const session = useSession()
  const [params, setParams] = useSearchParams()
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<Discussion | null>(null)
  const [draft, setDraft] = useState<DiscussionForm | null>(null)
  const { message, showError } = useApiMessage()
  const client = useQueryClient()
  const navigate = useNavigate()
  const q = params.get('q') ?? ''
  const tags = params.get('tags') ?? ''
  const page = pageFromParams(params)
  const pageSize = pageSizeFromParams(params)
  const query = useQuery({ queryKey: ['discussion', q, tags, page, pageSize], queryFn: () => apiData(api.GET('/api/discussion', { params: { query: { q, tags, page, pageSize } } })) })
  const create = useMutation({
    mutationFn: (body: DiscussionCreate) => apiData(api.POST('/api/discussion', { body })),
    onSuccess: (item) => {
      void client.invalidateQueries({ queryKey: ['discussion'] })
      message.success(text.discussion.createdTip)
      setOpen(false)
      setDraft(null)
      navigate(`/discussion/${item.id}`)
    },
    onError: showError
  })
  const update = useMutation({
    mutationFn: (values: DiscussionForm) => {
      if (!editing) {
        throw new Error(text.common.emptyResponse)
      }
      return apiData(api.PATCH('/api/discussion/{id}', { params: { path: { id: editing.id } }, body: values }))
    },
    onSuccess: (item) => {
      void client.invalidateQueries({ queryKey: ['discussion'] })
      void client.invalidateQueries({ queryKey: ['discussion', item.id] })
      message.success(text.common.saved)
      closeModal()
    },
    onError: showError
  })
  const toggleState = useMutation({
    mutationFn: ({ item, pinned, locked }: { item: Discussion; pinned?: boolean; locked?: boolean }) => apiData(api.PATCH('/api/discussion/{id}', { params: { path: { id: item.id } }, body: { pinned, locked } })),
    onSuccess: (item) => {
      void client.invalidateQueries({ queryKey: ['discussion'] })
      void client.invalidateQueries({ queryKey: ['discussion', item.id] })
      message.success(text.common.saved)
    },
    onError: showError
  })
  const remove = useMutation({
    mutationFn: (id: number) => apiEmpty(api.DELETE('/api/discussion/{id}', { params: { path: { id } } })),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['discussion'] })
      message.success(text.common.saved)
    },
    onError: showError
  })

  const closeModal = () => {
    setOpen(false)
    setEditing(null)
    setDraft(null)
  }

  function openCreate() {
    setEditing(null)
    setDraft({ title: '', content: '', tags: tags ? [tags] : [] })
    setOpen(true)
  }

  async function openEdit(item: Discussion) {
    setEditing(item)
    try {
      const detail = await client.fetchQuery({ queryKey: ['discussion', item.id], queryFn: () => apiData(api.GET('/api/discussion/{id}', { params: { path: { id: item.id } } })) })
      setEditing(detail.discussion)
      setDraft({
        title: detail.discussion.title,
        content: detail.content,
        tags: detail.discussion.tags
      })
      setOpen(true)
    } catch (error) {
      showError(error)
      setEditing(null)
    }
  }

  function save(values: DiscussionForm) {
    if (editing) {
      update.mutate(values)
      return
    }
    const body: DiscussionCreate = {
      title: values.title,
      content: values.content,
      tags: values.tags
    }
    create.mutate(body)
  }

  function submitSearch(values: { q?: string; tags?: string }) {
    const next = new URLSearchParams()
    if (values.q) {
      next.set('q', values.q)
    }
    if (values.tags) {
      next.set('tags', values.tags)
    }
    setParams(next)
  }

  function clearSearch() {
    setParams(new URLSearchParams())
  }

  return (
    <Card>
      <Flex vertical gap={16}>
        <Flex className="tableToolbar" justify="space-between" align="center" gap={12} wrap>
          <Form className="tableToolbarForm" layout="inline" initialValues={{ q: q || undefined, tags: tags || undefined }} onFinish={submitSearch} key={`${q}:${tags}`}>
            <Form.Item name="q">
              <Input placeholder={text.discussion.search} allowClear />
            </Form.Item>
            <Form.Item name="tags">
              <TagSelect kind="discussion" placeholder={text.discussion.tags} allowClear />
            </Form.Item>
            <Form.Item>
              <Button onClick={clearSearch}>{text.common.clear}</Button>
            </Form.Item>
            <Form.Item>
              <Button type="primary" htmlType="submit" icon={<SearchOutlined />}>
                {text.common.search}
              </Button>
            </Form.Item>
          </Form>
          {session.signedIn ? (
            <Button icon={<PlusOutlined />} onClick={openCreate}>
              {text.discussion.create}
            </Button>
          ) : null}
        </Flex>
        {query.isError ? (
          <ErrorBlock error={query.error} />
        ) : query.isLoading ? (
          <LoadingBlock />
        ) : (
          <Table<Discussion>
            rowKey="id"
            scroll={{ x: session.signedIn ? 1120 : 940 }}
            columns={discussionColumns(text, lang, {
              edit: openEdit,
              togglePin: (item) => toggleState.mutate({ item, pinned: !item.pinned }),
              toggleLock: (item) => toggleState.mutate({ item, locked: !item.locked }),
              remove: (id) => remove.mutate(id)
            }, session.admin, session.name)}
            dataSource={query.data?.items ?? []}
            pagination={{ current: query.data?.page ?? page, pageSize: query.data?.pageSize ?? pageSize, total: query.data?.total ?? 0, showSizeChanger: true }}
            onChange={(pagination) => setParams(setPageParams(params, pagination.current ?? page, pagination.pageSize ?? pageSize))}
          />
        )}
      </Flex>
      {open && draft ? (
        <DiscussionModal
          editing={Boolean(editing)}
          initial={draft}
          loading={create.isPending || update.isPending}
          onCancel={closeModal}
          onSave={save}
        />
      ) : null}
    </Card>
  )
}

function DiscussionModal({
  editing,
  initial,
  loading,
  onCancel,
  onSave
}: {
  editing: boolean
  initial: DiscussionForm
  loading: boolean
  onCancel: () => void
  onSave: (values: DiscussionForm) => void
}) {
  const { text } = useLocale()
  const [form] = Form.useForm<DiscussionForm>()
  const [editorKey, setEditorKey] = useState(0)

  return (
    <Modal
      open
      destroyOnHidden
      title={editing ? text.common.edit : text.discussion.create}
      okText={editing ? text.common.save : text.common.create}
      cancelText={text.common.cancel}
      confirmLoading={loading}
      width={{ xs: 'calc(100vw - 32px)', sm: 960 }}
      afterOpenChange={(open) => {
        if (open) {
          setEditorKey((key) => key + 1)
        }
      }}
      onCancel={onCancel}
      onOk={() => form.submit()}
    >
      <Form<DiscussionForm> form={form} layout="vertical" initialValues={initial} onFinish={onSave}>
        <Form.Item name="title" label={text.discussion.title} rules={[{ required: true, whitespace: true }]}>
          <Input maxLength={limits.title} showCount />
        </Form.Item>
        <Form.Item name="tags" label={text.discussion.tags}>
          <TagSelect kind="discussion" mode="tags" />
        </Form.Item>
        <Form.Item name="content" label={text.discussion.content} rules={[{ required: true, whitespace: true }]}>
          <MarkdownEditor key={editorKey} />
        </Form.Item>
      </Form>
    </Modal>
  )
}

function discussionColumns(
  text: ReturnType<typeof useLocale>['text'],
  lang: Lang,
  actions: {
    edit: (item: Discussion) => void
    togglePin: (item: Discussion) => void
    toggleLock: (item: Discussion) => void
    remove: (id: number) => void
  },
  admin: boolean,
  userName: string
): TableProps<Discussion>['columns'] {
  const columns: TableProps<Discussion>['columns'] = [
    {
      title: text.discussion.title,
      dataIndex: 'title',
      width: 420,
      ellipsis: { showTitle: false },
      render: (title: string, row) => (
        <Flex align="center" gap={8} className="tableTitleLine">
          <Typography.Text ellipsis={{ tooltip: title }}>
            <Link to={`/discussion/${row.id}`}>{title}</Link>
          </Typography.Text>
          {row.pinned ? <Tag color="green">{text.discussion.pinned}</Tag> : null}
          {row.locked ? <Tag color="warning">{text.discussion.locked}</Tag> : null}
        </Flex>
      )
    },
    {
      title: text.discussion.tags,
      dataIndex: 'tags',
      render: (tags: string[]) => <TagList tags={tags} empty={<Typography.Text type="secondary">-</Typography.Text>} />
    },
    {
      title: text.discussion.author,
      dataIndex: 'author',
      width: 180,
      render: (author: string) => <UserLink name={author} />
    },
    {
      title: text.discussion.replies,
      dataIndex: 'replies',
      width: 96,
      align: 'center'
    },
    {
      title: text.discussion.created,
      dataIndex: 'createdAt',
      render: (value: string) => <Typography.Text className="nowrap">{formatTime(value, lang)}</Typography.Text>
    }
  ]
  if (admin || userName) {
    columns.push({
      title: text.common.actions,
      width: admin ? 156 : 56,
      align: 'right',
      render: (_, row) => (
        <Space size={4}>
          {admin ? (
            <>
              <Tooltip title={row.pinned ? text.discussion.unpin : text.discussion.pinned}>
                <Button
                  aria-label={`${row.pinned ? text.discussion.unpin : text.discussion.pinned} #${row.id}`}
                  type="text"
                  icon={<PushpinOutlined />}
                  style={{ color: row.pinned ? 'var(--ant-color-primary)' : undefined }}
                  onClick={() => actions.togglePin(row)}
                />
              </Tooltip>
              <Tooltip title={row.locked ? text.discussion.unlock : text.discussion.locked}>
                <Button
                  aria-label={`${row.locked ? text.discussion.unlock : text.discussion.locked} #${row.id}`}
                  type="text"
                  icon={row.locked ? <LockOutlined /> : <UnlockOutlined />}
                  style={{ color: row.locked ? 'var(--ant-color-warning)' : undefined }}
                  onClick={() => actions.toggleLock(row)}
                />
              </Tooltip>
              <Tooltip title={text.common.edit}>
                <Button aria-label={`${text.common.edit} #${row.id}`} type="text" icon={<EditOutlined />} onClick={() => actions.edit(row)} />
              </Tooltip>
            </>
          ) : null}
          {admin || row.author.toLowerCase() === userName.toLowerCase() ? (
            <Popconfirm title={text.common.confirmDelete} okText={text.common.delete} cancelText={text.common.cancel} onConfirm={() => actions.remove(row.id)}>
              <Button aria-label={`${text.common.delete} #${row.id}`} type="text" danger icon={<DeleteOutlined />} />
            </Popconfirm>
          ) : null}
        </Space>
      )
    })
  }
  return columns
}
