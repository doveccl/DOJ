import { IUser, UserGroup } from './interface'

export function isGroup(user: IUser, type: string | number, diff = 0) {
	let group: UserGroup
	switch (type) {
		case 'root':
		case 'admin':
		case 'common':
			group = UserGroup[type]
			break
		default:
			group = +new Number(type)
	}
	return user && user.group - group >= diff
}
