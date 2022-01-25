import { config } from './config'

const logLevel = {
	debug: 0,
	info: 1,
	warn: 2,
	error: 3
}[config.log] ?? 1

function getDateStr() {
  const date = new Date()
  const Y = date.getFullYear()
  const M = date.getMonth() + 1
  const D = date.getDate()
  const h = date.getHours()
  const m = date.getMinutes()
  const s = date.getSeconds()
  return `${Y}.${M}.${D} ${h}:${m}:${s}`
}

const getLogger = (type: string) => ({
	debug(...args: unknown[]) {
		logLevel <= 0 && console.debug(`[${getDateStr()}] [Debug] [${type}]`, ...args)
	},
	info(...args: unknown[]) {
		logLevel <= 1 && console.info(`[${getDateStr()}] [Info] [${type}]`, ...args)
	},
	warn(...args: unknown[]) {
		logLevel <= 2 && console.warn(`[${getDateStr()}] [Warn] [${type}]`, ...args)
	},
	error(...args: unknown[]) {
		logLevel <= 3 && console.error(`[${getDateStr()}] [Error] [${type}]`, ...args)
	}
})

export const logHttp = getLogger('Http')
export const logSocket = getLogger('Socket')
export const logServer = getLogger('Server')
export const logJudger = getLogger('Judger')
