import * as Router from 'koa-router'

import Problem from '../model/problem'
import Submission from '../model/submission'

import { urlFetch } from '../middleware/fetch'
import { UserGroup as G } from '../model/user'
import { checkGroup } from '../middleware/auth'
import { toStringCompare } from '../util/function'

const router = new Router()

router.get('/submission', async ctx => {
	let { page, size } = ctx.query
	page = parseInt(page) || 1
	size = parseInt(size) || 50
	const total = await Submission.countDocuments()
	const list = await Submission.find()
		.select('-cases -code')
		.skip(size * (page - 1)).limit(size)
	ctx.body = { total, list }
})

router.get('/submission/:id', async ctx => {
	if (
		!ctx.body.open && !checkGroup(ctx.self, G.admin) &&
		!toStringCompare(ctx.body.user._id, ctx.self._id)
	) {
		// can not get code
	}
})

router.post('/submission', async ctx => {
	let { problem, contest, code, language, open } = ctx.request.body
	const p = await Problem.findById(problem)
	if (!p) { throw new Error('problem not found') }
	ctx.body = await Submission.create({
		user: ctx.self._id,
		problem, contest,
		code, language, open
	})
})

export default router
