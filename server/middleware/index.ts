import * as compose from 'koa-compose'
import * as body from 'koa-body'
import { Logger } from 'log4js'

import log from './log'

export default (logger: Logger) => compose([
	log(logger),
	body({ multipart: true })
])
