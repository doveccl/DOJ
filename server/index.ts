import * as Koa from 'koa'
import { get } from 'config'
import { connect } from 'mongoose'

import middleware from './middleware'
import router from './router'
import { logServer } from './util/log'

const port = get<number>('port')
const dbUri = get<string>('dbUri')

const app = new Koa()

app.use(middleware())
app.use(router())

connect(dbUri, { useNewUrlParser: true }, err => {
	if (err) { throw err }
	app.listen(port, () => logServer.info(`listening on port ${port}`))
})
