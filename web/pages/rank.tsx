import { Avatar, Card, Flex, Table, Typography } from 'antd'
import type { TableProps } from 'antd'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'

import { getRank } from '../client'
import type { RankUser } from '../client'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { useLocale } from '../locale'

export function RankPage() {
  const { text } = useLocale()
  const query = useQuery({ queryKey: ['rank'], queryFn: getRank })

  return (
    <Card>
      {query.isError ? (
        <ErrorBlock error={query.error} />
      ) : query.isLoading ? (
        <LoadingBlock />
      ) : (
        <Table<RankUser> rowKey="user" columns={rankColumns(text)} dataSource={query.data} pagination={false} />
      )}
    </Card>
  )
}

function rankColumns(text: ReturnType<typeof useLocale>['text']): TableProps<RankUser>['columns'] {
  return [
    {
      title: text.rank.rank,
      dataIndex: 'rank',
      width: 100
    },
    {
      title: text.rank.user,
      render: (_, row) => (
        <Flex align="center" gap={12}>
          <Avatar src={row.avatar || undefined}>{row.user.slice(0, 1).toUpperCase()}</Avatar>
          <Flex vertical>
            <Typography.Text strong>
              <Link to={`/users/${row.user}`}>{row.user}</Link>
            </Typography.Text>
            <Typography.Text type="secondary" ellipsis className="lineText">
              {row.bio}
            </Typography.Text>
          </Flex>
        </Flex>
      )
    },
    {
      title: text.rank.ac,
      dataIndex: 'ac',
      width: 120
    },
    {
      title: text.rank.submit,
      dataIndex: 'submit',
      width: 140
    }
  ]
}
