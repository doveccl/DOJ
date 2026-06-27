import { DeleteOutlined, EditOutlined, LockOutlined, PlusOutlined, PushpinOutlined, SearchOutlined, UnlockOutlined } from '@ant-design/icons'
import { App as AntApp, Button, Card, Flex, Form, Input, Modal, Popconfirm, Space, Table, Tag, Tooltip, Typography } from 'antd'
import type { TableProps } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useState } from 'react'

import { createDiscussion, deleteDiscussion, getDiscussion, getDiscussionPage, updateDiscussion } from '../client'
import type { Discussion, DiscussionCreate } from '../client'
import { UserLink } from '../components/entity'
import { MarkdownEditor } from '../components/markdown'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { TagSelect } from '../components/tag-select'
import { useLocale } from '../locale'
import { useSession } from '../session'
import { formatTime } from '../utils/format'
import { limits } from '../utils/limits'
import { pageFromParams, pageSizeFromParams, setPageParams } from '../utils/pagination'

type DiscussionForm = {
  title: string
  content: string
  tags: string[]
}

export function DiscussionPage() {
  const { lang, text } = useLocale()
  const session = useSession()
  const [params, setParams] = useSearchParams()
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<Discussion | null>(null)
  const [draft, setDraft] = useState<DiscussionForm | null>(null)
  const { message } = AntApp.useApp()
  const client = useQueryClient()
  const navigate = useNavigate()
  const q = params.get('q') ?? ''
  const tags = params.get('tags') ?? ''
  const page = pageFromParams(params)
  const pageSize = pageSizeFromParams(params)
  const query = useQuery({ queryKey: ['discussion', q, tags, page, pageSize], queryFn: () => getDiscussionPage({ q, tags, page, pageSize }) })
  const showError = (error: unknown) => {
    message.error(error instanceof Error ? error.message : text.common.loadingFailed)
  }
  const create = useMutation({
    mutationFn: createDiscussion,
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
      return updateDiscussion(editing.id, values)
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
    mutationFn: ({ item, pinned, locked }: { item: Discussion; pinned?: boolean; locked?: boolean }) => updateDiscussion(item.id, { pinned, locked }),
    onSuccess: (item) => {
      void client.invalidateQueries({ queryKey: ['discussion'] })
      void client.invalidateQueries({ queryKey: ['discussion', item.id] })
      message.success(text.common.saved)
    },
    onError: showError
  })
  const remove = useMutation({
    mutationFn: deleteDiscussion,
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
      const detail = await client.fetchQuery({ queryKey: ['discussion', item.id], queryFn: () => getDiscussion(item.id) })
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
      <Flex justify="space-between" align="center" gap={12} wrap style={{ marginBottom: 18 }}>
        <Form layout="inline" initialValues={{ q: q || undefined, tags: tags || undefined }} onFinish={submitSearch} key={`${q}:${tags}`}>
          <Form.Item name="q">
            <Input placeholder={text.discussion.search} allowClear style={{ width: 280 }} />
          </Form.Item>
          <Form.Item name="tags">
            <TagSelect kind="discussion" placeholder={text.discussion.tags} allowClear style={{ width: 180 }} />
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
          columns={discussionColumns(text, lang, {
            edit: openEdit,
            togglePin: (item) => toggleState.mutate({ item, pinned: !item.pinned }),
            toggleLock: (item) => toggleState.mutate({ item, locked: !item.locked }),
            remove: (id) => remove.mutate(id)
          }, session.admin)}
          dataSource={query.data?.items ?? []}
          pagination={{ current: query.data?.page ?? page, pageSize: query.data?.pageSize ?? pageSize, total: query.data?.total ?? 0, showSizeChanger: true }}
          onChange={(pagination) => setParams(setPageParams(params, pagination.current ?? page, pagination.pageSize ?? pageSize))}
        />
      )}
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

  return (
    <Modal
      open
      destroyOnHidden
      title={editing ? text.common.edit : text.discussion.create}
      okText={editing ? text.common.save : text.common.create}
      cancelText={text.common.cancel}
      confirmLoading={loading}
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
          <MarkdownEditor minHeight={300} />
        </Form.Item>
      </Form>
    </Modal>
  )
}

function discussionColumns(
  text: ReturnType<typeof useLocale>['text'],
  lang: string,
  actions: {
    edit: (item: Discussion) => void
    togglePin: (item: Discussion) => void
    toggleLock: (item: Discussion) => void
    remove: (id: number) => void
  },
  admin: boolean
): TableProps<Discussion>['columns'] {
  const columns: TableProps<Discussion>['columns'] = [
    {
      title: text.discussion.title,
      dataIndex: 'title',
      render: (title: string, row) => (
        <Flex align="center" gap={8} className="tableTitleLine">
          <Typography.Text ellipsis className="lineText">
            <Link to={`/discussion/${row.id}`}>{title}</Link>
          </Typography.Text>
          {row.pinned ? <Tag color="green">{text.discussion.pinned}</Tag> : null}
          {row.locked ? <Tag>{text.discussion.locked}</Tag> : null}
          {row.tags.map((tag) => (
            <Tag key={tag}>{tag}</Tag>
          ))}
        </Flex>
      )
    },
    {
      title: text.discussion.author,
      dataIndex: 'author',
      render: (author: string) => <UserLink name={author} />
    },
    {
      title: text.discussion.replies,
      dataIndex: 'replies',
    },
    {
      title: text.discussion.created,
      dataIndex: 'createdAt',
      render: (value: string) => <Typography.Text className="nowrap">{formatTime(value, lang)}</Typography.Text>
    }
  ]
  if (admin) {
    columns.push({
      title: text.common.actions,
      align: 'right',
      render: (_, row) => (
        <Space size={4}>
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
          <Popconfirm title={text.common.confirmDelete} okText={text.common.delete} cancelText={text.common.cancel} onConfirm={() => actions.remove(row.id)}>
            <Button aria-label={`${text.common.delete} #${row.id}`} type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      )
    })
  }
  return columns
}
