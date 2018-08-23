import { Middleware } from 'koa'

import user, { IUser } from '../model/user'
import problem, { IProblem } from '../model/problem'
import submission, { ISubmission } from '../model/submission'
import file, { IFile } from '../model/file'

declare module "koa" {
	interface Context {
		user?: IUser
		problem?: IProblem
		submission?: ISubmission
		file?: IFile
	}
}

const models = { user, problem, submission, file }

interface IFetchProps {
	type: 'user' | 'problem' | 'submission' | 'file'
	check?: boolean
}

export default ({ type, check = true }: IFetchProps): Middleware =>
	async (ctx, next) => {
		if (ctx.params.id) {
			ctx[type] = await models[type].findById(ctx.params.id)
			if (check && !ctx[type]) {
				throw new Error(`${type} not found`)
			}
		}
		await next()
	}
