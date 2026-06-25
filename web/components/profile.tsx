import { CheckCircleOutlined, CodeOutlined, MessageOutlined, UserOutlined } from '@ant-design/icons'
import { Avatar, Card, Col, Empty, Flex, Row, Space, Statistic, Tag, Timeline, Typography } from 'antd'
import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'

import type { Problem, UserActivity, UserProfile } from '../client'
import { useLocale } from '../locale'
import { formatPass, formatTime, problemCode } from '../utils/format'
import { YearHeatmap } from './heatmap'
import { SubmissionStatus } from './status'

type ProfileOverviewProps = {
  profile: UserProfile
  renderAvatar?: (avatar: ReactNode) => ReactNode
  sidebarAction?: ReactNode
}

export function ProfileOverview({ profile, renderAvatar, sidebarAction }: ProfileOverviewProps) {
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
                {user.admin ? <Tag color="cyan">{text.admin.roles.admin}</Tag> : null}
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
              <SolvedCard problems={profile.solved} />
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
    <Card title={text.user.recent} className="profileActivity">
      {activities.length === 0 ? (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} />
      ) : (
        <Timeline
          items={activities.slice(0, 12).map((row) => ({
            icon: activityIcon(row),
            content: <ActivityItem activity={row} lang={lang} submitted={text.user.submitted} posted={text.user.posted} />
          }))}
        />
      )}
    </Card>
  )
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
      <Flex vertical gap={4} className="profileActivityItem">
        <Flex align="center" gap={8} wrap>
          <Typography.Text>{posted}</Typography.Text>
          <Typography.Text ellipsis className="lineText">
            <Link to={`/discussion/${activity.id}`}>{activity.title}</Link>
          </Typography.Text>
        </Flex>
        <Typography.Text type="secondary">{formatTime(activity.createdAt, lang)}</Typography.Text>
      </Flex>
    )
  }

  return (
    <Flex vertical gap={4} className="profileActivityItem">
      <Flex align="center" gap={8} wrap>
        {activity.status ? (
          <Link to={`/submissions/${activity.id}`}>
            <SubmissionStatus status={activity.status} />
          </Link>
        ) : null}
        <Typography.Text>{submitted}</Typography.Text>
        <Typography.Text ellipsis className="lineText">
          {activity.problemId ? (
            <Link to={`/problems/${activity.problemId}`}>
              {problemCode(activity.problemId)} {activity.problemTitle}
            </Link>
          ) : (
            activity.title
          )}
        </Typography.Text>
      </Flex>
      <Typography.Text type="secondary">{formatTime(activity.createdAt, lang)}</Typography.Text>
    </Flex>
  )
}

function SolvedCard({ problems }: { problems: Problem[] }) {
  const { text } = useLocale()

  return (
    <Card title={text.user.solved}>
      {problems.length === 0 ? (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} />
      ) : (
        <Flex vertical className="profileProblemList">
          {problems.slice(0, 12).map((row) => (
            <Flex key={row.id} align="center" justify="space-between" gap={12} className="profileProblemRow">
              <Flex align="center" gap={8} className="profileProblemTitle">
                <Typography.Text ellipsis className="lineText">
                  <Link to={`/problems/${row.id}`}>
                    {problemCode(row.id)} {row.title}
                  </Link>
                </Typography.Text>
                {row.tags.slice(0, 2).map((tag) => (
                  <Tag key={tag}>{tag}</Tag>
                ))}
              </Flex>
              <Typography.Text type="secondary" className="nowrap">
                {formatPass(row)}
              </Typography.Text>
            </Flex>
          ))}
        </Flex>
      )}
    </Card>
  )
}
