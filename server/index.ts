import * as Koa from 'koa'
import * as log4js from 'log4js'

import { get } from 'config'
import { connect } from 'mongoose'

import middleware from './middleware'
import router from './router'

const app = new Koa()
const logger = log4js.getLogger('server')

const port = get<number>('port')
const dbUri = get<string>('dbUri')
const logLevel = get<string>('logLevel')

logger.level = logLevel
app.use(middleware(logger))
app.use(router())

connect(dbUri, { useNewUrlParser: true }, err => {
	if (err) { throw err }
	app.listen(port, () => {
		logger.info(
			'Server is running on:',
			`http://127.0.0.1:${port}`
		)
	})
})
