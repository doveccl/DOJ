import { createReadStream, stat } from 'fs-extra'
import { Middleware } from 'koa'
import { getType } from 'mime'
import { join } from 'path'

const fileExists = async (path: string) => {
	try {
		const s = await stat(path)
		return s.isFile()
	} catch (e) {
		return false
	}
}

export default (): Middleware =>
	async (ctx, next) => {
		if (!/:?^\/(api|socket\.io)/.test(ctx.url)) {
			const p = join('dist', ctx.url)
			const f = join('dist', ctx.url, 'index.html')
			if (await fileExists(p)) {
				ctx.type = getType(p)
				ctx.body = createReadStream(p)
			} else if (await fileExists(f)) {
				ctx.type = getType(f)
				ctx.body = createReadStream(f)
			} else if (await fileExists('dist/index.html')) {
				ctx.type = getType('index.html')
				ctx.body = createReadStream('dist/index.html')
			}
		} else {
			await next()
		}
	}
