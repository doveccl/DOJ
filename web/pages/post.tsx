import { DeleteOutlined, EditOutlined, LockOutlined, PushpinOutlined, UnlockOutlined } from '@ant-design/icons'
import { App as AntApp, Button, Card, Divider, Flex, Form, Input, Modal, Pagination, Popconfirm, Select, Space, Tag, Tooltip, Typography } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { createComment, deleteDiscussion, getDiscussion, updateDiscussion } from '../client'
import type { CommentCreate, DiscussionDetail, DiscussionUpdate } from '../client'
import { UserLink } from '../components/entity'
import { MarkdownEditor, MarkdownPreview } from '../components/markdown'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { useLocale } from '../locale'
import { useSession } from '../session'
import { formatTime } from '../utils/format'
import { limits } from '../utils/limits'

const commentPageSize = 20

export function PostPage() {
  const { lang, text } = useLocale()
  const session = useSession()
  const { message } = AntApp.useApp()
  const client = useQueryClient()
  const navigate = useNavigate()
  const [form] = Form.useForm<CommentCreate>()
  const [editOpen, setEditOpen] = useState(false)
  const [commentPage, setCommentPage] = useState(1)
  const params = useParams()
  const id = Number(params.id)
  const query = useQuery({
    queryKey: ['discussion', id],
    queryFn: () => getDiscussion(id),
    enabled: Number.isFinite(id)
  })
  const showError = (error: unknown) => {
    message.error(error instanceof Error ? error.message : text.common.loadingFailed)
  }
  const reply = useMutation({
    mutationFn: (values: CommentCreate) => createComment(id, values),
    onSuccess: (item) => {
      const nextCount = (query.data?.comments.length ?? 0) + 1
      setCommentPage(Math.max(1, Math.ceil(nextCount / commentPageSize)))
      client.setQueryData<DiscussionDetail>(['discussion', id], (old) =>
        old
          ? {
              ...old,
              discussion: { ...old.discussion, replies: old.discussion.replies + 1 },
              comments: [...old.comments, item]
            }
          : old
      )
      form.resetFields()
      message.success(text.discussion.repliedTip)
    },
    onError: showError
  })
  const edit = useMutation({
    mutationFn: (values: DiscussionUpdate) => updateDiscussion(id, values),
    onSuccess: (item) => {
      void client.invalidateQueries({ queryKey: ['discussion'] })
      void client.invalidateQueries({ queryKey: ['discussion', id] })
      setEditOpen(false)
      message.success(text.common.saved)
      query.refetch()
      if (item.id !== id) {
        navigate(`/discussion/${item.id}`)
      }
    },
    onError: showError
  })
  const toggleState = useMutation({
    mutationFn: (values: { pinned?: boolean; locked?: boolean }) => {
      if (!query.data) {
        throw new Error(text.common.emptyResponse)
      }
      return updateDiscussion(id, {
        title: query.data.discussion.title,
        content: query.data.content,
        tags: query.data.discussion.tags,
        pinned: values.pinned ?? query.data.discussion.pinned,
        locked: values.locked ?? query.data.discussion.locked
      })
    },
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['discussion'] })
      void client.invalidateQueries({ queryKey: ['discussion', id] })
      message.success(text.common.saved)
      query.refetch()
    },
    onError: showError
  })
  const remove = useMutation({
    mutationFn: () => deleteDiscussion(id),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['discussion'] })
      message.success(text.common.saved)
      navigate('/discussion')
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

  const { discussion, content, comments } = query.data
  const pageStart = (commentPage - 1) * commentPageSize
  const pageComments = comments.slice(pageStart, pageStart + commentPageSize)

  return (
    <Flex vertical gap={16}>
      <Card>
        <Flex vertical gap={12}>
          <Flex align="flex-start" justify="space-between" gap={16} wrap>
            <Flex align="center" gap={8} wrap>
              {discussion.pinned ? <Tag color="green">{text.discussion.pinned}</Tag> : null}
              {discussion.locked ? <Tag>{text.discussion.locked}</Tag> : null}
              <Typography.Title level={3} style={{ margin: 0 }}>
                {discussion.title}
              </Typography.Title>
            </Flex>
            {session.admin ? (
              <Space size={4}>
                <Tooltip title={discussion.pinned ? text.discussion.unpin : text.discussion.pinned}>
                  <Button
                    aria-label={`${discussion.pinned ? text.discussion.unpin : text.discussion.pinned} #${discussion.id}`}
                    type="text"
                    size="small"
                    icon={<PushpinOutlined />}
                    style={{ color: discussion.pinned ? 'var(--ant-color-primary)' : undefined }}
                    loading={toggleState.isPending && toggleState.variables?.pinned !== undefined}
                    onClick={() => toggleState.mutate({ pinned: !discussion.pinned })}
                  />
                </Tooltip>
                <Tooltip title={discussion.locked ? text.discussion.unlock : text.discussion.locked}>
                  <Button
                    aria-label={`${discussion.locked ? text.discussion.unlock : text.discussion.locked} #${discussion.id}`}
                    type="text"
                    size="small"
                    icon={discussion.locked ? <LockOutlined /> : <UnlockOutlined />}
                    style={{ color: discussion.locked ? 'var(--ant-color-warning)' : undefined }}
                    loading={toggleState.isPending && toggleState.variables?.locked !== undefined}
                    onClick={() => toggleState.mutate({ locked: !discussion.locked })}
                  />
                </Tooltip>
                <Tooltip title={text.common.edit}>
                  <Button aria-label={`${text.common.edit} #${discussion.id}`} type="text" size="small" icon={<EditOutlined />} onClick={() => setEditOpen(true)} />
                </Tooltip>
                <Popconfirm title={text.common.confirmDelete} okText={text.common.delete} cancelText={text.common.cancel} onConfirm={() => remove.mutate()}>
                  <Button aria-label={`${text.common.delete} #${discussion.id}`} type="text" size="small" danger icon={<DeleteOutlined />} loading={remove.isPending} />
                </Popconfirm>
              </Space>
            ) : null}
          </Flex>
          <Space size={8} wrap>
            <UserLink name={discussion.author} />
            <Typography.Text type="secondary">{formatTime(discussion.createdAt, lang)}</Typography.Text>
            {discussion.tags.map((tag) => (
              <Tag key={tag}>
                <Link to={`/discussion?tags=${encodeURIComponent(tag)}`}>{tag}</Link>
              </Tag>
            ))}
          </Space>
          <MarkdownPreview value={content} />
        </Flex>
      </Card>
      <Card title={text.discussion.replies}>
        <Flex vertical gap={16}>
          {pageComments.map((item, index) => (
            <div key={item.id}>
              {index > 0 ? <Divider className="softDivider" /> : null}
              <Flex vertical gap={8} style={{ width: '100%' }}>
                <Space size={8}>
                  <Typography.Text type="secondary">{text.discussion.floor(pageStart + index + 1)}</Typography.Text>
                  <UserLink name={item.author} strong />
                  <Typography.Text type="secondary">{formatTime(item.createdAt, lang)}</Typography.Text>
                </Space>
                <div className="compactMarkdown">
                  <MarkdownPreview value={item.content} />
                </div>
              </Flex>
            </div>
          ))}
          {comments.length > commentPageSize ? (
            <Pagination current={commentPage} pageSize={commentPageSize} total={comments.length} showSizeChanger={false} onChange={setCommentPage} />
          ) : null}
          {!discussion.locked && session.signedIn ? (
            <Form<CommentCreate> form={form} layout="vertical" onFinish={(values) => reply.mutate(values)}>
              <Form.Item name="content" rules={[{ required: true, whitespace: true }]}>
                <MarkdownEditor minHeight={180} />
              </Form.Item>
              <Button type="primary" htmlType="submit" loading={reply.isPending}>
                {text.common.send}
              </Button>
            </Form>
          ) : null}
        </Flex>
      </Card>
      {editOpen ? (
        <PostEditModal
          initial={{ title: discussion.title, content, tags: discussion.tags, pinned: discussion.pinned, locked: discussion.locked }}
          loading={edit.isPending}
          onCancel={() => setEditOpen(false)}
          onSave={(values) => edit.mutate({ ...values, pinned: discussion.pinned, locked: discussion.locked })}
        />
      ) : null}
    </Flex>
  )
}

function PostEditModal({
  initial,
  loading,
  onCancel,
  onSave
}: {
  initial: DiscussionUpdate
  loading: boolean
  onCancel: () => void
  onSave: (values: DiscussionUpdate) => void
}) {
  const { text } = useLocale()
  const [form] = Form.useForm<DiscussionUpdate>()

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
      <Form<DiscussionUpdate> form={form} layout="vertical" initialValues={initial} onFinish={onSave}>
        <Form.Item name="title" label={text.discussion.title} rules={[{ required: true, whitespace: true }]}>
          <Input maxLength={limits.title} showCount />
        </Form.Item>
        <Form.Item name="tags" label={text.discussion.tags}>
          <Select mode="tags" tokenSeparators={[',', '，', ' ']} />
        </Form.Item>
        <Form.Item name="content" label={text.discussion.content} rules={[{ required: true, whitespace: true }]}>
          <MarkdownEditor minHeight={300} />
        </Form.Item>
      </Form>
    </Modal>
  )
}
