import { Flex, Space, Tag, Tooltip, Typography } from 'antd'
import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'

import { EntityTag } from './entity'

type TagListProps = {
  tags: string[]
  color?: string
  maxWidth?: number
  linkTo?: (tag: string) => string
  empty?: ReactNode
}

export function TagList({ tags, color, maxWidth = 96, linkTo, empty = null }: TagListProps) {
  if (tags.length === 0) {
    return empty
  }
  if (tags.length <= 2) {
    return (
      <Space size={[4, 4]} wrap>
        {tags.map((tag) => (
          <TagItem key={tag} tag={tag} color={color} maxWidth={maxWidth} linkTo={linkTo} />
        ))}
      </Space>
    )
  }

  const folded = tags.slice(1)
  const hidden = tags.slice(2)
  return (
    <Space size={[4, 4]} wrap>
      <TagItem tag={tags[0]} color={color} maxWidth={maxWidth} linkTo={linkTo} />
      <Tooltip title={<TagTooltip tags={folded} color={color} maxWidth={maxWidth} linkTo={linkTo} />}>
        <Tag color={color} className="nowrap">
          <span style={{ display: 'inline-flex', alignItems: 'center', maxWidth }}>
            <Typography.Text ellipsis={{ tooltip: tags[1] }} style={{ maxWidth: Math.max(24, maxWidth - 32), lineHeight: 'inherit' }}>
              {linkTo ? <Link to={linkTo(tags[1])}>{tags[1]}</Link> : tags[1]}
            </Typography.Text>
            <span style={{ flex: 'none' }}>...+{hidden.length}</span>
          </span>
        </Tag>
      </Tooltip>
    </Space>
  )
}

function TagItem({ tag, color, maxWidth, linkTo }: { tag: string; color?: string; maxWidth: number; linkTo?: (tag: string) => string }) {
  return linkTo ? (
    <EntityTag color={color} maxWidth={maxWidth} title={tag}>
      <Link to={linkTo(tag)}>{tag}</Link>
    </EntityTag>
  ) : (
    <EntityTag color={color} maxWidth={maxWidth}>{tag}</EntityTag>
  )
}

function TagTooltip({ tags, color, maxWidth, linkTo }: { tags: string[]; color?: string; maxWidth: number; linkTo?: (tag: string) => string }) {
  return (
    <Flex gap={4} wrap style={{ maxWidth: 260 }}>
      {tags.map((tag) => (
        <TagItem key={tag} tag={tag} color={color} maxWidth={maxWidth} linkTo={linkTo} />
      ))}
    </Flex>
  )
}
