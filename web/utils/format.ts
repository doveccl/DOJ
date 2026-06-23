import type { Assignment, Problem } from '../client'

export function problemCode(id: number) {
  return `P${id}`
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
