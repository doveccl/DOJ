import WebSocket from 'ws'
import { Server } from 'http'
import { routeClient } from './client'
import { routeJudger } from './judger'
import { logSocket } from '../util/log'

export const attachWebSocket = async (server: Server) => {
	const wss = new WebSocket.Server({ server, path: '/socket' })
	wss.on('connection', (ws, req) => {
		ws.on('error', e => logSocket.warn(e.message))
		if (req.url.endsWith('client')) routeClient(ws)
		if (req.url.endsWith('judger')) routeJudger(ws)
	})
}
