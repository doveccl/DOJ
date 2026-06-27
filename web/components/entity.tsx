import { Typography } from 'antd'
import { Link } from 'react-router-dom'

import { problemLabel } from '../utils/format'

export function ProblemLink({ id, title, strong }: { id: number; title?: string; strong?: boolean }) {
  return (
    <Typography.Text strong={strong} ellipsis className="lineText">
      <Link to={`/problems/${id}`}>{problemLabel(id, title)}</Link>
    </Typography.Text>
  )
}

export function UserLink({ name, strong }: { name: string; strong?: boolean }) {
  return (
    <Typography.Text strong={strong} className="nowrap">
      <Link to={`/users/${name}`}>{name}</Link>
    </Typography.Text>
  )
}
