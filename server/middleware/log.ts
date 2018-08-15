import { Middleware } from 'koa'
import { logHttp } from '../util/log'

export default (): Middleware =>
	async (ctx, next) => {
		const stime = Date.now()
		await next()
		logHttp.info(ctx.method, ctx.url, `${Date.now() - stime}ms`)
	}
