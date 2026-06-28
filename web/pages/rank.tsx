import { Avatar, Card, Flex, Table, Typography } from 'antd'
import type { TableProps } from 'antd'
import { useQuery } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router-dom'

import { api, apiData } from '../client'
import type { RankUser } from '../client'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { useLocale } from '../locale'
import { pageFromParams, pageSizeFromParams, setPageParams } from '../utils/pagination'

export function RankPage() {
  const { text } = useLocale()
  const [params, setParams] = useSearchParams()
  const page = pageFromParams(params)
  const pageSize = pageSizeFromParams(params)
  const query = useQuery({ queryKey: ['rank', page, pageSize], queryFn: () => apiData(api.GET('/api/rank', { params: { query: { page, pageSize } } })) })

  return (
    <Card>
      {query.isError ? (
        <ErrorBlock error={query.error} />
      ) : query.isLoading ? (
        <LoadingBlock />
      ) : (
        <Table<RankUser>
          rowKey="user"
          columns={rankColumns(text)}
          dataSource={query.data?.items ?? []}
          pagination={{ current: query.data?.page ?? page, pageSize: query.data?.pageSize ?? pageSize, total: query.data?.total ?? 0, showSizeChanger: true }}
          onChange={(pagination) => setParams(setPageParams(params, pagination.current ?? page, pagination.pageSize ?? pageSize))}
        />
      )}
    </Card>
  )
}

function rankColumns(text: ReturnType<typeof useLocale>['text']): TableProps<RankUser>['columns'] {
  return [
    {
      title: text.rank.rank,
      dataIndex: 'rank',
    },
    {
      title: text.rank.user,
      width: 220,
      render: (_, row) => (
        <Flex align="center" gap={12}>
          <Avatar src={row.avatar || undefined}>{row.user.slice(0, 1).toUpperCase()}</Avatar>
          <Typography.Text strong>
            <Link to={`/users/${row.user}`}>{row.user}</Link>
          </Typography.Text>
        </Flex>
      )
    },
    {
      title: text.profile.bio,
      dataIndex: 'bio',
      width: 360,
      render: (bio: string) => (
        <Typography.Text type={bio ? undefined : 'secondary'} ellipsis className="lineText">
          {bio || text.user.noBio}
        </Typography.Text>
      )
    },
    {
      title: text.rank.ac,
      dataIndex: 'ac',
    },
    {
      title: text.rank.submit,
      dataIndex: 'submit',
    }
  ]
}
