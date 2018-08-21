import * as Router from 'koa-router'

import Auth from '../middleware/auth'
import Problem from '../model/problem'
import File from '../model/file'

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
	let query = Problem.findById(ctx.params.id)
	if (ctx.user.admin) {
		query = query.populate('data', 'md5')
	} else {
		query = query.select('-data')
	}
	ctx.body = await query.exec()
})

router.post('/problem', Auth({ type: 'admin' }), async ctx => {
	let { body, files } = ctx.request
	if (files.data) {
		const { path, name: filename } = files.data
		const opts = { filename, metadata: { type: 'data' } }
		const data =  await File.create(path, opts)
		body.data = data._id
	}
	for (let item of EXCLUDE_LIST) { delete body[item] }
	ctx.body = await Problem.create(body)
})

router.put('/problem/:id', Auth({ type: 'admin' }), async ctx => {
	const problem = await Problem.findById(ctx.params.id)
	let { body, files } = ctx.request, originData: any = false
	if (files.data) {
		originData = problem.data
		const { path, name: filename } = files.data
		const opts = { filename, metadata: { type: 'data' } }
		const data =  await File.create(path, opts)
		body.data = data._id
	}
	for (let item of EXCLUDE_LIST) { delete body[item] }
	ctx.body = await problem.update(body, { runValidators: true })
	if (originData) { await File.findByIdAndRemove(originData) }
})

router.del('/problem/:id', Auth({ type: 'admin' }), async ctx => {
	const problem = await Problem.findById(ctx.params.id)
	const { data } = problem
	if (data) { await File.findByIdAndRemove(data) }
	ctx.body = await problem.remove()
})

export default router
