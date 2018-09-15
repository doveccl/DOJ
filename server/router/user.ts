import * as bcrypt from 'bcryptjs'
import * as Router from 'koa-router'

import { ensureGroup, forGroup, token } from '../middleware/auth'
import { urlFetch } from '../middleware/fetch'
import { User } from '../model/user'
import { toStringCompare, validatePassword } from '../util/function'

const EXCLUDE_LIST = [ 'solve', 'submit' ]

const router = new Router()

router.use('/user', token())

router.get('/user', async (ctx) => {
	const { rank } = ctx.query
	let { page, size } = ctx.query

	if (!rank) { ensureGroup(ctx.self, 'admin') }
	page = parseInt(page, 10) || 1
	size = parseInt(size, 10) || 50

	const total = await User.countDocuments()
	const list = await User.find()
		.select(rank ? '-password -group' : '')
		.sort(rank ? '-solve submit name' : '')
		.skip(size * (page - 1)).limit(size)
	ctx.body = { total, list }
})

router.get('/user/info', async (ctx) => {
	ctx.body = ctx.self.toJSON()
	delete ctx.body.password
})

router.post('/user', forGroup('admin'), async (ctx) => {
	const { body } = ctx.request
	const { group } = body

	if (group) { ensureGroup(ctx.self, group, 1) }
	validatePassword(body.password)

	for (const item of EXCLUDE_LIST) { delete body[item] }
	body.password = bcrypt.hashSync(body.password)
	ctx.body = await User.create(body)
})

router.put('/user/:id', urlFetch('user'), async (ctx) => {
	const { self, user } = ctx
	const { body } = ctx.request

	if (!toStringCompare(self._id, user._id)) {
		ensureGroup(self, user.group, 1)
		delete body.password
	} else { delete body.group }

	if (body.password !== undefined) {
		if (!bcrypt.compareSync(body.oldPassword, self.password)) {
			throw new Error('wrong old password')
		}
		validatePassword(body.password)
		body.password = bcrypt.hashSync(body.password)
	}

	for (const item of EXCLUDE_LIST) { delete body[item] }
	ctx.body = await user.update(body, { runValidators: true })
	user.set(body)
})

router.del('/user/:id', urlFetch('user'), async (ctx) => {
	ensureGroup(ctx.self, ctx.user.group, 1)
	ctx.body = await ctx.user.remove()
})

export default router
