import * as Body from 'koa-body'
import * as Compose from 'koa-compose'

import { Middleware } from 'koa'
import { logServer } from '../util/log'

export default (): Middleware => Compose([
	Body({
		multipart: true,
		formidable: {
			maxFileSize: 200 * 1024 * 1024
		}
	}),
	async (ctx, next) => {
		/**
		 * 'createdAt' and 'updatedAt' are maintained automatically
		 * user (include admin) should not modify them manually
		 */
		if (ctx.method === 'PUT') {
			delete ctx.request.body.createdAt
			delete ctx.request.body.updatedAt
		}
		/**
		 * handle errors and response objects
		 * wrap them in a uniform format
		 */
		try {
			await next()
			if (ctx.type !== 'application/json') { return }
			ctx.body = { success: true, data: ctx.body }
		} catch (e) {
			logServer.error(e.message)
			logServer.debug(e.stack)
			if (/(:?login|jwt)/i.test(e.message)) {
				ctx.status = 401
			} else if (/permission/i.test(e.message)) {
				ctx.status = 403
			} else if (/not found/i.test(e.message)) {
				ctx.status = 404
			} else { ctx.status = 400 }
			ctx.body = { success: false, message: e.message }
		}
	}
])
