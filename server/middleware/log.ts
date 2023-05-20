import { Middleware } from 'koa'
import { logHttp } from '../util/log'

export default (): Middleware =>
  async (ctx, next) => {
    const stime = Date.now()
    await next()
    const cost = Date.now() - stime
    logHttp.info(ctx.method, ctx.status, ctx.url, `${cost}ms`)
    logHttp.debug('request headers:', ctx.request.headers)
    logHttp.debug('request body:', ctx.request.body)
    logHttp.debug('response:', ctx.body)
  }
