import config from 'config'
import WebSocket from 'ws'
import { SE } from '../common/pack'
import { judge } from './judge'
import { logJudger } from './log'

const host = config.get<string>('host')
const name = config.get<string>('name')
const secret = config.get<string>('secret')
const concurrent = config.get<number>('concurrent')

function connectServer() {
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

connectServer()
