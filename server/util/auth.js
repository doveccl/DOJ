const { ObjectID } = require('mongodb')
const { createHash } = require('crypto')

const md5 = str => createHash('md5').update(str).digest('hex')
const sha1 = str => createHash('sha1').update(str).digest('hex')

module.exports = async (ctx, next) => {
	ctx.breq = ctx.request.body || {}
	ctx.freq = ctx.request.field || {}

	let id = ctx.cookies.get('id')
	let token = ctx.cookies.get('token')
	let ua = ctx.headers['user-agent']
	let usr = ctx.request.body.user
	let pwd = ctx.request.body.password

	ctx.user = ctx.valid = false

	if (usr && pwd) { // verify password
		let cur = ctx.db.usr.find({ uid: usr })
		if (await cur.hasNext()) {
			let u = await cur.next()
			let t = md5(`${ua}-${u.pwd}`)
			let h = sha1(pwd)
			if (u.pwd === h) {
				ctx.user = u
				ctx.valid = true
				ctx.cookies.set('id', u._id)
				ctx.cookies.set('token', t)
			}
		}
	} else 	if (id && token) { // verify token
		let cur = ctx.db.usr.find({ _id: ObjectID(id) })
		if (await cur.hasNext()) {
			let u = await cur.next()
			let h = md5(`${ua}-${u.pwd}`)
			if (token === h) {
				ctx.user = u
				ctx.valid = true
			}
		}
	}

	await next()
}
