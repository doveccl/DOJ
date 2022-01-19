import Koa from 'koa'
import config from 'config'
import Compose from 'koa-compose'
import mongoose from 'mongoose'

import Log from './middleware/log'
import Static from './middleware/static'
import Wrap from './middleware/wrap'
import Router from './router'

import { attachWebSocket } from './socket'
import { startCron } from './util/cron'
import { logServer } from './util/log'

const app = new Koa()
const port = config.get<number>('port')
const database = config.get<string>('database')

app.use(Compose([
	Log(),
	Static(),
	Wrap(),
	Router()
]))

mongoose.connect(database, error => {
	if (error) {
		logServer.fatal(error)
	} else {
		startCron()
		attachWebSocket(app.listen(port, () => {
			logServer.info(`listening on port ${port}`)
		}))
	}
})
