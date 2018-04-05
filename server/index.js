const Koa = require('koa')
const Body = require('koa-body')
const Router = require('koa-router')

const db = require('./database')
const routes = require('./router')
const util = require('./util')

const app = new Koa()
const router = new Router()
const opt = { multipart: true }

router.use(util('auth'))
for (let i of routes) {
	router.post(i.path, i.func)
}
router.all('*', ctx => {
	ctx.status = 404
})

app.use(Body(opt))
app.use(router.routes())
app.use(router.allowedMethods())

exports.start = async config => {
	let con = config.database
	app.context.db = await db.init(con)
	app.listen(config.port)
}
