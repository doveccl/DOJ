import { Card, Table, Typography } from 'antd'
import type { TableProps } from 'antd'
import { useQuery } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'

import { api, apiData } from '../client'
import type { RankUser } from '../client'
import { UserLink } from '../components/entity'
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
          scroll={{ x: 560 }}
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
      width: 72,
      align: 'center'
    },
    {
      title: text.rank.user,
      width: 220,
      ellipsis: { showTitle: false },
      render: (_, row) => <UserLink name={row.user} avatar={row.avatar} maxWidth={168} />
    },
    {
      title: text.profile.bio,
      dataIndex: 'bio',
      ellipsis: { showTitle: false },
      render: (bio: string) => (
        <Typography.Text type={bio ? undefined : 'secondary'} ellipsis={{ tooltip: bio || text.user.noBio }}>
          {bio || text.user.noBio}
        </Typography.Text>
      )
    },
    {
      title: text.rank.ac,
      dataIndex: 'ac',
      width: 88,
      align: 'center'
    },
    {
      title: text.rank.submit,
      dataIndex: 'submit',
      width: 88,
      align: 'center'
    }
  ]
}
