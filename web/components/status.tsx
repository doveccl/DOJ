import { Tag } from 'antd'

import { useLocale } from '../locale'

export function AssignmentStatus({ status }: { status: string }) {
  const { text } = useLocale()
  if (status === 'ended') {
    return <Tag>{text.assignments.ended}</Tag>
  }
  return <Tag color="green">{text.assignments.running}</Tag>
}

export function ContestStatus({ status }: { status: string }) {
  const { text } = useLocale()
  const color = status === 'ended' ? 'default' : status === 'pending' ? 'gold' : status === 'frozen' ? 'blue' : 'green'
  const label = status === 'ended' ? text.contests.ended : status === 'pending' ? text.contests.pending : status === 'frozen' ? text.contests.frozen : text.contests.running
  return <Tag color={color}>{label}</Tag>
}

export function SubmissionStatus({ status }: { status: string }) {
  const color = status === 'AC' ? 'green' : status === 'queued' ? 'blue' : status === 'judging' ? 'geekblue' : status === 'CE' ? 'default' : status === 'TLE' || status === 'MLE' ? 'gold' : 'red'
  return <Tag color={color}>{status}</Tag>
}
