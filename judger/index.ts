import config from 'config'
import { Manager } from 'socket.io-client'

import { SE } from '../common/pack'
import { judge } from './judge'
import { logJudger } from './log'

const host: string = config.get('host')
const secret: string = config.get('secret')
const concurrent: number = config.get('concurrent')

const manager = new Manager(host)
const socket = manager.socket('/judger')
logJudger.info('connect to server:', host)

socket.on('judge', (s: any) => {
	judge(s, socket)
		.then((val) => socket.emit('finish', val))
		.catch((err) => {
			logJudger.error('judge error:', err)
			socket.emit('finish', SE(s._id, err))
		})
})

socket.on('connect', () => {
	logJudger.info('connected, register as judger')
	socket.emit('register', { secret, concurrent }, (success: boolean) => {
		if (!success) {
			logJudger.fatal('failed to register')
			process.exit(1)
		}
	})
})
