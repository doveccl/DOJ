import * as config from 'config'
import * as IO from 'socket.io'

import { Pack } from '../../common/pack'
import { problem } from '../middleware/fetch'
import { DSubmission } from '../model/submission'
import { logSocket } from '../util/log'
import { update } from './client'

let currentNS: IO.Namespace

const secret: string = config.get('secret')

interface ParsedSubmission {
	_id: any
	language: number,
	code: string
	timeLimit: number
	memoryLimit: number
	data: any
}

const judgers: string[] = []
const submissions: ParsedSubmission[] = []

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
			addJudger(socket.id, data.concurrent)
			callback(true)
		} else {
			callback(false)
		}
	})
	socket.on('finish', (pack: Pack) => {
		if (!pack || !pack._id) { return }
		addJudger(socket.id, 1)
		update(pack)
	})
	socket.on('disconnect', () => {
		if (judgers.includes(socket.id)) {
			logSocket.info('Del judger:', socket.id)
			delJudger(socket.id)
		}
	})
}

const parseSubmission = async (s: DSubmission) => {
	const p = await problem(s.pid)
	const { _id, language, code } = s
	const { timeLimit, memoryLimit, data } = p
	return {
		_id, language, code,
		timeLimit, memoryLimit, data
	} as ParsedSubmission
}
const dispatchSubmission = () => {
	while (judgers.length && submissions.length) {
		const judger = judgers.shift()
		const submission = submissions.shift()
		currentNS.sockets.get(judger).emit('judge', submission)
		logSocket.info(
			'submission', submission._id, '->', judger,
			'-- judgers/submissions queue:',
			`${judgers.length}/${submissions.length}`
		)
	}
}

export const doJudge = async (s: DSubmission) => {
	logSocket.info(
		'add submission to queue:', s._id,
		`-- queue length = ${submissions.length}`
	)
	submissions.push(await parseSubmission(s))
	dispatchSubmission()
}
