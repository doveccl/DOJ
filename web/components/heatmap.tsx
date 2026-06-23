import { Flex, Tooltip, Typography } from 'antd'

import type { HeatCell } from '../client'
import { useLocale } from '../locale'

export function YearHeatmap({ cells }: { cells: HeatCell[] }) {
  const { text } = useLocale()
  const weeks = buildWeeks(cells)
  const labels = monthLabels(weeks, text.calendar.months)
  const total = cells.reduce((sum, cell) => sum + cell.count, 0)

  return (
    <div className="heatFrame">
      <div className="heatScroller">
        <div className="heatMonths" style={{ gridTemplateColumns: `repeat(${weeks.length}, 14px)` }}>
          {weeks.map((week, index) => (
            <span key={week[0].date} className="heatMonth">
              {labels[index]}
            </span>
          ))}
        </div>
        <div className="heatBody">
          <div className="heatWeekdays">
            {text.calendar.weekdays.map((day, index) => (
              <span key={day}>{index === 1 || index === 3 || index === 5 ? day : ''}</span>
            ))}
          </div>
          <div className="heatWeeks">
            {weeks.map((week) => (
              <div className="heatWeek" key={week[0].date}>
                {week.map((day) => (
                  <Tooltip key={day.date} title={`${day.date}: ${text.home.count(day.count)}`}>
                    <span className={`heatCell heatLevel${heatLevel(day.count)}${day.active ? '' : ' heatInactive'}`} />
                  </Tooltip>
                ))}
              </div>
            ))}
          </div>
        </div>
      </div>
      <Flex align="center" justify="space-between" gap={16} wrap className="heatLegend">
        <Typography.Text type="secondary">{text.home.total(total)}</Typography.Text>
        <Flex align="center" gap={6}>
          <Typography.Text type="secondary">{text.home.less}</Typography.Text>
          {[0, 1, 2, 3, 4].map((level) => (
            <span key={level} className={`heatCell legendCell heatLevel${level}`} />
          ))}
          <Typography.Text type="secondary">{text.home.more}</Typography.Text>
        </Flex>
      </Flex>
    </div>
  )
}

type HeatDay = HeatCell & {
  active: boolean
}

function buildWeeks(cells: HeatCell[]) {
  if (cells.length === 0) {
    return []
  }
  const counts = new Map(cells.map((cell) => [cell.date, cell.count]))
  const first = parseDay(cells[0].date)
  const last = parseDay(cells[cells.length - 1].date)
  const start = new Date(first)
  start.setDate(start.getDate() - start.getDay())
  const end = new Date(last)
  end.setDate(end.getDate() + (6 - end.getDay()))

  const weeks: HeatDay[][] = []
  for (const cursor = new Date(start); cursor <= end; cursor.setDate(cursor.getDate() + 7)) {
    const week: HeatDay[] = []
    for (let offset = 0; offset < 7; offset += 1) {
      const day = new Date(cursor)
      day.setDate(cursor.getDate() + offset)
      const date = formatDay(day)
      week.push({
        date,
        count: counts.get(date) ?? 0,
        active: day >= first && day <= last
      })
    }
    weeks.push(week)
  }
  return weeks
}

function parseDay(value: string) {
  const [year, month, day] = value.split('-').map(Number)
  return new Date(year, month - 1, day)
}

function formatDay(value: Date) {
  const year = value.getFullYear()
  const month = String(value.getMonth() + 1).padStart(2, '0')
  const day = String(value.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function monthLabels(weeks: HeatDay[][], months: string[]) {
  let lastLabelWeek = -5
  return weeks.map((week, index) => {
    const firstDay = week.find((day) => day.active && parseDay(day.date).getDate() === 1)
    if (!firstDay && index !== 0) {
      return ''
    }

    if (index - lastLabelWeek < 4) {
      return ''
    }

    lastLabelWeek = index
    const labelDay = firstDay ?? week.find((day) => day.active) ?? week[0]
    return months[parseDay(labelDay.date).getMonth()]
  })
}

function heatLevel(count: number) {
  if (count <= 0) {
    return 0
  }
  if (count === 1) {
    return 1
  }
  if (count <= 3) {
    return 2
  }
  if (count <= 5) {
    return 3
  }
  return 4
}
