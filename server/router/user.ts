import * as Route from 'koa-router'
import { hashSync } from 'bcryptjs'

import Auth from '../middleware/auth'
import User from '../model/user'

const router = new Route()

router.get('/user', async ctx => {
	let { rank, page, size } = ctx.query
	if (!rank && !ctx.user.admin) {
		throw new Error('permission denied')
	}
	page = parseInt(page) || 1
	size = parseInt(size) || 10
	const total = await User.countDocuments()
	const list = await User.find()
		.select(rank ? '-password -admin' : '')
		.sort(rank ? '-solve submit name' : '')
		.skip(size * (page - 1)).limit(size)
	ctx.body = { total, list }
})

router.post('/user', Auth({ type: 'admin' }), async ctx => {
	const p = ctx.request.body.password
	if (p) { ctx.request.body.password = hashSync(p) }
	ctx.body = await User.create(ctx.request.body)
})

router.put('/user/:id', Auth({ type: 'admin' }), async ctx => {
	const p = ctx.request.body.password
	if (p) { ctx.request.body.password = hashSync(p) }
	ctx.body = await User.findByIdAndUpdate(ctx.params.id, ctx.request.body)
})

router.del('/user/:id', Auth({ type: 'admin' }), async ctx => {
	ctx.body = await User.findByIdAndRemove(ctx.params.id)
})

export default router
