import bcrypt from 'bcryptjs'
import Router from 'koa-router'

import { checkPassword, compare } from '../../common/function'
import { Group } from '../../common/interface'
import { ensureGroup } from '../../common/user'
import { group, token } from '../middleware/auth'
import { fetch } from '../middleware/fetch'
import { DUser, User } from '../model/user'
import { sign } from '../util/jwt'
import { send } from '../util/mail'

const EXCLUDE_LIST = [ 'solve', 'submit' ]

const router = new Router<any, { self: DUser }>()

router.use('/user', token())

router.get('/user', async (ctx) => {
	const { rank } = ctx.query
	let { page, size } = ctx.query

	if (!rank) { ensureGroup(ctx.self, Group.admin) }
	page = parseInt(page, 10) || 1
	size = parseInt(size, 10) || 50

	const total = await User.countDocuments()
	const list = await User.find()
		.select(rank ? '-password -group' : '')
		.sort(rank ? '-solve submit' : '')
		.skip(size * (page - 1)).limit(size)
	ctx.body = { total, list }
})

router.get('/user/info', async (ctx) => {
	ctx.body = ctx.self.toJSON()
	delete ctx.body.password
})

router.post('/user', group(Group.admin), async (ctx) => {
	const { body } = ctx.request
	const { group: ugroup } = body

	if (ugroup) { ensureGroup(ctx.self, ugroup, 1) }
	checkPassword(body.password, true)

	for (const item of EXCLUDE_LIST) { delete body[item] }
	body.password = bcrypt.hashSync(body.password)
	ctx.body = await User.create(body)
})

router.put('/user/:id', fetch('user'), async (ctx) => {
	const { self, user } = ctx
	const { body } = ctx.request

	if (!compare(self._id, user._id)) {
		ensureGroup(self, user.group, 1)
		delete body.password
	} else { delete body.group }

	if (body.password) {
		if (!bcrypt.compareSync(body.oldPassword, self.password)) {
			throw new Error('wrong old password')
		}
		checkPassword(body.password, true)
		body.password = bcrypt.hashSync(body.password)
	} else { // 防止为 '' 的情况
		delete body.password
	}

	for (const item of EXCLUDE_LIST) { delete body[item] }
	ctx.body = await user.updateOne(body, { runValidators: true })
	user.set(body)
})

router.del('/user/:id', fetch('user'), async (ctx) => {
	ensureGroup(ctx.self, ctx.user.group, 1)
	ctx.body = await ctx.user.remove()
})

router.post('/user/invite', group(Group.admin), async (ctx) => {
	const { mail } = ctx.request.body
	const invitation = await sign({ mail })
	await send(mail, 'Invitation code', invitation)
	ctx.body = { mail: mail.replace(/^(\w)[^@]*@/, '$1***@') }
})

export default router
