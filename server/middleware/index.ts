import * as Compose from 'koa-compose'
import * as Body from 'koa-body'

import Log from './log'
import Wrap from './wrap'

export default () => Compose([
	Log(), Wrap(),
	Body({ multipart: true })
])
