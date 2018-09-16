import * as Router from 'koa-router'

import { ensureGroup, token } from '../middleware/auth'
import { contest, problem, urlFetch, user } from '../middleware/fetch'
import { Post } from '../model/post'
import { toStringCompare } from '../util/function'

const router = new Router()

router.use('/post', token())

router.get('/post', async (ctx) => {
	const { topic } = ctx.query
	let { page, size } = ctx.query

	page = parseInt(page, 10) || 1
	size = parseInt(size, 10) || 50
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

router.put('/post/:id', urlFetch('post'), async (ctx) => {
	const { self, post } = ctx
	const { content } = ctx.request.body
	if (!toStringCompare(self._id, post.uid)) { ensureGroup(self, 'admin') }
	ctx.body = await post.update({ content }, { runValidators: true })
})

router.post('/post', async (ctx) => {
	const { body } = ctx.request
	body.uid = ctx.self._id
	const p = await problem(body.topic)
	const c = await contest(body.topic)
	if (!p && !c) { throw new Error('invalid topic id') }
	ctx.body = await Post.create(body)
})

router.del('/post/:id', urlFetch('post'), async (ctx) => {
	const { self, post } = ctx
	if (!toStringCompare(self._id, post.uid)) { ensureGroup(self, 'admin') }
	ctx.body = await post.remove()
})

export default router
