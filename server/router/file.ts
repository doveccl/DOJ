import * as config from 'config'
import * as Router from 'koa-router'

import { Group } from '../../common/interface'
import { ensureGroup } from '../../common/user'
import { group, token } from '../middleware/auth'
import { fetch } from '../middleware/fetch'
import { File, TYPE_REG } from '../model/file'

const router = new Router()

router.use('/file', token(true))

router.get('/file', group(Group.admin), async (ctx) => {
	let { page, size } = ctx.query
	page = parseInt(page, 10) || 1
	size = parseInt(size, 10) || 50
	const total = await File.countDocuments()
	const list = await File.find()
		.sort('-_id')
		.skip(size * (page - 1)).limit(size)
	ctx.body = { total, list }
})

router.get('/file/:id', fetch('file'), async (ctx) => {
	if (ctx.file.metadata === 'data') {
		ensureGroup(ctx.self, Group.admin)
	}

	ctx.type = ctx.file.contentType
	ctx.body = File.creatReadStream(ctx.params.id)
})

router.post('/file', group(Group.admin), async (ctx) => {
	const keys = Object.keys(ctx.request.files)
	ctx.body = [] as any[]
	for (const key of keys) {
		const file = ctx.request.files[key]
		if (TYPE_REG.test(file.type)) {
			const { path, name, type } = file
			ctx.body.push(await File.create(path, name, {
				contentType: type, metadata: { type: 'common' }
			}))
		}
	}
})

router.put('/file/:id', group(Group.admin), fetch('file'), async (ctx) => {
	const { filename } = ctx.request.body
	if (!filename) { throw new Error('Invalid filename') }
	ctx.body = await ctx.file.update({ filename })
})

router.del('/file/:id', group(Group.admin), fetch('file'), async (ctx) => {
	ctx.body = await File.findByIdAndRemove(ctx.params.id)
})

/**
 * for judger to download data
 */
router.get('/data/:id', fetch('file'), async (ctx) => {
	if (ctx.query.secret !== config.get('secret')) {
		throw new Error('invalid secret')
	}

	ctx.type = ctx.file.contentType
	ctx.body = File.creatReadStream(ctx.params.id)
})

export default router
