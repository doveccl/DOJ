import * as Route from 'koa-router'
import * as Compose from 'koa-compose'

import user from './user'

const router = new Route({ prefix: '/api' })

router.use(user.routes(), user.allowedMethods())

export default () => Compose([
	router.routes(),
	router.allowedMethods()
])
