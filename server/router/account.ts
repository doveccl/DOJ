import * as Router from 'koa-router'
import { hashSync, compareSync } from 'bcryptjs'
import { get } from 'config'

import User, { UserGroup } from '../model/user'
import { password } from '../middleware/auth'
import { validatePassword } from '../util/function'
import { sign, verify } from '../util/jwt'
import { has, put } from '../util/timeset'
import { send } from '../util/mail'

const router = new Router()

router.get('/login', password(), async ctx => {
	ctx.body = ctx.self.toJSON()
	ctx.body.token = await sign({ id: ctx.self._id })
	delete ctx.body.password
})

router.post('/register', async ctx => {
	const { name, mail, password, invitation } = ctx.request.body
	if (!get<boolean>('openRegistration') && !invitation) {
		throw new Error('invitation code is required')
	}

	validatePassword(password)
	let group = UserGroup.common
	if (invitation) {
		try {
			const data: any = await verify(invitation)
			const { mail: m, group: g } = data
			if (typeof g === 'number') { group = g }
			if (m !== mail) { throw new Error() }
		} catch(e) {
			throw new Error('invalid invitation code')
		}
	}

	if (await User.findOne({ name })) {
		throw new Error('invalid username: already exists')
	}
	if (await User.findOne({ mail })) {
		throw new Error('invalid mail: already exists')
	}

	const { _id: id } = await User.create({
		name, mail, group, password: hashSync(password)
	})
	ctx.body = { name, mail, token: await sign({ id }) }
})

router.get('/reset', async ctx => {
	const user: string = ctx.query.user, name = user, mail = user
	const u = await User.findOne({ $or: [{ name }, { mail }] })
	if (!u) { throw new Error('invalid user name or mail') }
	if (has(u.mail)) { throw new Error('please try later') } else { put(u.mail) }

	const id = u._id, hash = hashSync(u.password)
	const code = await sign({ id, hash }, undefined, { expiresIn: '1h' })
	await send(u.mail, 'Verify code for password reset', code)
	ctx.body = { mail: u.mail.replace(/^(\w)[^@]*@/, '$1***@') }
})

router.put('/reset', async ctx => {
	let { code, password } = ctx.request.body
	const data: any = await verify(code)
	const user = await User.findById(data.id)

	validatePassword(password)
	if (!compareSync(user.password, data.hash)) {
		throw new Error('invalid verify code')
	}
	if (compareSync(password, user.password)) {
		throw new Error('new password should be different from the old one')
	}
	ctx.body = await user.update({ password: hashSync(password) })
})

export default router
