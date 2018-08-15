import * as Compose from 'koa-compose'
import * as Body from 'koa-body'

import log from './log'

export default () => Compose([
	log(),
	Body({ multipart: true })
])
