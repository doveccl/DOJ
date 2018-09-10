import * as Router from 'koa-router'

import File, { TYPE_REG } from '../model/file'

import { urlFetch } from '../middleware/fetch'
import { token, ensureGroup, forGroup } from '../middleware/auth'

const router = new Router()

router.use('/file', token(true))

router.get('/file', forGroup('admin'), async ctx => {
	let { page, size } = ctx.query
	page = parseInt(page) || 1
	size = parseInt(size) || 50
	const total = await File.countDocuments()
	const list = await File.find()
		.skip(size * (page - 1)).limit(size)
	ctx.body = { total, list }
})

router.get('/file/:id', urlFetch('file'), async ctx => {
	if (ctx.file.metadata === 'data') {
		ensureGroup(ctx.self, 'admin')
	}

	ctx.type = ctx.file.contentType
	ctx.attachment(ctx.file.filename)
	ctx.body = File.creatReadStream(ctx.params.id)
})

router.post('/file', forGroup('admin'), async ctx => {
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

router.put('/file/:id', forGroup('admin'), urlFetch('file'), async ctx => {
	const { filename } = ctx.request.body
	ctx.body = await ctx.file.update({ filename })
})

router.del('/file/:id', forGroup('admin'), urlFetch('file'), async ctx => {
	ctx.body = await File.findByIdAndRemove(ctx.params.id)
})

export default router