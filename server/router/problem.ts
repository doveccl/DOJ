import * as Router from 'koa-router'

import Auth from '../middleware/auth'
import Problem from '../model/problem'

const EXCLUDE_LIST = ['solve', 'submit', 'createdAt', 'updatedAt']

const router = new Router()

router.get('/problem', async ctx => {
	let { all, page, size, search } = ctx.query
	let searchRegExp = new RegExp(search)
	let condition: any = { $or: [
		{ tags: search },
		{ title: searchRegExp },
		{ content: searchRegExp }
	] }
	if (!search) { delete condition.$or }
	if (all === undefined || !all) {
		condition = {
			...condition,
			data: { $exists: true },
			contest: { $exists: false },
		}
	} else if (!ctx.user.admin) {
		throw new Error('permission denied')
	}
	page = parseInt(page) || 1
	size = parseInt(size) || 50
	const total = await Problem.countDocuments(condition)
	const list = await Problem.find(condition)
		.select(all ? '-content' : 'title tags solve submit')
		.skip(size * (page - 1)).limit(size)
	ctx.body = { total, list }
})

router.get('/problem/:id', async ctx => {
	ctx.body = await Problem.findById(ctx.params.id).select('-data')
})

router.post('/problem', Auth({ type: 'admin' }), async ctx => {
	for (let item of EXCLUDE_LIST) { delete ctx.request.body[item] }
	ctx.body = await Problem.create(ctx.request.body)
})

router.put('/problem/:id', Auth({ type: 'admin' }), async ctx => {
	for (let item of EXCLUDE_LIST) { delete ctx.request.body[item] }
	ctx.body = await Problem.findByIdAndUpdate(ctx.params.id, ctx.request.body)
})

router.del('/problem/:id', Auth({ type: 'admin' }), async ctx => {
	ctx.body = await Problem.findByIdAndRemove(ctx.params.id)
})

export default router
