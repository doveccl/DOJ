import * as config from 'config'
import * as io from 'socket.io-client'

import { judge } from './judge'
import { logJudger } from './log'
import { SE } from './pack'

const secret: string = config.get('secret')
const concurrent: number = config.get('concurrent')

const socket = io(config.get('host'))

socket.on('judge', (s: any) => {
	judge(s)
		.then((val) => socket.emit('update', val))
		.catch((err) => socket.emit('update', SE(err)))
})

socket.on('connect', () => {
	socket.emit('register', { secret, concurrent }, (success: boolean) => {
		if (!success) {
			logJudger.fatal('failed to register')
			process.exit(1)
		}
	})
})
