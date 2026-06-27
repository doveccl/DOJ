import { Tag } from 'antd'

import { useLocale } from '../locale'

export function SubmissionStatus({ status }: { status: string }) {
  const { text } = useLocale()
  return <Tag color={submissionStatusColor(status)}>{text.submissions.statuses[status as keyof typeof text.submissions.statuses] ?? status}</Tag>
}

function submissionStatusColor(status: string) {
  switch (status) {
    case 'AC':
      return 'green'
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
