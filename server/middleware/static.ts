import { Middleware } from 'koa'
import { join, extname } from 'path'
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
            /(js|css)$/.test(file) && ctx.set('Cache-Control', 'max-age=31536000')
            ctx.body = createReadStream(file)
            ctx.type = extname(file)
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
