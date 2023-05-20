import { Group } from './interface'

type MaybeUser = { group?: Group } | undefined

export function diffGroup(user: MaybeUser, group = Group.common, diff = 0) {
  return user && (user.group ?? Group.common) - group >= diff
}

export function ensureGroup(user: MaybeUser, group = Group.common, diff = 0) {
  if (!diffGroup(user, group, diff)) throw new Error('permission denied')
}
