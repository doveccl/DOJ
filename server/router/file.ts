import config from 'config'
import Router from 'koa-router'

import { Group } from '../../common/interface'
import { ensureGroup } from '../../common/user'
import { group, token } from '../middleware/auth'
import { fetch } from '../middleware/fetch'
import { DFile, File, TYPE_REG } from '../model/file'

const router = new Router<any, { file: DFile }>()

router.use('/file', token(true))

router.get('/file', group(Group.admin), async (ctx) => {
	const page = Number(ctx.query.page) || 1
	const size = Number(ctx.query.size) || 50

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
	ctx.body = new Array()
	for (const key of keys) {
		const item = ctx.request.files[key]
		const files = Array.isArray(item) ? item : [item]
		for (const file of files) {
			if (TYPE_REG.test(file.type)) {
				const { path, name, type } = file
				ctx.body.push(await File.create(path, name, {
					contentType: type, metadata: { type: 'common' }
				}))
			}
		}
	}
})

router.put('/file/:id', group(Group.admin), fetch('file'), async (ctx) => {
	const { filename } = ctx.request.body
	if (!filename) { throw new Error('Invalid filename') }
	ctx.body = await ctx.file.updateOne({ filename })
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
