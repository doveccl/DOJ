export const judgeStatuses = [
  'WAITING',
  'JUDGING',
  'AC',
  'WA',
  'PE',
  'TLE',
  'MLE',
  'OLE',
  'RE',
  'CE',
  'SE'
] as const

export type JudgeStatus = (typeof judgeStatuses)[number]

export const contestTypes = ['OI', 'ICPC'] as const
export type ContestType = (typeof contestTypes)[number]
