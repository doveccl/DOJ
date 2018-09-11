import { BaseRequest } from 'koa'
import { Files } from 'formidable'

/**
 * fix error TS2459 thrown by ts-node
 * @see https://github.com/dlau/koa-body/issues/109
 */
declare module 'koa' {
	interface Request extends BaseRequest{
		body?: any
		files?: Files
	}
}
