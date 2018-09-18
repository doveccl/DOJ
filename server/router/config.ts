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

router.put('/config', forGroup('admin'), async (ctx) => {
	const { body } = ctx.request
	for (const i in body) { if (typeof i === 'string') {
		await Config.findByIdAndUpdate(i, { value: body[i] })
	} }
	ctx.body = body
})

export default router
