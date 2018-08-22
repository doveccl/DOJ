import * as Router from 'koa-router'

import Auth from '../middleware/auth'
import File, { FS } from '../model/file'

const router = new Router()

router.get('/file', Auth({ type: 'admin' }), async ctx => {
	let { page, size } = ctx.query
	page = parseInt(page) || 1
	size = parseInt(size) || 50
	const total = await FS.countDocuments()
	const list = await FS.find()
		.skip(size * (page - 1)).limit(size)
	ctx.body = { total, list }
})

router.get('/file/:id', async ctx => {
	const file = await File.findById(ctx.params.id)
	if (file.metadata.type === 'data' && !ctx.user.admin) {
		throw new Error('permission denied')
	}
	ctx.type = 'zip'
	ctx.body = File.createReadStream(ctx.params.id)
})

router.post('/file', Auth({ type: 'admin' }), async ctx => {
	console.log(ctx.request.files)
})

router.del('/file/:id', Auth({ type: 'admin' }), async ctx => {
	ctx.body = await File.findByIdAndRemove(ctx.params.id)
})

export default router
