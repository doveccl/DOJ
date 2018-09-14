import * as Router from 'koa-router'

import { ensureGroup, forGroup, token } from '../middleware/auth'
import { contest, urlFetch } from '../middleware/fetch'
import { File } from '../model/file'
import { Problem } from '../model/problem'

const router = new Router()

router.use('/problem', token())

async function addData(file: any, problem?: any) {
	const { path, name, type } = file
	if (type !== 'application/zip') {
		throw new Error('invalid data type')
	}
	return await File.create(path, name, {
		contentType: type, metadata: { type: 'data', problem }
	})
}

router.get('/problem', async (ctx) => {
	const { all, search, cid } = ctx.query
	let { page, size } = ctx.query

	const searchRegExp = new RegExp(search)
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
		.select(all ? '-content' : '-content -data')
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
	const { body, files } = ctx.request
	const { data } = files || {} as any

	if (data) { body.data = await addData(data) }
	ctx.body = await Problem.create(body)
	if (!data) { return }

	await File.findByIdAndUpdate(body.data, {
		'metadata.problem': ctx.body._id
	})
})

router.put('/problem/:id', forGroup('admin'), urlFetch('problem'), async (ctx) => {
	const { body, files } = ctx.request
	const { data } = files || {} as any

	if (data) { body.data = await addData(data, ctx.problem._id) }
	ctx.body = await ctx.problem.update(body, { runValidators: true })
	ctx.problem.set(body)
})

router.del('/problem/:id', forGroup('admin'), urlFetch('problem'), async (ctx) => {
	ctx.body = await ctx.problem.remove()
})

export default router
