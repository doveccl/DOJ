import * as Router from 'koa-router'
import { hashSync, compareSync } from 'bcryptjs'

import User from '../model/user'

import { urlFetch } from '../middleware/fetch'
import { ensureGroup, forGroup } from '../middleware/auth'
import { toStringCompare, validatePassword } from '../util/function'

const EXCLUDE_LIST = [ 'solve', 'submit' ]

const router = new Router()

router.get('/user', async ctx => {
	let { rank, page, size } = ctx.query
	if (!rank) { ensureGroup(ctx.self, 'admin') }

	page = parseInt(page) || 1
	size = parseInt(size) || 50
	const total = await User.countDocuments()
	const list = await User.find()
		.select(rank ? '-password -group' : '')
		.sort(rank ? '-solve submit name' : '')
		.skip(size * (page - 1)).limit(size)
	ctx.body = { total, list }
})

router.post('/user', forGroup('admin'), async ctx => {
	const { body } = ctx.request, { group } = body
	if (group) { ensureGroup(ctx.self, group, 1) }
	validatePassword(body.password)

	for (let item of EXCLUDE_LIST) { delete body[item] }
	body.password = hashSync(body.password)
	ctx.body = await User.create(body)
})

router.put('/user/:id', urlFetch('user'), async ctx => {
	const { self, user } = ctx
	const { body } = ctx.request

	if (!toStringCompare(self._id, user._id)) {
		ensureGroup(self, user.group, 1)
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
	user.set(body)
})

router.del('/user/:id', urlFetch('user'), async ctx => {
	ensureGroup(ctx.self, ctx.user.group, 1)
	ctx.body = await ctx.user.remove()
})

export default router
