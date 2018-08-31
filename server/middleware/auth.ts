import { Middleware } from 'koa'
import { compareSync } from 'bcryptjs'

import User, { IUser, UserGroup } from '../model/user'
import { verify } from '../util/jwt'

declare module "koa" {
	interface Context {
		self?: IUser
	}
}

export function password(): Middleware {
	return async (ctx, next) => {
		const user = ctx.get('user')
		const password = ctx.get('password')
		const name = user, mail = user
		ctx.self = await User.findOne({ $or: [{ name }, { mail }] })
		if (!ctx.self || !compareSync(password, ctx.self.password)) {
			throw new Error('invalid user or password')
		}
		await next()
	}
}

export function token(getCookie = false): Middleware {
	return async (ctx, next) => {
		let token: string = ctx.get('token')
		if (!token && getCookie && ctx.method === 'GET') {
			token = ctx.cookies.get('token')
		}
		const data: any = await verify(token)
		ctx.self = await User.findById(data.id)
		if (!ctx.self) { throw new Error('login required') }
		await next()
	}
}

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

export function ensureGroup(user: IUser, type: string | number, diff = 0) {
	if (!isGroup(user, type, diff)) {
		throw new Error('permission denied')
	}
}

export function forGroup(type: 'root' | 'admin' | 'common'): Middleware {
	return async (ctx, next) => {
		ensureGroup(ctx.self, type)
		await next()
	}
}
