import * as Router from 'koa-router'

import Auth from '../middleware/auth'
import File, { TYPE_REG } from '../model/file'

const router = new Router()

router.get('/file', Auth({ type: 'admin' }), async ctx => {
	let { page, size } = ctx.query
	page = parseInt(page) || 1
	size = parseInt(size) || 50
	const total = await File.countDocuments()
	const list = await File.find()
		.skip(size * (page - 1)).limit(size)
	ctx.body = { total, list }
})

router.get('/file/:id', async ctx => {
	const file = await File.findById(ctx.params.id)
	if (file.metadata.type === 'data' && !ctx.user.admin) {
		throw new Error('permission denied')
	}
	ctx.type = file.contentType
	ctx.attachment(file.filename)
	ctx.body = File.creatReadStream(ctx.params.id)
})

router.post('/file', Auth({ type: 'admin' }), async ctx => {
	const keys = Object.keys(ctx.request.files)
	ctx.body = <any[]>[]
	for (let key of keys) {
		const file = ctx.request.files[key]
		if (TYPE_REG.test(file.type)) {
			const { path, name, type } = file
			ctx.body.push(await File.create(path, name, {
				contentType: type, metadata: { type: 'common' }
			}))
		}
	}
})

router.del('/file/:id', Auth({ type: 'admin' }), async ctx => {
	ctx.body = await File.findByIdAndRemove(ctx.params.id)
})

export default router
