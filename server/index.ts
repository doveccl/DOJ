import * as Koa from 'koa'
import { get } from 'config'
import { connect } from 'mongoose'

import Middleware from './middleware'
import Router from './router'
import File from './model/file'
import { logServer } from './util/log'

const port = get<number>('port')
const dbUri = get<string>('dbUri')

const app = new Koa()

app.use(Middleware())
app.use(Router())

connect(dbUri, {
	useNewUrlParser: true,
	reconnectTries: 0
}, error => {
	if (error) {
		logServer.fatal(error)
		process.exit(1)
	}
	app.listen(port, () => {
		logServer.info(`listening on port ${port}`)
	})
})
