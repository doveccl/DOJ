import * as config from 'config'
import * as IO from 'socket.io'

import { compare } from '../../common/function'
import { Pack, SE } from '../../common/pack'
import { problem } from '../middleware/fetch'
import { DSubmission } from '../model/submission'
import { logSocket } from '../util/log'
import { update } from './client'

let currentNS: IO.Namespace

const secret: string = config.get('secret')

const judgers: string[] = []
const judgings: { [index: string]: any[] } = {}

const addJudger = (id: string, concurrent: number) => {
	let count = concurrent
	while (count--) { judgers.push(id) }
	dispatchSubmission()
}
const delJudger = (id: string) => {
	for (let i = judgers.length - 1; i >= 0; i--) {
		if (judgers[i] === id) {
			judgers.splice(i, 1)
		}
	}
}

export const routeJudger = (io: IO.Namespace, socket: IO.Socket) => {
	currentNS = io
	socket.on('register', (data, callback) => {
		if (data && data.secret === secret) {
			logSocket.info('Add judger:', socket.id)
			judgings[socket.id] = []
			addJudger(socket.id, data.concurrent)
			callback(true)
		} else {
			callback(false)
		}
	})
	socket.on('finish', (pack: Pack) => {
		if (!pack || !pack._id) { return }
		addJudger(socket.id, 1)
		for (let i = judgings[socket.id].length - 1; i >= 0; i--) {
			if (compare(judgings[socket.id][i]._id, pack._id)) {
				logSocket.debug('Update submission:', pack)
				judgings[socket.id].splice(i, 1)
			}
		}
		update(pack)
	})
	socket.on('disconnect', () => {
		if (judgers.includes(socket.id)) {
			logSocket.info('Del judger:', socket.id)
			for (const submission of judgings[socket.id]) {
				logSocket.debug('Update judging queue:', submission._id)
				update(SE(submission._id, 'judger disconnected'))
			}
			delete judgings[socket.id]
			delJudger(socket.id)
		}
	})
}

const submissions: any[] = []

const parseSubmission = async (s: DSubmission) => {
	const p = await problem(s.pid)
	const { _id, language, code } = s
	const { timeLimit, memoryLimit, data } = p
	return { _id, language, code, timeLimit, memoryLimit, data }
}
const dispatchSubmission = () => {
	while (judgers.length && submissions.length) {
		const judger = judgers.shift()
		const submission = submissions.shift()
		judgings[judger].push(submission)
		currentNS.sockets[judger].emit('judge', submission)
	}
}

export const doJudge = async (ss: DSubmission[]) => {
	for (const s of ss) {
		submissions.push(await parseSubmission(s))
	}
	dispatchSubmission()
}
