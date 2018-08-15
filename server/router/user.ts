import * as Route from 'koa-router'
import User from '../model/user'

const router = new Route()

router.get('/user', async ctx => {
	ctx.body = await User.find(ctx.query)
})
router.post('/user', async ctx => {
	const body = ctx.request.body
	const user = new User(body)
	ctx.body = await user.save()
})
router.get('/user/:id', async ctx => {
	const id = ctx.params.id
	ctx.body = await User.findById(id)
})
router.put('/user/:id', async ctx => {
	const id = ctx.params.id
	const body = ctx.request.body
	ctx.body = await User.findByIdAndUpdate(id, body, {
		new: true,
		runValidators: true
	}).exec()
})
router.del('/user/:id', async ctx => {
	const id = ctx.params.id
	ctx.body = await User.findByIdAndRemove(id)
})

export default router
