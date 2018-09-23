import * as SocketIO from 'socket.io'

import { routeJudger } from './judger'
import { routeSocket } from './status'

const io = SocketIO()
export const attachSocketIO = (s: any) => io.attach(s)

io.of('/judger').on('connection', (socket) => {
	routeJudger(io.of('/judger'), socket)
})

io.of('/client').on('connection', (socket) => {
	routeSocket(io.of('/client'), socket)
})
