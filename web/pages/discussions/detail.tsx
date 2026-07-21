import { DeleteOutlined, EditOutlined, LockOutlined, PushpinOutlined, UnlockOutlined } from '@ant-design/icons'
import { App as AntApp, Button, Card, Divider, Flex, Form, Input, Pagination, Popconfirm, Space, Tag, Tooltip, Typography } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom'

import { api, apiData, apiEmpty } from '../../client'
import type { CommentCreate } from '../../client'
import { UserLink } from '../../components/entity'
import { MarkdownEditor, MarkdownPreview } from '../../components/markdown'
import { ErrorBlock, LoadingBlock } from '../../components/state'
import { TagList } from '../../components/tags'
import { TagSelect } from '../../components/tag-select'
import { useLocale } from '../../locale'
import { useSession } from '../../session'
import { usePageTitle } from '../../title'
import { formatTime } from '../../utils/format'
import { limits } from '../../utils/limits'
import { discussionDraftKey } from '../../utils/draft'

const commentPageSize = 20

type DiscussionForm = {
  title: string
  content: string
  tags: string[]
}

export function DiscussionDetailPage() {
  const { lang, text } = useLocale()
  const session = useSession()
  const { message } = AntApp.useApp()
  const client = useQueryClient()
  const navigate = useNavigate()
  const [form] = Form.useForm<CommentCreate>()
  const [editOpen, setEditOpen] = useState(false)
  const [commentPage, setCommentPage] = useState(1)
  const params = useParams()
  const [searchParams, setSearchParams] = useSearchParams()
  const id = Number(params.id)
  const rawCommentID = Number(searchParams.get('comment'))
  const commentID = Number.isSafeInteger(rawCommentID) && rawCommentID > 0 ? rawCommentID : undefined
  const rawNotificationID = Number(searchParams.get('notification'))
  const notificationID = Number.isSafeInteger(rawNotificationID) && rawNotificationID > 0 ? rawNotificationID : undefined
  const editDraftKey = discussionDraftKey(session.name, 'edit', id)
  const replyDraftKey = discussionDraftKey(session.name, 'reply', id)
  const query = useQuery({
    queryKey: ['discussion', id, commentPage, commentPageSize, commentID],
    queryFn: () => apiData(api.GET('/api/discussion/{id}', { params: { path: { id }, query: { page: commentPage, pageSize: commentPageSize, comment: commentID } } })),
    enabled: Number.isFinite(id)
  })
  usePageTitle(query.data?.discussion.title)
  const showError = (error: unknown) => {
    message.error(error instanceof Error ? error.message : text.common.loadingFailed)
  }
  const clearCommentTarget = () => {
    if (!searchParams.has('comment')) {
      return
    }
    const next = new URLSearchParams(searchParams)
    next.delete('comment')
    setSearchParams(next, { replace: true })
  }
  useEffect(() => {
    if (!notificationID) {
      return
    }
    void apiEmpty(api.POST('/api/notifications/{id}/read', { params: { path: { id: notificationID } } }))
      .then(() => client.invalidateQueries({ queryKey: ['notifications'] }))
      .catch(() => undefined)
  }, [client, notificationID])
  useEffect(() => {
    if (!commentID || !query.data?.comments.items.some((item) => item.id === commentID)) {
      return
    }
    const frame = requestAnimationFrame(() => document.getElementById(`comment-${commentID}`)?.scrollIntoView({ block: 'center' }))
    return () => cancelAnimationFrame(frame)
  }, [commentID, query.data])
  const localMentions = useMemo(() => {
    if (!query.data) {
      return []
    }
    const names: string[] = []
    const seen = new Set<string>([session.name.toLowerCase()])
    const add = (name: string) => {
      const key = name.toLowerCase()
      if (!seen.has(key)) {
        seen.add(key)
        names.push(name)
      }
    }
    for (let index = query.data.comments.items.length - 1; index >= 0; index--) {
      const comment = query.data.comments.items[index]
      if (!comment.deleted) {
        add(comment.author)
      }
    }
    add(query.data.discussion.author)
    return names
  }, [query.data, session.name])
  const reply = useMutation({
    mutationFn: (body: CommentCreate) => apiData(api.POST('/api/discussion/{id}/comments', { params: { path: { id } }, body })),
    onSuccess: () => {
      window.localStorage.removeItem(replyDraftKey)
      const nextCount = (query.data?.comments.total ?? 0) + 1
      clearCommentTarget()
      setCommentPage(Math.max(1, Math.ceil(nextCount / commentPageSize)))
      void client.invalidateQueries({ queryKey: ['discussion'] })
      form.resetFields()
      message.success(text.discussion.repliedTip)
    },
    onError: showError
  })
  const edit = useMutation({
    mutationFn: (body: DiscussionForm) => apiData(api.PATCH('/api/discussion/{id}', { params: { path: { id } }, body })),
    onSuccess: (item) => {
      window.localStorage.removeItem(editDraftKey)
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
    mutationFn: (body: { pinned?: boolean; locked?: boolean }) => apiData(api.PATCH('/api/discussion/{id}', { params: { path: { id } }, body })),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['discussion'] })
      void client.invalidateQueries({ queryKey: ['discussion', id] })
      message.success(text.common.saved)
      query.refetch()
    },
    onError: showError
  })
  const remove = useMutation({
    mutationFn: () => apiEmpty(api.DELETE('/api/discussion/{id}', { params: { path: { id } } })),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['discussion'] })
      message.success(text.common.saved)
      navigate('/discussion')
    },
    onError: showError
  })
  const removeComment = useMutation({
    mutationFn: (commentId: number) => apiEmpty(api.DELETE('/api/discussion/{id}/comments/{commentId}', { params: { path: { id, commentId } } })),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['discussion'] })
      void client.invalidateQueries({ queryKey: ['discussion', id] })
      message.success(text.common.saved)
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
  const pageStart = (comments.page - 1) * comments.pageSize
  const pageComments = comments.items
  const canDelete = session.admin || (session.signedIn && discussion.author.toLowerCase() === session.name.toLowerCase())

  return (
    <Flex vertical gap={16}>
      <Card
        title={
          <Flex align="center" gap={8} wrap={false} style={{ minWidth: 0 }}>
            {discussion.pinned ? <Tag color="green" style={{ marginInlineEnd: 0 }}>{text.discussion.pinned}</Tag> : null}
            {discussion.locked ? <Tag color="warning" style={{ marginInlineEnd: 0 }}>{text.discussion.locked}</Tag> : null}
            <Typography.Text ellipsis={{ tooltip: discussion.title }} style={{ minWidth: 0 }}>
              {discussion.title}
            </Typography.Text>
          </Flex>
        }
        extra={
          editOpen ? (
            <Space size={8}>
              <Button onClick={() => setEditOpen(false)}>{text.common.cancel}</Button>
              <Button type="primary" htmlType="submit" form="discussion-edit-form" loading={edit.isPending}>{text.common.save}</Button>
            </Space>
          ) : canDelete ? (
            <Space size={4}>
              {session.admin ? (
                <>
                  <Tooltip title={discussion.pinned ? text.discussion.unpin : text.discussion.pinned}>
                    <Button
                      aria-label={`${discussion.pinned ? text.discussion.unpin : text.discussion.pinned} #${discussion.id}`}
                      type="text"
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
                      icon={discussion.locked ? <LockOutlined /> : <UnlockOutlined />}
                      style={{ color: discussion.locked ? 'var(--ant-color-warning)' : undefined }}
                      loading={toggleState.isPending && toggleState.variables?.locked !== undefined}
                      onClick={() => toggleState.mutate({ locked: !discussion.locked })}
                    />
                  </Tooltip>
                  <Tooltip title={text.common.edit}>
                    <Button aria-label={`${text.common.edit} #${discussion.id}`} type="text" icon={<EditOutlined />} onClick={() => setEditOpen(true)} />
                  </Tooltip>
                </>
              ) : null}
              <Popconfirm title={text.common.confirmDelete} okText={text.common.delete} cancelText={text.common.cancel} onConfirm={() => remove.mutate()}>
                <Button aria-label={`${text.common.delete} #${discussion.id}`} type="text" danger icon={<DeleteOutlined />} loading={remove.isPending} />
              </Popconfirm>
            </Space>
          ) : null
        }
      >
        {editOpen ? (
          <Form<DiscussionForm>
            id="discussion-edit-form"
            key={discussion.id}
            preserve={false}
            layout="vertical"
            initialValues={{ title: discussion.title, content, tags: discussion.tags }}
            onFinish={(values) => edit.mutate(values)}
          >
            <Form.Item name="title" label={text.discussion.title} rules={[{ required: true, whitespace: true }]}>
              <Input maxLength={limits.title} showCount />
            </Form.Item>
            <Form.Item name="tags" label={text.discussion.tags}>
              <TagSelect kind="discussion" mode="tags" />
            </Form.Item>
            <Form.Item name="content" label={text.discussion.content} rules={[{ required: true, whitespace: true }]}>
              <MarkdownEditor mentions={localMentions} draftKey={editDraftKey} />
            </Form.Item>
          </Form>
        ) : (
          <Flex vertical gap={12}>
            <Space size={8} align="center" wrap>
              <UserLink name={discussion.author} avatar={discussion.avatar} />
              <Typography.Text type="secondary">{formatTime(discussion.createdAt, lang)}</Typography.Text>
              <TagList tags={discussion.tags} linkTo={(tag) => `/discussion?tags=${encodeURIComponent(tag)}`} />
            </Space>
            <MarkdownPreview id={`discussion-${discussion.id}`} value={content} />
          </Flex>
        )}
      </Card>
      <Card title={text.discussion.replies}>
        <Flex vertical gap={16}>
          {pageComments.map((item, index) => (
            <div key={item.id} id={`comment-${item.id}`}>
              {index > 0 ? <Divider style={{ margin: '0 0 8px' }} /> : null}
              <Flex vertical gap={8} style={{ width: '100%' }}>
                <Flex align="center" justify="space-between" gap={8}>
                  <Space size={8} align="center">
                    <Typography.Text type="secondary">{text.discussion.floor(pageStart + index + 1)}</Typography.Text>
                    {item.deleted ? <Typography.Text type="secondary">{text.discussion.deletedReply}</Typography.Text> : <UserLink name={item.author} avatar={item.avatar} />}
                    {item.deleted ? null : <Typography.Text type="secondary">{formatTime(item.createdAt, lang)}</Typography.Text>}
                  </Space>
                  {!item.deleted && (session.admin || (session.signedIn && item.author.toLowerCase() === session.name.toLowerCase())) ? (
                    <Popconfirm title={text.common.confirmDelete} okText={text.common.delete} cancelText={text.common.cancel} onConfirm={() => removeComment.mutate(item.id)}>
                      <Button aria-label={`${text.common.delete} #${item.id}`} type="text" size="small" danger icon={<DeleteOutlined />} loading={removeComment.isPending && removeComment.variables === item.id} />
                    </Popconfirm>
                  ) : null}
                </Flex>
                {item.deleted ? null : (
                  <div className="compactMarkdown">
                    <MarkdownPreview id={`discussion-comment-${item.id}`} value={item.content} />
                  </div>
                )}
              </Flex>
            </div>
          ))}
          {comments.total > comments.pageSize ? (
            <Pagination
              current={comments.page}
              pageSize={comments.pageSize}
              total={comments.total}
              showSizeChanger={false}
              onChange={(page) => {
                clearCommentTarget()
                setCommentPage(page)
              }}
            />
          ) : null}
          {!discussion.locked && session.signedIn ? (
            <Form<CommentCreate> form={form} layout="vertical" onFinish={(values) => reply.mutate(values)}>
              <Form.Item name="content" rules={[{ required: true, whitespace: true }]}>
                <MarkdownEditor mentions={localMentions} draftKey={replyDraftKey} />
              </Form.Item>
              <Button type="primary" htmlType="submit" loading={reply.isPending}>
                {text.common.send}
              </Button>
            </Form>
          ) : null}
        </Flex>
      </Card>
    </Flex>
  )
}
