import { CheckCircleOutlined, CodeOutlined, MessageOutlined, UserOutlined } from '@ant-design/icons'
import { Avatar, Card, Col, Empty, Flex, Pagination, Row, Space, Statistic, Table, Timeline, Typography } from 'antd'
import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'

import type { SolvedProblem, SolvedProblemPage, UserActivity, UserProfile } from '../client'
import { useLocale } from '../locale'
import { formatTime } from '../utils/format'
import { EntityTag, ProblemLink } from './entity'
import { YearHeatmap } from './heatmap'
import { SubmissionStatus } from './status'

type ProfileOverviewProps = {
  profile: UserProfile
  renderAvatar?: (avatar: ReactNode) => ReactNode
  sidebarAction?: ReactNode
  onSolvedPageChange?: (page: number, pageSize: number) => void
}

export function ProfileOverview({ profile, renderAvatar, sidebarAction, onSolvedPageChange }: ProfileOverviewProps) {
  const { text } = useLocale()
  const user = profile.user
  const avatarText = user.name.slice(0, 1).toUpperCase()
  const avatar = (
    <Avatar className="profileAvatar" size={168} src={user.avatar || undefined} icon={avatarText ? undefined : <UserOutlined />}>
      {avatarText}
    </Avatar>
  )

  return (
    <Row gutter={[24, 24]} align="top" className="profileLayout">
      <Col xs={24} md={8} lg={6}>
        <Card className="profileSidebar">
          <Flex vertical align="center" gap={14}>
            {renderAvatar ? renderAvatar(avatar) : avatar}
            <Flex vertical align="center" gap={6} className="profileIdentity">
              <Space size={8} align="center" wrap className="profileNameLine">
                <Typography.Title level={2} style={{ margin: 0 }}>
                  {user.name}
                </Typography.Title>
              </Space>
              <Typography.Paragraph type={user.bio ? undefined : 'secondary'} className="profileBio">
                {user.bio || text.user.noBio}
              </Typography.Paragraph>
            </Flex>
            <Row gutter={12} className="profileStats">
              <Col span={12}>
                <Statistic title={text.rank.ac} value={user.ac} />
              </Col>
              <Col span={12}>
                <Statistic title={text.rank.submit} value={user.submit} />
              </Col>
            </Row>
            {sidebarAction ? <div className="profileSidebarAction">{sidebarAction}</div> : null}
          </Flex>
        </Card>
      </Col>
      <Col xs={24} md={16} lg={18}>
        <Flex vertical gap={20}>
          <Card title={text.user.contributions}>
            <YearHeatmap cells={profile.heatmap} />
          </Card>
          <Row gutter={[20, 20]}>
            <Col xs={24} xl={14}>
              <ActivityCard activities={profile.activities} />
            </Col>
            <Col xs={24} xl={10}>
              <SolvedCard page={profile.solved} onPageChange={onSolvedPageChange} />
            </Col>
          </Row>
        </Flex>
      </Col>
    </Row>
  )
}

function ActivityCard({ activities }: { activities: UserActivity[] }) {
  const { lang, text } = useLocale()

  return (
    <Card title={text.user.recent}>
      {activities.length === 0 ? (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} />
      ) : (
        <Timeline
          items={activities.map((activity) => ({
            key: `${activity.type}-${activity.id}`,
            color: activityColor(activity),
            icon: activityIcon(activity),
            content: <ActivityItem activity={activity} lang={lang} submitted={text.user.submitted} posted={text.user.posted} />
          }))}
        />
      )}
    </Card>
  )
}

function activityColor(row: UserActivity) {
  if (row.type === 'discussion') {
    return 'blue'
  }
  return row.status === 'AC' ? 'green' : 'red'
}

function activityIcon(row: UserActivity) {
  if (row.type === 'discussion') {
    return <MessageOutlined />
  }
  return row.status === 'AC' ? <CheckCircleOutlined /> : <CodeOutlined />
}

function ActivityItem({ activity, lang, submitted, posted }: { activity: UserActivity; lang: string; submitted: string; posted: string }) {
  if (activity.type === 'discussion') {
    return (
      <Flex vertical gap={4}>
        <Flex align="center" gap={8} className="profileActivityLine">
          <Typography.Text>{posted}</Typography.Text>
          <span className="profileActivityTitle">
            <Typography.Text ellipsis className="lineText">
              <Link to={`/discussion/${activity.id}`}>{activity.title}</Link>
            </Typography.Text>
          </span>
        </Flex>
        <Typography.Text type="secondary" className="nowrap">
          {formatTime(activity.createdAt, lang)}
        </Typography.Text>
      </Flex>
    )
  }

  return (
    <Flex vertical gap={4}>
      <Flex align="center" gap={8} className="profileActivityLine">
        {activity.status ? (
          <Link to={`/submissions/${activity.id}`}>
            <SubmissionStatus status={activity.status} />
          </Link>
        ) : null}
        <Typography.Text className="nowrap">{submitted}</Typography.Text>
        <span className="profileActivityTitle">
          {activity.problemId ? <ProblemLink id={activity.problemId} title={activity.problemTitle} /> : <Typography.Text ellipsis className="lineText">{activity.title}</Typography.Text>}
        </span>
      </Flex>
      <Typography.Text type="secondary" className="nowrap">
        {formatTime(activity.createdAt, lang)}
      </Typography.Text>
    </Flex>
  )
}

function SolvedCard({ page, onPageChange }: { page: SolvedProblemPage; onPageChange?: (page: number, pageSize: number) => void }) {
  const { text } = useLocale()
  const problems = page.items

  return (
    <Card title={text.user.solved}>
      {problems.length === 0 ? (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} />
      ) : (
        <Flex vertical gap={12}>
          <Table<SolvedProblem>
            size="small"
            showHeader={false}
            tableLayout="fixed"
            pagination={false}
            rowKey="id"
            dataSource={problems}
            columns={[
              {
                ellipsis: true,
                render: (_, row) => <ProblemLink id={row.id} title={row.title} />
              },
              {
                width: 132,
                render: (_, row) => (
                  <Space size={[4, 4]} wrap>
                    {row.tags.slice(0, 2).map((tag) => (
                      <EntityTag key={tag} maxWidth={72}>{tag}</EntityTag>
                    ))}
                  </Space>
                )
              }
            ]}
          />
          {page.total > page.pageSize ? (
            <Flex justify="end">
              <Pagination size="small" current={page.page} pageSize={page.pageSize} total={page.total} showSizeChanger={false} onChange={onPageChange} />
            </Flex>
          ) : null}
        </Flex>
      )}
    </Card>
  )
}
