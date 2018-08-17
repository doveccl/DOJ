import { Middleware } from 'koa'
import { compareSync } from 'bcryptjs'

import { verify } from '../util/jwt'
import User, { IUser } from '../model/user'

declare module "koa" {
	interface Context {
		user: IUser
	}
}

export interface IAuthProps {
	type: 'password' | 'token' | 'admin';
	exclude?: RegExp;
}

export default ({ exclude, type }: IAuthProps): Middleware =>
	async (ctx, next) => {
		if (!(exclude && exclude.test(ctx.path))) {
			if (type === 'password') {
				const user: string = ctx.get('user')
				const password: string = ctx.get('password')
				ctx.user = await User.findOne({ $or: [
					{ name: user }, { mail: user }
				] })
				if (!ctx.user || !compareSync(password, ctx.user.password)) {
					throw new Error('invalid user or password')
				}
			} else {
				const token: string = ctx.get('token')
				const data: any = await verify(token)
				ctx.user = await User.findById(data.id)
				if (!ctx.user || type === 'admin' && ctx.user.admin === 0) {
					throw new Error('permission denied')
				}
			}
		}
		await next()
	}
