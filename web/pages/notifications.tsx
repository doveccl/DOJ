import { CheckOutlined } from '@ant-design/icons'
import { Button, Card, Flex, List, Pagination, Typography } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link } from 'react-router-dom'

import { api, apiData, apiEmpty } from '../client'
import { UserLink } from '../components/entity'
import { EmptyBlock, ErrorBlock, LoadingBlock } from '../components/state'
import { useApiMessage } from '../components/use-api-message'
import { useLocale } from '../locale'
import { useSession } from '../session'
import { usePageTitle } from '../title'
import { formatTime } from '../utils/format'

const pageSize = 20

export function NotificationsPage() {
  const { lang, text } = useLocale()
  const session = useSession()
  const client = useQueryClient()
  const { showError } = useApiMessage()
  const [page, setPage] = useState(1)
  const query = useQuery({
    queryKey: ['notifications', page, pageSize],
    queryFn: () => apiData(api.GET('/api/notifications', { params: { query: { page, pageSize } } })),
    enabled: session.signedIn
  })
  const readAll = useMutation({
    mutationFn: () => apiEmpty(api.POST('/api/notifications/read-all')),
    onSuccess: () => client.invalidateQueries({ queryKey: ['notifications'] }),
    onError: showError
  })
  usePageTitle(text.notification.title)

  if (!session.signedIn) {
    return <ErrorBlock error={text.common.forbidden} />
  }
  if (query.isLoading) {
    return <LoadingBlock />
  }
  if (query.isError) {
    return <ErrorBlock error={query.error} />
  }
  if (!query.data) {
    return <ErrorBlock error={text.common.emptyResponse} />
  }

  const data = query.data
  return (
    <Card
      title={text.notification.title}
      extra={
        <Button
          type="link"
          size="small"
          icon={<CheckOutlined />}
          disabled={data.unread === 0}
          loading={readAll.isPending}
          onClick={() => readAll.mutate()}
        >
          {text.notification.readAll}
        </Button>
      }
    >
      {data.items.length === 0 ? (
        <EmptyBlock />
      ) : (
        <Flex vertical gap={16}>
          <List
            dataSource={data.items}
            renderItem={(item) => {
              const targetParams = new URLSearchParams({ notification: String(item.id) })
              if (item.commentId) {
                targetParams.set('comment', String(item.commentId))
              }
              const message = item.kind === 'mention'
                ? text.notification.mentioned(item.discussionTitle)
                : text.notification.replied(item.discussionTitle)
              return (
                <List.Item>
                  <Flex align="center" gap={12} style={{ width: '100%' }}>
                    <span className={item.read ? undefined : 'notificationUnreadDot'} aria-hidden />
                    <UserLink name={item.actor} avatar={item.avatar} />
                    <Flex vertical gap={2} style={{ minWidth: 0 }}>
                      <Link to={`/discussion/${item.discussionId}?${targetParams.toString()}`}>{message}</Link>
                      <Typography.Text type="secondary">{formatTime(item.createdAt, lang)}</Typography.Text>
                    </Flex>
                  </Flex>
                </List.Item>
              )
            }}
          />
          {data.total > data.pageSize ? (
            <Flex justify="end">
              <Pagination current={data.page} pageSize={data.pageSize} total={data.total} showSizeChanger={false} onChange={setPage} />
            </Flex>
          ) : null}
        </Flex>
      )}
    </Card>
  )
}
