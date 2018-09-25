import * as config from 'config'
import * as fs from 'fs-extra'
import * as io from 'socket.io-client'

import { SE } from '../common/pack'
import { judge } from './judge'
import { logJudger } from './log'
import { spawnSync } from './run'

const host: string = config.get('host')
const secret: string = config.get('secret')
const concurrent: number = config.get('concurrent')

const socket = io(`${host}/judger`)

if (!fs.pathExistsSync('/doj_tmp')) {
	fs.mkdirpSync('/doj_tmp')
}
if (!fs.pathExistsSync('/run/lrun/mirrorfs/doj')) {
	spawnSync('lrun-mirrorfs', [
		'--name', 'doj',
		'--setup', 'mirrorfs.cfg'
	])
}

socket.on('judge', (s: any) => {
	judge(s)
		.then((val) => socket.emit('finish', val))
		.catch((err) => {
			logJudger.error('judge error:', err)
			socket.emit('finish', SE(s._id, err))
		})
})

socket.on('connect', () => {
	socket.emit('register', { secret, concurrent }, (success: boolean) => {
		if (!success) {
			logJudger.fatal('failed to register')
			process.exit(1)
		}
	})
})
