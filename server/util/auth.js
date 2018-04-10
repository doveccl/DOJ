const { createHash } = require('crypto')

const md5 = str => createHash('md5').update(str).digest('hex')
const sha1 = str => createHash('sha1').update(str).digest('hex')

module.exports = async (ctx, next) => {
	ctx.breq = ctx.request.body || {}
	ctx.freq = ctx.request.field || {}

	let token = ctx.cookies.get('token')
	let ua = ctx.headers['user-agent']
	let usr = ctx.breq.user
	let pwd = ctx.breq.password
	let cok = {
		maxAge: ctx.breq.remember ?
			365 * 24 * 3600000 : -1
	}

	ctx.user = ctx.valid = false

	if (usr && pwd) { // verify password
		let cur = ctx.db('usr').find({
			$or: [{ id: usr }, { mail: usr }]
		})
		if (await cur.hasNext()) {
			let u = await cur.next()
			let v = md5(`${ua}-${u.pwd}`)
			let t = `${u.id}-${v}`
			let h = sha1(pwd)
			if (u.pwd === h) {
				ctx.user = u
				ctx.valid = true
				ctx.cookies.set('token', t, cok)
			}
		}
	} else 	if (token) { // verify token
		let [i, v] = token.split('-')
		let cur = ctx.db('usr').find({ id: i })
		if (await cur.hasNext()) {
			let u = await cur.next()
			let h = md5(`${ua}-${u.pwd}`)
			if (v === h) {
				ctx.user = u
				ctx.valid = true
			}
		}
	}

	await next()
}
