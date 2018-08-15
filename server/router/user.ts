import * as Route from 'koa-router'
import user from '../model/user'

const router = new Route()

router.get('/user', async ctx => {
	let users = await user.find()
	ctx.body = users
})

export default router
