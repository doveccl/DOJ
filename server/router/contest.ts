import * as Router from 'koa-router'

import Problem from '../model/problem'
import Contest from '../model/contest'

import { urlFetch } from '../middleware/fetch'
import { token, forGroup } from '../middleware/auth'

const router = new Router()

router.use('/contest', token())

router.get('/contest', async ctx => {
	let { page, size } = ctx.query
	page = parseInt(page) || 1
	size = parseInt(size) || 50
	const total = await Contest.countDocuments()
	const list = await Contest.find()
		.select('-description')
		.skip(size * (page - 1)).limit(size)
	ctx.body = { total, list }
})

router.post('/contest', forGroup('admin'), async ctx => {
	const { startAt, endAt } = ctx.request.body
	if (startAt >= endAt) { throw new Error('invalid datetime range') }
	ctx.body = await Contest.create(ctx.request.body)
})

router.get('/contest/:id', urlFetch('contest'), async ctx => {
	ctx.body = ctx.contest
})

router.put('/contest/:id', forGroup('admin'), urlFetch('contest'), async ctx => {
	const { startAt, endAt } = ctx.request.body
	const { startAt: s, endAt: e } = ctx.contest
	if (
		(startAt >= endAt) ||
		(startAt !== undefined && endAt === undefined && new Date(startAt) >= e) ||
		(startAt === undefined && endAt !== undefined && new Date(endAt) <= s)
	) { throw new Error('invalid datetime range') }
	ctx.body = await ctx.contest.update(ctx.request.body, { runValidators: true })
	ctx.contest.set(ctx.request.body)
})

router.del('/contest/:id', forGroup('admin'), urlFetch('contest'), async ctx => {
	ctx.body = await ctx.contest.remove()
	await Problem.updateMany(
		{ 'contest.id': ctx.params.id },
		{ $unset: { contest: 1 } }
	)
})

export default router
