import WebSocket from 'ws'
import { initPath } from './path'
import { config } from '../util/config'
import { judge } from './judge'
import { logJudger } from '../util/log'
import { SE } from '../../common/pack'

export function connectServer() {
	initPath()

	const { host, name, secret, concurrent } = config
	const ws = new WebSocket(`${host}/wss?judger`)
	logJudger.info('connect to server:', ws.url)

	ws.onerror = e => logJudger.warn('connect error:', e.message)
	ws.onclose = () => setTimeout(connectServer, 5000) // reconnect
	ws.onopen = () => {
		logJudger.info('connected, register as judger')
		ws.send(JSON.stringify({ name, secret, concurrent }))
		ws.on('message', raw => {
			const data = JSON.parse(raw.toString())
			judge(data, ws).then(res => {
				ws.send(JSON.stringify(res))
			}).catch(e => {
				logJudger.warn('judge error:', e?.message ?? e)
				ws.send(JSON.stringify(SE(data._id, e)))
			})
		})
	}
}
