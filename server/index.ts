import Koa from 'koa'
import compress from 'koa-compress'
import mongoose from 'mongoose'

import Log from './middleware/log'
import Static from './middleware/static'
import Wrap from './middleware/wrap'
import Router from './router'

import config from '../config'
import { initDB } from './model/initdb'
import { attachWebSocket } from './socket'
import { startCron } from './util/cron'
import { logServer } from './util/log'

export async function startServer() {
  startCron()
  await mongoose.connect(config.database)
  initDB()
  const app = new Koa()
  app.use(compress()).use(Log()).use(Static()).use(Wrap()).use(Router.routes())
  attachWebSocket(app.listen(config.port))
  logServer.info(`doj is running on port ${config.port}`)
}
