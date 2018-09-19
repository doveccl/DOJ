import * as config from 'config'
import * as Router from 'koa-router'

import { forGroup, token } from '../middleware/auth'
import { Config } from '../model/config'

const router = new Router()

router.use('/config', token())

router.get('/config/languages', async (ctx) => {
	ctx.body = config.get('languages')
})

router.get('/config/notification', async (ctx) => {
	ctx.body = await Config.findById('notification')
})

router.put('/config/:id', forGroup('admin'), async (ctx) => {
	ctx.body = await Config.findByIdAndUpdate(ctx.params.id, ctx.request.body)
})

export default router
