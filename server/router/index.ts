import * as Route from 'koa-router'
import * as Compose from 'koa-compose'

import Account from './account'
import User from './user'
import Problem from './problem'
import Submission from './submission'
import File from './file'

const router = new Route({ prefix: '/api' })

router.use(
	Account.routes(), Account.allowedMethods(),
	User.routes(), User.allowedMethods(),
	Problem.routes(), Problem.allowedMethods(),
	Submission.routes(), Submission.allowedMethods(),
	File.routes(), File.allowedMethods()
)

export default () => Compose([
	router.routes(),
	router.allowedMethods()
])
