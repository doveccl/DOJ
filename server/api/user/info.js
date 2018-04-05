module.exports = async ctx => {
	if (!ctx.valid) {
		return ctx.body = { err: -1 }
	}

	let user = ctx.user
	user.pwd = undefined
	ctx.body = user
}
