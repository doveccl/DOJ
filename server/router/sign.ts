import * as Route from 'koa-router'
import { hashSync } from 'bcryptjs'
import { get } from 'config'

import Auth from '../middleware/auth'
import User from '../model/user'
import { sign, verify } from '../util/jwt'

const router = new Route()

router.get('/login', Auth({ type: 'password' }), async ctx => {
	ctx.body = {
		name: ctx.user.name,
		mail: ctx.user.mail,
		token: await sign({ id: ctx.user._id })
	}
})

router.post('/register', async ctx => {
	let { name, mail, password, introduction, invitation } = ctx.request.body
	if (!get<boolean>('openRegistration') && !invitation) {
		throw new Error('invitation code is required')
	}
	if (!/^.{6,20}$/.test(password)) {
		throw new Error('invitation password')
	}
	let admin: number = 0
	if (invitation) {
		const data: any = await verify(invitation)
		const { mail: m, admin: a } = data
		if (typeof a === 'number') { admin = a }
		if (m !== mail) { throw new Error('invalid invitation code') }
	}
	password = hashSync(password)
	ctx.user = await User.create({ name, mail, password, introduction, admin })
	ctx.body = {
		name: ctx.user.name,
		mail: ctx.user.mail,
		token: await sign({ id: ctx.user._id })
	}
})

export default router
