import * as Koa from 'koa'
import * as Compose from 'koa-compose'

import { get } from 'config'
import { connect, set } from 'mongoose'

import Log from './middleware/log'
import Wrap from './middleware/wrap'
import Router from './router'

import { logServer } from './util/log'

const port = get<number>('port')
const dbUri = get<string>('dbUri')

const app = new Koa()

app.use(Compose([ Log(), Wrap(), Router() ]))

/**
 * Fix deprecation warnings
 */
set('useNewUrlParser', true)
set('useFindAndModify', false)
set('useCreateIndex', true)

connect(dbUri, error => {
	if (error) {
		logServer.fatal(error)
		process.exit(1)
	}
	app.listen(port, () => {
		logServer.info(`listening on port ${port}`)
	})
})
