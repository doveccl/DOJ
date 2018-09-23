import * as SocketIO from 'socket.io'

import { Status } from '../../common/interface'
import { Submission } from '../model/submission'
import { routeClient } from './client'
import { doJudge, routeJudger } from './judger'

const io = SocketIO()
export const attachSocketIO = async (s: any) => {
	io.attach(s)
	doJudge(await Submission.find({
		'result.status': Status.WAIT
	}))
}

io.of('/judger').on('connection', (socket) => {
	routeJudger(io.of('/judger'), socket)
})

io.of('/client').on('connection', (socket) => {
	routeClient(io.of('/client'), socket)
})
