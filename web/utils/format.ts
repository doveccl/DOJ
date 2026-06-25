import type { Assignment, Problem } from '../client'

export function problemCode(id: number) {
  return `P${id}`
}

export function submissionCode(id: number) {
  return `S${id}`
}

export function caseCode(no: number) {
  return `C${no}`
}

export function formatLimit(problem: Pick<Problem, 'timeMs' | 'memoryMb'>) {
  return `${problem.timeMs}ms / ${problem.memoryMb}MB`
}

export function formatPass(problem: Pick<Problem, 'ac' | 'submit'>) {
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
  return `${(value / 1024 / 1024).toFixed(1)}MB`
}

export function memoryText(kb?: number) {
  return kb === undefined ? '-' : formatBytes(kb * 1024)
}

export function progress(row: Pick<Assignment, 'done' | 'total'>) {
  return row.total > 0 ? Math.round((row.done / row.total) * 100) : 0
}

export function formatTime(value: string, lang: string) {
  return new Date(value).toLocaleString(lang === 'zh' ? 'zh-CN' : 'en-US', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

export function formatDuration(seconds: number, lang: string) {
  const value = Math.max(0, Math.floor(seconds))
  if (value < 60) {
    return lang === 'zh' ? `${value} 秒` : `${value}s`
  }
  const minutes = Math.floor(value / 60)
  if (minutes < 60) {
    return lang === 'zh' ? `${minutes} 分钟` : `${minutes}m`
  }
  const hours = Math.floor(minutes / 60)
  if (hours < 24) {
    return lang === 'zh' ? `${hours} 小时` : `${hours}h`
  }
  const days = Math.floor(hours / 24)
  return lang === 'zh' ? `${days} 天` : `${days}d`
}
