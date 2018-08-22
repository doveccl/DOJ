import { Middleware } from 'koa'
import { logServer } from '../util/log'

export default (): Middleware =>
	async (ctx, next) => {
		try {
			await next()
			if (ctx.type) { return }
			ctx.body = { success: true, data: ctx.body }
		} catch(e) {
			logServer.error(e.message)
			logServer.debug(e.stack)
			ctx.body = { success: false, message: e.message }
		}
	}
