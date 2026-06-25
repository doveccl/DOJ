import { Flex, Statistic, Typography } from 'antd'

import { formatTime } from '../utils/format'

type DeadlineKind = 'assignment' | 'contest'

type DeadlineTimerProps = {
  kind: DeadlineKind
  status: string
  target: string
  lang: string
  range?: string
  strong?: boolean
  align?: 'flex-start' | 'flex-end'
  onFinish?: () => void
}

export function DeadlineTimer({
  kind,
  status,
  target,
  lang,
  range,
  strong = false,
  align = 'flex-start',
  onFinish
}: DeadlineTimerProps) {
  const targetMs = new Date(target).getTime()
  if (!Number.isFinite(targetMs)) {
    return <Typography.Text>-</Typography.Text>
  }

  return (
    <Flex vertical gap={0} align={align}>
      {status === 'ended' ? (
        <Typography.Text className="nowrap">{formatTime(target, lang)}</Typography.Text>
      ) : (
        <Statistic.Timer
          type="countdown"
          value={targetMs}
          format={timerFormat(targetMs, lang)}
          suffix={timerSuffix(status, kind, lang)}
          onFinish={onFinish}
          styles={{
            content: {
              fontSize: strong ? 20 : 14,
              fontWeight: strong ? 600 : 500,
              lineHeight: strong ? '28px' : '22px',
              fontVariantNumeric: 'tabular-nums'
            }
          }}
        />
      )}
      {range ? (
        <Typography.Text type="secondary" className="nowrap">
          {range}
        </Typography.Text>
      ) : null}
    </Flex>
  )
}

export function contestTarget(status: string, startAt: string, endAt: string) {
  return status === 'pending' ? startAt : endAt
}

function timerFormat(targetMs: number, lang: string) {
  const seconds = Math.max(0, Math.ceil((targetMs - Date.now()) / 1000))
  if (seconds >= 24 * 60 * 60) {
    return lang === 'zh' ? 'D[天] HH:mm:ss' : 'D[d] HH:mm:ss'
  }
  return 'HH:mm:ss'
}

function timerSuffix(status: string, kind: DeadlineKind, lang: string) {
  if (status === 'pending') {
    return lang === 'zh' ? ' 后开始' : ' to start'
  }
  if (kind === 'assignment') {
    return lang === 'zh' ? ' 后截止' : ' until due'
  }
  return lang === 'zh' ? ' 后结束' : ' left'
}
