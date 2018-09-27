import * as config from 'config'
import * as log4js from 'log4js'

log4js.configure({
	appenders: {
		out: {
			type: 'stdout',
			layout: {
				type: 'pattern',
				pattern: '%[[%d{yyyy-MM-dd hh:mm:ss.SSS}] [%p] %c -%] %m'
			}
		}
	},
	categories: {
		default: {
			appenders: [ 'out' ],
			level: 'info'
		}
	}
})

export const logJudger = log4js.getLogger('Judger')

logJudger.level = config.get('log')
