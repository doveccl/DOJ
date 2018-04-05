module.exports = async ctx => {
	ctx.body = { err: ctx.valid ? 0 : 1 }
}
