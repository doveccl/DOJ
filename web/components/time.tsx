import { Flex, Statistic, Tag, Tooltip, Typography } from 'antd'

import { useLocale } from '../locale'
import { formatTime } from '../utils/format'

type DeadlineKind = 'assignment' | 'contest'
type TimerCopy = {
  prefix: string
  suffix: string
}

type DeadlineTimerProps = {
  kind: DeadlineKind
  status: string
  target: string
  range?: string
  strong?: boolean
  align?: 'flex-start' | 'flex-end'
  onFinish?: () => void
}

type ScheduleTagProps = {
  kind: DeadlineKind
  status: string
  target: string
  range?: string
  onFinish?: () => void
}

export function DeadlineTimer({
  kind,
  status,
  target,
  range,
  strong = false,
  align = 'flex-start',
  onFinish
}: DeadlineTimerProps) {
  const { lang, text } = useLocale()
  const targetMs = new Date(target).getTime()
  if (!Number.isFinite(targetMs)) {
    return <Typography.Text>-</Typography.Text>
  }

  return (
    <Flex vertical gap={0} align={align}>
      {status === 'ended' ? (
        <Typography.Text className="nowrap">{formatTime(target, lang)}</Typography.Text>
      ) : (
        <TimerText copy={timerCopy(status, kind, text)} targetMs={targetMs} strong={strong} onFinish={onFinish} />
      )}
      {range ? (
        <Typography.Text type="secondary" className="nowrap">
          {range}
        </Typography.Text>
      ) : null}
    </Flex>
  )
}

export function ScheduleTag({ kind, status, target, range, onFinish }: ScheduleTagProps) {
  const { lang, text } = useLocale()
  const targetMs = new Date(target).getTime()
  const tooltip = range ?? formatTime(target, lang)
  if (!Number.isFinite(targetMs)) {
    return <Tag>-</Tag>
  }
  if (status === 'ended') {
    return (
      <Tooltip title={tooltip}>
        <Tag>{kind === 'assignment' ? text.assignments.ended : text.contests.ended}</Tag>
      </Tooltip>
    )
  }
  return (
    <Tooltip title={tooltip}>
      <Tag color={status === 'pending' ? 'processing' : status === 'frozen' ? 'warning' : 'success'}>
        <TimerText copy={timerCopy(status, kind, text)} targetMs={targetMs} compact onFinish={onFinish} />
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
  compact = false,
  strong = false,
  onFinish
}: {
  copy: TimerCopy
  targetMs: number
  compact?: boolean
  strong?: boolean
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
          fontSize: compact ? 12 : strong ? 20 : 14,
          fontWeight: strong ? 600 : 500,
          lineHeight: compact ? '20px' : strong ? '28px' : '22px',
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
  return text.time.endsIn
}
