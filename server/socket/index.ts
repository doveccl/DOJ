import * as SocketIO from 'socket.io'

import { routeJudger } from './judger'

const io = SocketIO()
export const attachSocketIO = (s: any) => io.attach(s)

io.on('connection', (socket) => {
	routeJudger(io, socket)
})
