import Router from 'koa-router'

import { compare } from '../../common/function'
import { Group } from '../../common/interface'
import { ensureGroup } from '../../common/user'
import { token } from '../middleware/auth'
import { contest, fetch, problem, user } from '../middleware/fetch'
import { Post } from '../model/post'
import { DUser } from '../model/user'

const router = new Router<any, { self: DUser }>()

router.use('/post', token())

router.get('/post', async (ctx) => {
	const topic = ctx.query.topic as string
	const page = Number(ctx.query.page) || 1
	const size = Number(ctx.query.size) || 50

	const total = await Post.countDocuments({ topic })
	const arr = await Post.find({ topic })
		.sort('-_id').skip(size * (page - 1)).limit(size)
	const list: any[] = []
	for (const item of arr) {
		const { name: uname, mail: umail } = await user(item.uid)
		list.push(Object.assign(item.toJSON(), { uname, umail }))
	}
	ctx.body = { total, list }
})

router.put('/post/:id', fetch('post'), async (ctx) => {
	const { self, post } = ctx
	const { content } = ctx.request.body
	if (!compare(self._id, post.uid)) { ensureGroup(self, Group.admin) }
	ctx.body = await post.updateOne({ content }, { runValidators: true })
})

router.post('/post', async (ctx) => {
	const { body } = ctx.request
	body.uid = ctx.self._id
	const p = await problem(body.topic)
	const c = await contest(body.topic)
	if (!p && !c) { throw new Error('invalid topic id') }
	ctx.body = await Post.create(body)
})

router.del('/post/:id', fetch('post'), async (ctx) => {
	const { self, post } = ctx
	if (!compare(self._id, post.uid)) { ensureGroup(self, Group.admin) }
	ctx.body = await post.remove()
})

export default router
