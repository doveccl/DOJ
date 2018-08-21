import * as Router from 'koa-router'
import { hashSync, compareSync } from 'bcryptjs'
import { get } from 'config'

import Auth from '../middleware/auth'
import User from '../model/user'
import { sign, verify } from '../util/jwt'
import { has, put } from '../util/timeset'
import { send } from '../util/mail'

const router = new Router()

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
	if (!password || !/^.{6,20}$/.test(password)) {
		throw new Error('invalid password (length 6-20)')
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

router.get('/reset', async ctx => {
	const user: string = ctx.query.user
	const u = await User.findOne({ $or: [{ name: user }, { mail: user }] })
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
	ctx.user = await User.findById(data.id)
	if (!compareSync(ctx.user.password, data.hash)) {
		throw new Error('invalid verify code')
	}
	if (compareSync(password, ctx.user.password)) {
		throw new Error('new password should be different from the old one')
	}
	if (!/^.{6,20}$/.test(password)) {
		throw new Error('invalid password (length 6-20)')
	}
	ctx.body = await ctx.user.update({ password: hashSync(password) })
})

export default router
