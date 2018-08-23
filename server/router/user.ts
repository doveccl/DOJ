import * as Router from 'koa-router'
import { hashSync, compareSync } from 'bcryptjs'

import User, { UserGroup as G } from '../model/user'

import fetch from '../middleware/fetch'
import { token, group, guard, password } from '../middleware/auth'
import { toStringCompare, validatePassword } from '../util/function'

const EXCLUDE_LIST = ['solve', 'submit', 'createdAt', 'updatedAt']

const router = new Router()

router.use(token(), fetch({ type: 'user' }))

router.get('/user', async ctx => {
	let { rank, page, size } = ctx.query
	if (!rank) { guard(ctx.self, G.admin) }

	page = parseInt(page) || 1
	size = parseInt(size) || 50
	const total = await User.countDocuments()
	const list = await User.find()
		.select(rank ? '-password -admin' : '')
		.sort(rank ? '-solve submit name' : '')
		.skip(size * (page - 1)).limit(size)
	ctx.body = { total, list }
})

router.post('/user', group(G.admin), async ctx => {
	const body = ctx.request.body
	guard(ctx.self, body.group, 1)
	validatePassword(body.password)

	for (let item of EXCLUDE_LIST) { delete body[item] }
	body.password = hashSync(body.password)
	ctx.body = await User.create(body)
})

router.put('/user/:id', async ctx => {
	const { self, user } = ctx
	const { body } = ctx.request

	if (!toStringCompare(self._id, user._id)) {
		guard(self, user.group, 1)
		delete body.password
	} else { delete body.group }

	if (body.password !== undefined) {
		validatePassword(body.oldPassword)
		if (!compareSync(body.oldPassword, self.password)) {
			throw new Error('invalid old password')
		}
		validatePassword(body.password)
		body.password = hashSync(body.password)
	}

	for (let item of EXCLUDE_LIST) { delete body[item] }
	ctx.body = await user.update(body, { runValidators: true })
})

router.del('/user/:id', group(G.admin), async ctx => {
	const { self, user } = ctx
	guard(self, user.group, 1)
	ctx.body = await user.remove()
})

export default router
