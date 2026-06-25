import { Tag } from 'antd'

import { useLocale } from '../locale'

export function AssignmentStatus({ status }: { status: string }) {
  const { text } = useLocale()
  if (status === 'ended') {
    return <Tag>{text.assignments.ended}</Tag>
  }
  return <Tag color="success">{text.assignments.running}</Tag>
}

export function ContestStatus({ status }: { status: string }) {
  const { text } = useLocale()
  const color = status === 'ended' ? undefined : status === 'pending' ? 'processing' : status === 'frozen' ? 'warning' : 'success'
  const label = status === 'ended' ? text.contests.ended : status === 'pending' ? text.contests.pending : status === 'frozen' ? text.contests.frozen : text.contests.running
  return <Tag color={color}>{label}</Tag>
}

export function SubmissionStatus({ status }: { status: string }) {
  const { text } = useLocale()
  return <Tag color={submissionStatusColor(status)}>{text.submissions.statuses[status as keyof typeof text.submissions.statuses] ?? status}</Tag>
}

export function submissionStatusColor(status: string) {
  switch (status) {
    case 'AC':
      return 'success'
    case 'queued':
    case 'judging':
      return 'processing'
    case 'CE':
      return undefined
    case 'TLE':
    case 'MLE':
    case 'OLE':
      return 'warning'
    case 'WA':
    case 'PE':
    case 'RE':
    case 'SE':
      return 'error'
    default:
      return undefined
  }
}
