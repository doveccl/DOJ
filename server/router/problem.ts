import * as Router from 'koa-router'

import Problem from '../model/problem'
import File from '../model/file'
import { UserGroup as G } from '../model/user'

import fetch from '../middleware/fetch'
import { token, check, group, guard } from '../middleware/auth'

const EXCLUDE_LIST = ['solve', 'submit', 'createdAt', 'updatedAt']

const router = new Router()

router.use(token(), fetch({ type: 'problem' }))

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
	} else { guard(ctx.self, G.admin) }

	page = parseInt(page) || 1
	size = parseInt(size) || 50
	const total = await Problem.countDocuments(condition)
	const list = await Problem.find(condition)
		.select(all ? '-content' : 'title tags solve submit')
		.skip(size * (page - 1)).limit(size)
	ctx.body = { total, list }
})

router.get('/problem/:id', async ctx => {
	if (!check(ctx.self, G.admin)) {
		delete ctx.problem.data
	}
	ctx.body = ctx.problem
})

router.post('/problem', group(G.admin), async ctx => {
	let { body, files } = ctx.request
	if (files.data) {
		const { path, name, type } = files.data
		if (type !== 'application/zip') {
			throw new Error('invalid data type')
		}
		body.data = await File.create(path, name, {
			contentType: type, metadata: { type: 'data' }
		})
	}
	for (let item of EXCLUDE_LIST) { delete body[item] }
	ctx.body = await Problem.create(body)
})

router.put('/problem/:id', group(G.admin), async ctx => {
	let { body, files } = ctx.request, originData: string
	if (files.data) {
		originData = String(ctx.problem.data || '')
		const { path, name, type } = files.data
		if (type !== 'application/zip') {
			throw new Error('invalid data type')
		}
		body.data = await File.create(path, name, {
			contentType: type, metadata: { type: 'data' }
		})
	}
	for (let item of EXCLUDE_LIST) { delete body[item] }
	ctx.body = await ctx.problem.update(body, { runValidators: true })
	if (originData) { await File.findByIdAndRemove(originData) }
})

router.del('/problem/:id', group(G.admin), async ctx => {
	const data = String(ctx.problem.data || '')
	ctx.body = await ctx.problem.remove()
	if (data) { await File.findByIdAndRemove(data) }
})

export default router
