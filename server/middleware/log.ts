import { Middleware } from 'koa'
import { logHttp } from '../util/log'

export default (): Middleware =>
	async (ctx, next) => {
		const stime = Date.now()
		await next()
		logHttp.info(ctx.method, ctx.url, `${Date.now() - stime}ms`)
		logHttp.debug('request headers:', ctx.request.headers)
		logHttp.debug('request body:', ctx.request.body)
		logHttp.debug('response:', ctx.body)
	}
