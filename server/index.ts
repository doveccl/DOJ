import * as config from 'config'
import * as Koa from 'koa'
import * as Compose from 'koa-compose'
import * as mongoose from 'mongoose'

import Log from './middleware/log'
import Wrap from './middleware/wrap'
import Router from './router'
import { logServer } from './util/log'

const port: number = config.get('port')
const dbUri: string = config.get('dbUri')

const app = new Koa()

app.use(Compose([ Log(), Wrap(), Router() ]))

/**
 * Fix deprecation warnings
 */
mongoose.set('useNewUrlParser', true)
mongoose.set('useFindAndModify', false)
mongoose.set('useCreateIndex', true)

mongoose.connect(dbUri, (error) => {
	if (error) {
		logServer.fatal(error)
		process.exit(1)
	}
	app.listen(port, () => {
		logServer.info(`listening on port ${port}`)
	})
})
