import * as log4js from 'log4js'
import { get } from 'config'

log4js.configure({
	appenders: {
		out: {
			type: 'stdout',
			layout: {
				type: 'pattern',
				pattern: '%[[%d{yyyy-MM-dd hh:mm:ss.SSS}] [%p] %c -%] %m',
			},
		},
	},
	categories: {
		default: {
			appenders: [ 'out' ],
			level: 'info'
		}
	}
})

export const logHttp = log4js.getLogger('HTTP')
export const logServer = log4js.getLogger('Server')

logHttp.level = get<string>('logLevel')
logServer.level = get<string>('logLevel')

export default (category: string) => log4js.getLogger(category)
