import * as Compose from 'koa-compose'
import * as Body from 'koa-body'

import log from './log'
import wrap from './wrap'

export default () => Compose([
	log(),
	wrap(),
	Body({ multipart: true })
])
