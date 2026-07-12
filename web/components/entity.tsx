import { Tag, Typography } from 'antd'
import { Link } from 'react-router-dom'
import type { ReactNode } from 'react'

import { problemLabel } from '../utils/format'

export function ProblemLink({ id, title, search = '', strong, maxWidth }: { id: number; title?: string; search?: string; strong?: boolean; maxWidth?: number }) {
  const label = problemLabel(id, title)
  return (
    <Typography.Text strong={strong} ellipsis={{ tooltip: label }} style={maxWidth ? { maxWidth } : undefined}>
      <Link to={`/problems/${id}${search}`}>{label}</Link>
    </Typography.Text>
  )
}

export function UserLink({ name, strong }: { name: string; strong?: boolean }) {
  return (
    <Typography.Text strong={strong} ellipsis={{ tooltip: name }} style={{ maxWidth: '100%' }}>
      <Link to={`/users/${name}`}>{name}</Link>
    </Typography.Text>
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
