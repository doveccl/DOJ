import Router from 'koa-router'

import { Group } from '../../common/interface'
import { group, token } from '../middleware/auth'
import { fetch } from '../middleware/fetch'
import { Contest, DContest } from '../model/contest'
import { Problem } from '../model/problem'

const router = new Router<any, { contest: DContest }>()

router.use('/contest', token())

router.get('/contest', async (ctx) => {
	const page = Number(ctx.query.page) || 1
	const size = Number(ctx.query.size) || 50
	const total = await Contest.countDocuments()
	const list = await Contest.find()
		.sort({ _id: -1 })
		.select('-description')
		.skip(size * (page - 1)).limit(size)
	ctx.body = { total, list }
})

router.post('/contest', group(Group.admin), async (ctx) => {
	const { startAt, endAt, freezeAt } = ctx.request.body
	if (startAt >= endAt) { throw new Error('invalid datetime range') }
	if (freezeAt >= endAt) { throw new Error('invalid freeze time') }
	ctx.body = await Contest.create(ctx.request.body)
})

router.get('/contest/:id', fetch('contest'), async (ctx) => {
	ctx.body = ctx.contest
})

router.put('/contest/:id', group(Group.admin), fetch('contest'), async (ctx) => {
	const { startAt, endAt, freezeAt } = ctx.request.body
	const { startAt: s, endAt: e, freezeAt: f } = ctx.contest
	if (
		(startAt >= endAt) ||
		(startAt !== undefined && endAt === undefined && new Date(startAt) >= e) ||
		(startAt === undefined && endAt !== undefined && new Date(endAt) <= s)
	) { throw new Error('invalid datetime range') }
	if (
		(freezeAt >= endAt) ||
		(freezeAt !== undefined && endAt === undefined && new Date(freezeAt) >= e) ||
		(freezeAt === undefined && endAt !== undefined && new Date(endAt) <= f)
	) { throw new Error('invalid datetime range') }
	ctx.body = await ctx.contest.updateOne(ctx.request.body, { runValidators: true })
	ctx.contest.set(ctx.request.body)
})

router.del('/contest/:id', group(Group.admin), fetch('contest'), async (ctx) => {
	ctx.body = await ctx.contest.remove()
	await Problem.updateMany(
		{ 'contest.id': ctx.params.id },
		{ $unset: { contest: 1 } }
	)
})

export default router
