import { Space, Tag, Tooltip } from 'antd'
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
  if (tags.length <= 3) {
    return (
      <Space size={[4, 4]} wrap>
        {tags.map((tag) => (
          <TagItem key={tag} tag={tag} color={color} maxWidth={maxWidth} linkTo={linkTo} />
        ))}
      </Space>
    )
  }

  const inlineTags = tags.slice(0, 2)
  const foldedTags = tags.slice(2)
  return (
    <Space size={[4, 4]} wrap>
      {inlineTags.map((tag) => (
        <TagItem key={tag} tag={tag} color={color} maxWidth={maxWidth} linkTo={linkTo} />
      ))}
      <Tooltip title={foldedTags.join(', ')}>
        <Tag color={color}>
          +{foldedTags.length}
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
