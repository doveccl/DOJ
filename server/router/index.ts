import * as Compose from 'koa-compose'
import * as Route from 'koa-router'

import Account from './account'
import Config from './config'
import Contest from './contest'
import File from './file'
import Problem from './problem'
import Submission from './submission'
import User from './user'

const router = new Route({ prefix: '/api' })

router.use(
	Account.routes(), Account.allowedMethods(),
	User.routes(), User.allowedMethods(),
	Problem.routes(), Problem.allowedMethods(),
	Contest.routes(), Contest.allowedMethods(),
	Submission.routes(), Submission.allowedMethods(),
	Config.routes(), Config.allowedMethods(),
	File.routes(), File.allowedMethods()
)

export default () => Compose([
	router.routes(),
	router.allowedMethods()
])
