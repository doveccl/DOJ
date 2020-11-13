import Compose from 'koa-compose'
import Route from 'koa-router'

import Account from './account'
import Config from './config'
import Contest from './contest'
import File from './file'
import Post from './post'
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
	Post.routes(), Post.allowedMethods(),
	Config.routes(), Config.allowedMethods(),
	File.routes(), File.allowedMethods()
)

export default () => Compose([
	router.routes(),
	router.allowedMethods()
])
