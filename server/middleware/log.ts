import { Middleware } from 'koa'
import { Logger } from 'log4js'

export default (logger: Logger): Middleware => async (ctx, next) => {
	const stime = Date.now()
	await next()
	logger.info(ctx.method, ctx.url, `${Date.now() - stime}ms`)
}
