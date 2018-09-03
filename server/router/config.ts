import * as Router from 'koa-router'
import { get } from 'config'

import Config from '../model/config'
import { token, forGroup } from '../middleware/auth'

const router = new Router()

router.use('/config', token())

router.get('/config', async ctx => {
	ctx.body = await Config.find()
})

router.get('/config/languages', async ctx => {
	ctx.body = get('languages')
})

router.put('/config/:id', forGroup('admin'), async ctx => {
	const { id } = ctx.params, { value } = ctx.request.body
	ctx.body = await Config.findByIdAndUpdate(id, { value })
})

export default router
