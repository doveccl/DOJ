import { Socket, Namespace } from 'socket.io'

import { compare } from '../../common/function'
import { ContestType, Group, Status } from '../../common/interface'
import { Pack } from '../../common/pack'
import { diffGroup } from '../../common/user'
import { contest, user } from '../middleware/fetch'
import { Submission } from '../model/submission'
import { verify } from '../util/jwt'
import { logSocket } from '../util/log'

let currentNS: Namespace

export const update = async (pack: Pack) => {
	await Submission.findByIdAndUpdate(pack._id, pack)
	currentNS && currentNS.to(pack._id).emit('result', pack)
}

export const step = async (pack: Pack) => {
	currentNS && currentNS.to(pack._id).emit('step', pack)
}

const verifyRegister = async (id: string, token: string) => {
	const data: any = await verify(token)
	const s = await Submission.findById(id)
	const u = await user(data.id)
	if (s.cid) {
		const c = await contest(s.cid)
		if (!diffGroup(u, Group.admin) && (
			(c.type === ContestType.OI && new Date() < c.endAt) || (
				c.type === ContestType.ICPC &&
				new Date() < c.freezeAt &&
				!compare(u._id, s.uid)
			)
		)) {
			throw new Error('result frozen')
		}
	}
	logSocket.info('User query:', u._id, s._id)
	if (s.result.status !== Status.WAIT) return s
}

export const routeClient = (io: Namespace, socket: Socket) => {
	currentNS = io
	socket.on('register', (data) => {
		if (!data) { return }
		const { id, token } = data
		verifyRegister(id, token)
			.then(s => {
				socket.join(id)
				s && currentNS.to(id).emit('result', s)
			})
			.catch(e => logSocket.error(e))
	})
}
