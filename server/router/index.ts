import * as Route from 'koa-router'
import * as Compose from 'koa-compose'

import user from './user'
import auth from '../middleware/auth'

const router = new Route({ prefix: '/api' })

router.use(auth({ exclude: /^\/api\/(login|register)$/ }))
router.use(user.routes(), user.allowedMethods())

export default () => Compose([
	router.routes(),
	router.allowedMethods()
])
