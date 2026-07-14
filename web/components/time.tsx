import { Statistic, Tag, Tooltip } from 'antd'

import { useLocale } from '../locale'
import { formatTime } from '../utils/format'

type DeadlineKind = 'assignment' | 'contest'
type TimerCopy = {
  prefix: string
  suffix: string
}

type ScheduleTagProps = {
  kind: DeadlineKind
  status: string
  target: string
  range?: string
  onFinish?: () => void
}

export function ScheduleTag({ kind, status, target, range, onFinish }: ScheduleTagProps) {
  const { lang, text } = useLocale()
  const targetMs = new Date(target).getTime()
  const tooltip = range ?? formatTime(target, lang)
  if (!Number.isFinite(targetMs)) {
    return <Tag style={{ marginInlineEnd: 0 }}>-</Tag>
  }
  if (status === 'ended') {
    return (
      <Tooltip title={tooltip}>
        <Tag style={{ marginInlineEnd: 0 }}>{kind === 'assignment' ? text.assignments.ended : text.contests.ended}</Tag>
      </Tooltip>
    )
  }
  return (
    <Tooltip title={tooltip}>
      <Tag color={status === 'pending' ? 'processing' : status === 'frozen' ? 'warning' : 'success'} style={{ marginInlineEnd: 0 }}>
        <TimerText copy={timerCopy(status, kind, text)} targetMs={targetMs} onFinish={onFinish} />
      </Tag>
    </Tooltip>
  )
}

export function contestTarget(status: string, startAt: string, endAt: string) {
  return status === 'pending' ? startAt : endAt
}

function TimerText({
  copy,
  targetMs,
  onFinish
}: {
  copy: TimerCopy
  targetMs: number
  onFinish?: () => void
}) {
  const { text } = useLocale()
  return (
    <Statistic.Timer
      type="countdown"
      value={targetMs}
      format={timerFormat(targetMs, text)}
      prefix={copy.prefix}
      suffix={copy.suffix}
      onFinish={onFinish}
      styles={{
        root: {
          color: 'currentColor'
        },
        content: {
          color: 'currentColor',
          fontSize: 12,
          fontWeight: 500,
          lineHeight: '20px',
          fontVariantNumeric: 'tabular-nums'
        },
        value: {
          color: 'currentColor'
        },
        prefix: {
          color: 'currentColor'
        },
        suffix: {
          color: 'currentColor'
        }
      }}
    />
  )
}

function timerFormat(targetMs: number, text: ReturnType<typeof useLocale>['text']) {
  const seconds = Math.max(0, Math.ceil((targetMs - Date.now()) / 1000))
  return seconds >= 24 * 60 * 60 ? text.time.dayTimer : text.time.shortTimer
}

function timerCopy(status: string, kind: DeadlineKind, text: ReturnType<typeof useLocale>['text']): TimerCopy {
  if (status === 'pending') {
    return text.time.startsIn
  }
  if (kind === 'assignment') {
    return text.time.dueIn
  }
  if (status === 'frozen') {
    return text.time.frozenEndsIn
  }
  return text.time.endsIn
}
