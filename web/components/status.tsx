import { Tag, Typography } from 'antd'

import { useLocale } from '../locale'

export function SubmissionStatus({ status }: { status: string }) {
  const { text } = useLocale()
  return <Tag color={submissionStatusColor(status)}>{text.submissions.statuses[status as keyof typeof text.submissions.statuses] ?? status}</Tag>
}

export function ProblemStatus({ status }: { status?: string }) {
  const { text } = useLocale()
  if (status === 'pending') {
    return <Tag color="processing">{text.submissions.statuses.pending}</Tag>
  }
  if (status === 'ac') {
    return <Tag color="success">{text.problem.passed}</Tag>
  }
  if (status === 'tried') {
    return <Tag color="warning">{text.problem.tried}</Tag>
  }
  return <Typography.Text type="secondary">-</Typography.Text>
}

function submissionStatusColor(status: string) {
  switch (status) {
    case 'AC':
      return 'green'
    case 'pending':
    case 'queued':
      return 'blue'
    case 'judging':
      return 'geekblue'
    case 'CE':
      return 'purple'
    case 'TLE':
      return 'orange'
    case 'MLE':
      return 'volcano'
    case 'OLE':
      return 'lime'
    case 'WA':
      return 'red'
    case 'PE':
      return 'gold'
    case 'RE':
      return 'magenta'
    case 'SE':
      return 'cyan'
    default:
      return undefined
  }
}
