import { Middleware } from 'koa'

export interface IAuthProps {
	exclude?: RegExp;
}

export default ({ exclude }: IAuthProps): Middleware =>
	async (ctx, next) => {
		if (!(exclude && exclude.test(ctx.path))) {
			console.log('start auth')
		}
		await next()
	}
