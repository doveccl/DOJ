import * as Route from 'koa-router'
import * as Compose from 'koa-compose'

import Account from './account'
import User from './user'
import Problem from './problem'
import Contest from './contest'
import Submission from './submission'
import File from './file'
import { token } from "../middleware/auth"

const router = new Route({ prefix: '/api' })

router.use(
	token(/^\/api\/(?:login|register|reset)/),
	Account.routes(), Account.allowedMethods(),
	User.routes(), User.allowedMethods(),
	Problem.routes(), Problem.allowedMethods(),
	Contest.routes(), Contest.allowedMethods(),
	Submission.routes(), Submission.allowedMethods(),
	File.routes(), File.allowedMethods()
)

export default () => Compose([
	router.routes(),
	router.allowedMethods()
])
