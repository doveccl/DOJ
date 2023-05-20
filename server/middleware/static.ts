import { join } from 'path'
import { getType } from 'mime'
import { Middleware } from 'koa'
import { createReadStream, statSync } from 'fs'

export default (): Middleware => async (ctx, next) => {
  if (ctx.url.startsWith('/api')) {
    await next()
  } else {
    [
      ['dist', ctx.url],
      ['dist/index.html'],
    ]
      .map(paths => join(...paths))
      .some(file => {
        try {
          if (statSync(file).isFile()) {
            ctx.type = getType(file) ?? ''
            ctx.body = createReadStream(file)
            return true
          } else {
            return false
          }
        } catch {
          return false
        }
      })
  }
}
