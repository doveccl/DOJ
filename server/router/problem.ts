import * as Router from 'koa-router'

import { ensureGroup, forGroup, token } from '../middleware/auth'
import { contest, urlFetch } from '../middleware/fetch'
import { File } from '../model/file'
import { Problem } from '../model/problem'

const router = new Router()

router.use('/problem', token())

router.get('/problem', async (ctx) => {
	const { all, cid } = ctx.query
	let { page, size, search } = ctx.query

	if (!search) { search = '' }
	const esc = search.replace(/[-\/\\^$*+?.()|[\]{}]/g, '\\$&')
	const searchRegExp = new RegExp(esc, 'i')
	const condition = {
		'contest.id': cid,
		'$or': [
			{ tags: search },
			{ title: searchRegExp },
			{ content: searchRegExp }
		]
	}
	if (!search) { delete condition.$or }
	if (!cid) { delete condition['contest.id'] }
	if (all) { ensureGroup(ctx.self, 'admin') }

	page = parseInt(page, 10) || 1
	size = parseInt(size, 10) || 50
	const total = await Problem.countDocuments(condition)
	const arr = await Problem.find(condition)
		.sort(all ? '-_id' : '_id')
		.select(all ? '' : '-content -data')
		.skip(size * (page - 1)).limit(size)
	const list: any[] = []
	for (const item of arr) {
		if (!all && item.contest) {
			const c = await contest(item.contest.id)
			const enableAt = cid ? c.startAt : c.endAt
			if (new Date() < enableAt) { continue }
		}
		list.push(item.toJSON())
	}
	ctx.body = { total, list }
})

router.get('/problem/:id', urlFetch('problem'), async (ctx) => {
	if (ctx.problem.contest) {
		const c = await contest(ctx.problem.contest.id)
		if (new Date() < c.startAt) { ensureGroup(ctx.self, 'admin') }
	}
	ctx.body = ctx.problem.toJSON()
	delete ctx.body.data
})

router.post('/problem', forGroup('admin'), async (ctx) => {
	ctx.body = await Problem.create(ctx.request.body)
	await File.findByIdAndUpdate(ctx.request.body.data, { metadata: { type: 'data' } })
})

router.put('/problem/:id', forGroup('admin'), urlFetch('problem'), async (ctx) => {
	await File.findByIdAndUpdate(ctx.request.body.data, { metadata: { type: 'data' } })
	ctx.body = await ctx.problem.update(ctx.request.body, { runValidators: true })
	ctx.problem.set(ctx.request.body)
})

router.del('/problem/:id', forGroup('admin'), urlFetch('problem'), async (ctx) => {
	ctx.body = await ctx.problem.remove()
})

export default router
