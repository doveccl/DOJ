import { Middleware } from 'koa'

import { IUser } from '../model/user'
import { IProblem } from '../model/problem'
import { ISubmission } from '../model/submission'
import { IFile } from '../model/file'

declare module "koa" {
	interface Context {
		user?: IUser
		problem?: IProblem
		submission?: ISubmission
		file?: IFile
		[index: string]: any
	}
}

export default (model: any): Middleware =>
	async (ctx, next) => {
		if (ctx.params.id) {
			const type = ctx.url.match(/^\/api\/([^/]+)/)[1]
			ctx[type] = await model.findById(ctx.params.id)
			if (!ctx[type]) { throw new Error(`${type} not found`) }
		}
		await next()
	}
