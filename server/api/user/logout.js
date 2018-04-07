module.exports = async ctx => {
	ctx.cookies.set('id', null)
	ctx.cookies.set('token', null)
}
