import * as SocketIO from 'socket.io'

import { Status } from '../../common/interface'
import { Submission } from '../model/submission'
import { routeClient } from './client'
import { doJudge, routeJudger } from './judger'

export const judgeFromDB = async () => {
	(await Submission.find({
		'result.status': Status.WAIT
	})).forEach(doJudge)
}

const io = SocketIO()
export const attachSocketIO = async (s: any) => {
	io.attach(s)
	await judgeFromDB()
}

io.of('/judger').on('connection', (socket) => {
	routeJudger(io.of('/judger'), socket)
})

io.of('/client').on('connection', (socket) => {
	routeClient(io.of('/client'), socket)
})
