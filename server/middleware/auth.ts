import { compareSync } from 'bcryptjs'
import { Middleware } from 'koa'

import { Group } from '../../common/interface'
import { ensureGroup } from '../../common/user'
import { DUser, User } from '../model/user'
import { verify } from '../util/jwt'

declare module 'koa' {
	interface Context {
		self?: DUser
	}
}

export function password(): Middleware {
	return async (ctx, next) => {
		const user = ctx.get('user')
		const pass = ctx.get('password')
		const condition = [{ name: user }, { mail: user }]
		ctx.self = await User.findOne({ $or: condition })
		if (!ctx.self || !compareSync(pass, ctx.self.password)) {
			throw new Error('invalid user or password')
		}
		await next()
	}
}

export function token(cookie = false, k = 'token'): Middleware {
	return async (ctx, next) => {
		const tkn = ctx.get(k) || cookie && ctx.cookies.get(k)
		const data: any = await verify(tkn || '')
		ctx.self = await User.findById(data.id)
		if (!ctx.self) { throw new Error('login required') }
		await next()
	}
}

export function group(ugroup: Group): Middleware {
	return async (ctx, next) => {
		ensureGroup(ctx.self, ugroup)
		await next()
	}
}
