import { Group } from './interface'

interface HasGroup {
	group?: Group
}

export function diffGroup(user: HasGroup, group: Group, diff = 0) {
	return user && user.group - group >= diff
}

export function ensureGroup(user: HasGroup, group: Group, diff = 0) {
	if (!diffGroup(user, group, diff)) {
		throw new Error('permission denied')
	}
}
