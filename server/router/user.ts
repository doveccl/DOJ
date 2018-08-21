import * as Router from 'koa-router'
import { hashSync, compareSync } from 'bcryptjs'

import Auth from '../middleware/auth'
import User from '../model/user'

const EXCLUDE_LIST = ['solve', 'submit', 'createdAt', 'updatedAt']

const router = new Router()

router.get('/user', async ctx => {
	let { rank, page, size } = ctx.query
	if (!rank && !ctx.user.admin) {
		throw new Error('permission denied')
	}
	page = parseInt(page) || 1
	size = parseInt(size) || 50
	const total = await User.countDocuments()
	const list = await User.find()
		.select(rank ? '-password -admin' : '')
		.sort(rank ? '-solve submit name' : '')
		.skip(size * (page - 1)).limit(size)
	ctx.body = { total, list }
})

router.post('/user', Auth({ type: 'admin' }), async ctx => {
	const self = ctx.user, body = ctx.request.body
	if (self.admin <= body.admin) {
		throw new Error('invalid admin level')
	}
	if (body.password === undefined) {
		throw new Error('password is required')
	}
	if (!/^.{6,20}$/.test(body.password)) {
		throw new Error('invalid password (length 6-20)')
	}
	for (let item of EXCLUDE_LIST) { delete body[item] }
	ctx.body = await User.create(body)
})

router.put('/user/:id', async ctx => {
	const self = ctx.user, body = ctx.request.body
	const it = await User.findById(ctx.params.id)
	if (self._id !== it._id) {
		if (self.admin <= it.admin) {
			throw new Error('permission denied')
		}
		if (body.password !== undefined) {
			throw new Error('unable to change others password')
		}
	} else if (body.admin !== undefined) {
		throw new Error('unable to change self-level')
	}
	if (body.password !== undefined) {
		if (!/^.{6,20}$/.test(body.password)) {
			throw new Error('invalid password (length 6-20)')
		}
		if (!compareSync(body.oldPassword, self.password)) {
			throw new Error('correct old password is required')
		}
		body.password = hashSync(body.password)
	}
	for (let item of EXCLUDE_LIST) { delete body[item] }
	ctx.body = await it.update(body)
})

router.del('/user/:id', Auth({ type: 'admin' }), async ctx => {
	const self = ctx.user
	const it = await User.findById(ctx.params.id)
	if (self.admin <= it.admin) {
		throw new Error('permission denied')
	}
	ctx.body = await it.remove()
})

export default router
