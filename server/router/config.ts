import * as config from 'config'
import * as Router from 'koa-router'

import { forGroup, token } from '../middleware/auth'
import { Config } from '../model/config'

const router = new Router()

router.use('/config', token())

router.get('/config', async (ctx) => {
	ctx.body = await Config.find()
})

router.get('/config/languages', async (ctx) => {
	ctx.body = config.get('languages')
})

router.put('/config/:id', forGroup('admin'), async (ctx) => {
	const { id } = ctx.params
	const { value } = ctx.request.body
	ctx.body = await Config.findByIdAndUpdate(id, { value })
})

export default router
