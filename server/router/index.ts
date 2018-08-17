import * as Route from 'koa-router'
import * as Compose from 'koa-compose'

import auth from '../middleware/auth'
import sign from './sign'
import user from './user'

const router = new Route({ prefix: '/api' })

router.use(
	auth({
		type: 'token',
		exclude: /^\/api\/(login|register)$/
	}),
	sign.routes(), sign.allowedMethods(),
	user.routes(), user.allowedMethods(),
)

export default () => Compose([
	router.routes(),
	router.allowedMethods()
])
