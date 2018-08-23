import * as Body from 'koa-body'
import * as Compose from 'koa-compose'

import { Middleware } from 'koa'
import { logServer } from '../util/log'

export default (): Middleware => Compose([
	Body({ multipart: true }),
	async (ctx, next) => {
		try {
			await next()
			if (ctx.type !== 'application/json') { return }
			ctx.body = { success: true, data: ctx.body }
		} catch(e) {
			logServer.error(e.message)
			logServer.debug(e.stack)
			ctx.body = { success: false, message: e.message }
		}
	}
])
