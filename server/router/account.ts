import * as bcrypt from 'bcryptjs'
import * as config from 'config'
import * as Router from 'koa-router'

import { checkPassword } from '../../common/function'
import { password } from '../middleware/auth'
import { User } from '../model/user'
import { sign, verify } from '../util/jwt'
import { send } from '../util/mail'
import { has, put } from '../util/timeset'

const router = new Router()

router.get('/login', password(), async (ctx) => {
	ctx.body = ctx.self.toJSON()
	ctx.body.token = await sign({ id: ctx.self._id })
	delete ctx.body.password
})

router.post('/register', async (ctx) => {
	const pass = ctx.request.body.password
	const { name, mail, invitation } = ctx.request.body
	if (!config.get('registration') && !invitation) {
		throw new Error('invitation code is required')
	}

	checkPassword(pass, true)
	if (invitation) {
		try {
			const data: any = await verify(invitation)
			if (data.mail !== mail) { throw new Error() }
		} catch (e) {
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
		name, mail, password: bcrypt.hashSync(pass)
	})
	ctx.body = { name, mail, token: await sign({ id }) }
})

router.get('/reset', async (ctx) => {
	const user = ctx.query.user
	const condition = [{ name: user }, { mail: user }]
	const u = await User.findOne({ $or: condition })

	if (!u) { throw new Error('invalid user name or mail') }
	if (has(u.mail)) { throw new Error('please try later') } else { put(u.mail) }

	const hash = bcrypt.hashSync(u.password)
	const code = await sign({ id: u._id, hash }, undefined, { expiresIn: '1h' })

	await send(u.mail, 'Verify code', code)
	ctx.body = { mail: u.mail.replace(/^(\w)[^@]*@/, '$1***@') }
})

router.put('/reset', async (ctx) => {
	const { code, password: pass } = ctx.request.body

	const data: any = await verify(code)
	const user = await User.findById(data.id)

	checkPassword(pass, true)
	if (!bcrypt.compareSync(user.password, data.hash)) {
		throw new Error('invalid verify code')
	}
	if (bcrypt.compareSync(pass, user.password)) {
		throw new Error('new password should be different from the old one')
	}
	ctx.body = await user.update({ password: bcrypt.hashSync(pass) })
})

export default router
