import * as config from 'config'
import * as http from 'http'
import * as Koa from 'koa'
import * as Compose from 'koa-compose'
import * as mongoose from 'mongoose'

import Log from './middleware/log'
import Wrap from './middleware/wrap'
import Router from './router'
import { attachSocketIO } from './socket'
import { logServer } from './util/log'

const port: number = config.get('port')
const database: string = config.get('database')

const app = new Koa()

app.use(Compose([ Log(), Wrap(), Router() ]))

/**
 * Fix deprecation warnings
 */
mongoose.set('useNewUrlParser', true)
mongoose.set('useFindAndModify', false)
mongoose.set('useCreateIndex', true)

mongoose.connect(database, (error) => {
	if (error) {
		logServer.fatal(error)
	} else {
		const server = http.createServer(app.callback())
		attachSocketIO(server)
		server.listen(port, () => {
			logServer.info(`listening on port ${port}`)
		})
	}
})
