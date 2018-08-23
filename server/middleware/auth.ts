import { Middleware } from 'koa'
import { compareSync } from 'bcryptjs'

import User, { IUser, UserGroup } from '../model/user'
import { verify } from '../util/jwt'

declare module "koa" {
	interface Context {
		self?: IUser
	}
}

export function check(user: IUser, group: UserGroup, diff = 0) {
	return user && (user.group - group >= diff)
}

export function guard(user: IUser, group: UserGroup, diff = 0) {
	if (!check(user, group, diff)) {
		throw new Error('permission denied')
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

export function token(): Middleware {
	return async (ctx, next) => {
		const token: string = ctx.get('token')
		const data: any = await verify(token)
		ctx.self = await User.findById(data.id)
		if (!ctx.self) { throw new Error('login required') }
		await next()
	}
}

export function group(type = UserGroup.admin): Middleware {
	return async (ctx, next) => {
		guard(ctx.self, type)
		await next()
	}
}
