import { durationText, localeCode } from '../locale'
import type { Lang } from '../locale'

type ProblemLimit = {
  timeMs: number
  memoryMb: number
}

type ProblemPass = {
  ac: number
  submit: number
}

type AssignmentProgressLike = {
  done: number
  total: number
}

export function problemCode(id: number) {
  return `P${id}`
}

export function problemLabel(id: number, title?: string) {
  return [problemCode(id), title].filter(Boolean).join(' ')
}

export function submissionCode(id: number) {
  return `#${id}`
}

export function caseCode(no: number) {
  return String(no)
}

export function formatLimit(problem: ProblemLimit) {
  return `${problem.timeMs}ms / ${problem.memoryMb}MB`
}

export function formatPass(problem: ProblemPass) {
  const percent = problem.submit > 0 ? Math.round((problem.ac / problem.submit) * 100) : 0
  return `${problem.ac}/${problem.submit} (${percent}%)`
}

export function formatBytes(value: number) {
  if (value < 1024) {
    return `${value}B`
  }
  if (value < 1024 * 1024) {
    return `${Math.round(value / 1024)}KB`
  }
  if (value < 1024 * 1024 * 1024) {
    return `${(value / 1024 / 1024).toFixed(1)}MB`
  }
  return `${(value / 1024 / 1024 / 1024).toFixed(1)}GB`
}

export function memoryText(kb?: number) {
  return kb === undefined ? '-' : formatBytes(kb * 1024)
}

export function isLiveSubmissionStatus(status?: string) {
  return status === 'queued' || status === 'judging'
}

export function progress(row: AssignmentProgressLike) {
  return row.total > 0 ? Math.round((row.done / row.total) * 100) : 0
}

export function formatTime(value: string, lang: Lang) {
  return new Date(value).toLocaleString(localeCode(lang), {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

export function formatShortTime(value: string, lang: Lang) {
  return new Date(value).toLocaleString(localeCode(lang), {
    year: '2-digit',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

export function formatDuration(seconds: number, lang: Lang) {
  const value = Math.max(0, Math.floor(seconds))
  if (value < 60) {
    return durationText(value, 'second', lang)
  }
  const minutes = Math.floor(value / 60)
  if (minutes < 60) {
    return durationText(minutes, 'minute', lang)
  }
  const hours = Math.floor(minutes / 60)
  if (hours < 24) {
    return durationText(hours, 'hour', lang)
  }
  const days = Math.floor(hours / 24)
  return durationText(days, 'day', lang)
}
