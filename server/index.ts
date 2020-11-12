import * as config from 'config'
import * as http from 'http'
import * as Koa from 'koa'
import * as Compose from 'koa-compose'
import * as mongoose from 'mongoose'

import Log from './middleware/log'
import Static from './middleware/static'
import Wrap from './middleware/wrap'
import Router from './router'
import { attachSocketIO } from './socket'
import { startCron } from './util/cron'
import { logServer } from './util/log'

const port: number = config.get('port')
const database: string = config.get('database')

const app = new Koa()

app.use(Compose([ Log(), Static(), Wrap(), Router() ]))

mongoose.connect(database, {
	useCreateIndex: true,
	useNewUrlParser: true,
	useUnifiedTopology: true
}, (error) => {
	if (error) {
		logServer.fatal(error)
	} else {
		startCron()
		const server = http.createServer(app.callback())
		attachSocketIO(server)
		server.listen(port, () => {
			logServer.info(`listening on port ${port}`)
		})
	}
})
