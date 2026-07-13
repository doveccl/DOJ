import { describe, expect, it } from 'vitest'

import { formatBytes, formatLimit, formatPass, formatShortTime, formatTime, problemCode } from './format'

describe('format helpers', () => {
  it('formats problem identity and limits', () => {
    expect(problemCode(1000)).toBe('P1000')
    expect(formatLimit({ timeMs: 1000, memoryMb: 256 })).toBe('1000ms / 256MB')
  })

  it('formats pass rate and bytes', () => {
    expect(formatPass({ ac: 1, submit: 3 })).toBe('1/3 (33%)')
    expect(formatPass({ ac: 0, submit: 0 })).toBe('0/0 (0%)')
    expect(formatBytes(13)).toBe('13B')
    expect(formatBytes(4096)).toBe('4KB')
  })

  it('includes the year in timestamps', () => {
    expect(formatTime('2020-01-02T03:04:00Z', 'en')).toContain('2020')
    expect(formatShortTime('2020-01-02T03:04:00Z', 'en')).toContain('20')
  })
})
