import { Tag, Typography } from 'antd'

export function ScoreTag({ score }: { score?: number }) {
  if (score === undefined) {
    return <Typography.Text type="secondary">-</Typography.Text>
  }
  return <Tag color={score >= 100 ? 'success' : score > 0 ? 'warning' : 'error'}>{score}</Tag>
}
