import { Middleware } from 'koa'
import { compareSync } from 'bcryptjs'

import User, { IUser } from '../model/user'
import { verify } from '../util/jwt'

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
			} else if (type === 'admin') {
				if (!ctx.user || ctx.user.admin === 0) {
					throw new Error('permission denied')
				}
			} else {
				const token: string = ctx.get('token')
				const data: any = await verify(token)
				ctx.user = await User.findById(data.id)
			}
		}
		await next()
	}
