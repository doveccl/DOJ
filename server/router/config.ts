import Router from '@koa/router'

import config from '../../config'
import { Group } from '../../common/interface'
import { group, token } from '../middleware/auth'
import { Config } from '../model/config'

interface Language {
  name: string
  suffix: string
}

const router = new Router()

router.use('/config', token())

router.get('/config/languages', async (ctx) => {
  const languages: Language[] = config.languages
  ctx.body = languages.map(({ name, suffix }) => ({ name, suffix }))
})

router.get('/config/notification', async (ctx) => {
  ctx.body = await Config.findById('notification')
})

router.put('/config/:id', group(Group.admin), async (ctx) => {
  ctx.body = await Config.findByIdAndUpdate(ctx.params.id, ctx.request.body)
})

export default router
