import * as Route from 'koa-router'
import * as Compose from 'koa-compose'

import Auth from '../middleware/auth'
import Sign from './sign'
import User from './user'

const router = new Route({ prefix: '/api' })

router.use(
	Auth({
		type: 'token',
		exclude: /^\/api\/(login|register)$/
	}),
	Sign.routes(), Sign.allowedMethods(),
	User.routes(), User.allowedMethods(),
)

export default () => Compose([
	router.routes(),
	router.allowedMethods()
])
