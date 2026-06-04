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
  'SE',
  'FROZEN'
] as const

export type JudgeStatus = (typeof judgeStatuses)[number]

export const contestTypes = ['OI', 'ICPC'] as const
export type ContestType = (typeof contestTypes)[number]

export const groupKeys = ['guest', 'user', 'admin'] as const
export type BuiltinGroupKey = (typeof groupKeys)[number]
