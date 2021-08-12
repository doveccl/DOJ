import Router from 'koa-router'

import { Group, Status } from '../../common/interface'
import { ensureGroup } from '../../common/user'
import { group, token } from '../middleware/auth'
import { contest, fetch } from '../middleware/fetch'
import { File } from '../model/file'
import { DProblem, Problem } from '../model/problem'
import { Submission } from '../model/submission'
import { DUser } from '../model/user'

const router = new Router<any, {
	self: DUser
	problem: DProblem
}>()

router.use('/problem', token())

router.get('/problem', async (ctx) => {
	const all = ctx.query.all as string
	const cid = ctx.query.cid as string
	let search = ctx.query.search as string
	const page = Number(ctx.query.page) || 1
	const size = Number(ctx.query.size) || 50

	if (!search) { search = '' }
	const esc = search.replace(/[-\/\\^$*+?.()|[\]{}]/g, '\\$&')
	const searchRegExp = new RegExp(esc, 'i')
	const condition = {
		'contest.id': cid,
		'$or': [
			{ tags: [search] },
			{ title: searchRegExp },
			{ content: searchRegExp }
		]
	}
	if (!search) { delete condition.$or }
	if (!cid) { delete condition['contest.id'] }
	if (all) { ensureGroup(ctx.self, Group.admin) }

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
			if (new Date() < enableAt) continue
		}
		list.push(Object.assign({
			solved: await Submission.countDocuments({
				pid: item._id,
				uid: ctx.self._id,
				'result.status': Status.AC
			})
		}, item.toJSON()))
	}
	ctx.body = { total, list }
})

router.get('/problem/:id', fetch('problem'), async (ctx) => {
	if (ctx.problem.contest) {
		const c = await contest(ctx.problem.contest.id)
		if (new Date() < c.startAt) { ensureGroup(ctx.self, Group.admin) }
	}
	ctx.body = ctx.problem.toJSON()
	delete ctx.body.data
})

router.post('/problem', group(Group.admin), async (ctx) => {
	ctx.body = await Problem.create(ctx.request.body)
	await File.findByIdAndUpdate(ctx.request.body.data, { metadata: { type: 'data' } })
})

router.put('/problem/:id', group(Group.admin), fetch('problem'), async (ctx) => {
	await File.findByIdAndUpdate(ctx.request.body.data, { metadata: { type: 'data' } })
	ctx.body = await ctx.problem.updateOne(ctx.request.body, { runValidators: true })
	ctx.problem.set(ctx.request.body)
})

router.del('/problem/:id', group(Group.admin), fetch('problem'), async (ctx) => {
	ctx.body = await ctx.problem.remove()
})

export default router
