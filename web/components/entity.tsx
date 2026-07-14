import { UserOutlined } from '@ant-design/icons'
import { Avatar, Tag, Typography } from 'antd'
import { Link } from 'react-router-dom'
import type { ReactNode } from 'react'

import { problemLabel } from '../utils/format'

export function ProblemLink({ id, title, search = '', maxWidth }: { id: number; title?: string; search?: string; maxWidth?: number | string }) {
  const label = problemLabel(id, title)
  return (
    <Link to={`/problems/${id}${search}`} className="entityTextLink">
      <Typography.Text className="ellipsisText" ellipsis={{ tooltip: label }} style={maxWidth === undefined ? undefined : { maxWidth }}>
        {label}
      </Typography.Text>
    </Link>
  )
}

export function UserLink({ name, avatar, maxWidth }: { name: string; avatar?: string; maxWidth?: number | string }) {
  return (
    <Link to={`/users/${name}`} className="entityUserLink">
      <Avatar size={24} src={avatar || undefined} icon={<UserOutlined />} />
      <Typography.Text className="ellipsisText" ellipsis={{ tooltip: name }} style={maxWidth === undefined ? undefined : { maxWidth }}>
        {name}
      </Typography.Text>
    </Link>
  )
}

export function EntityTag({ children, color, maxWidth = 96, title }: { children: ReactNode; color?: string; maxWidth?: number; title?: string }) {
  const label = title ?? (typeof children === 'string' || typeof children === 'number' ? String(children) : undefined)
  return (
    <Tag color={color} className="nowrap">
      <Typography.Text ellipsis={label ? { tooltip: label } : true} style={{ maxWidth, lineHeight: 'inherit' }}>
        {children}
      </Typography.Text>
    </Tag>
  )
}
