import * as Router from 'koa-router'

import Auth from '../middleware/auth'
import Problem from '../model/problem'
import Submission from '../model/submission'
import { toStringCompare } from '../util/function'

const router = new Router()

router.get('/submission', async ctx => {
	let { page, size } = ctx.query
	page = parseInt(page) || 1
	size = parseInt(size) || 50
	const total = await Submission.countDocuments()
	const list = await Submission.find()
		.select('-cases -code')
		.populate('user', 'name')
		.populate('problem', 'title')
		.populate('contest', 'title')
		.skip(size * (page - 1)).limit(size)
	ctx.body = { total, list }
})

router.get('/submission/:id', async ctx => {
	ctx.body = await Submission.findById(ctx.params.id)
		.populate('user', 'name')
		.populate('problem', 'title')
		.populate('contest', 'title endAt')
	if (
		!ctx.body.open && !ctx.user.admin &&
		Date.now() <= ctx.body.contest.endAt &&
		!toStringCompare(ctx.body.user._id, ctx.user._id)
	) {
		delete ctx.body.code
	}
})

router.post('/submission', async ctx => {
	let { problem, contest, code, language, open } = ctx.request.body
	const p = await Problem.findById(problem)
	if (!p) { throw new Error('problem not found') }
	ctx.body = await Submission.create({
		user: ctx.user._id,
		problem, contest,
		code, language, open
	})
})

router.del('/submission/:id', Auth({ type: 'admin' }), async ctx => {
	const submission = await Submission.findById(ctx.params.id)
	if (!submission) { throw new Error('submission not found') }
	ctx.body = await submission.remove()
})

export default router
