import Koa from 'koa'
import Compose from 'koa-compose'
import mongoose from 'mongoose'

import Log from './middleware/log'
import Static from './middleware/static'
import Wrap from './middleware/wrap'
import Router from './router'

import { config } from './util/config'
import { initDB } from './model/initdb'
import { attachWebSocket } from './socket'
import { startCron } from './util/cron'
import { logServer } from './util/log'
import { connectServer } from './judger'

if (process.argv.includes('--server')) {
	const app = new Koa()

	app.use(Compose([
		Log(),
		Static(),
		Wrap(),
		Router()
	]))

	console.log('config', config.database)
	mongoose.connect(config.database, err => {
		if (err) {
			logServer.error(err.message)
			process.exit(1)
		} else {
			initDB()
			startCron()
			attachWebSocket(app.listen(config.port))
			logServer.info(`doj is running on port ${config.port}`)
		}
	})
}

if (process.argv.includes('--judger')) {
	connectServer()
}
