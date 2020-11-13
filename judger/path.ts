import { spawnSync } from 'child_process'
import { pathExistsSync, mkdirpSync, removeSync } from 'fs-extra'

export const runRoot = '/doj/run'
export const dataRoot = '/doj/data'
export const mirrorfs = '/run/lrun/mirrorfs/doj'

// keep judger root clean
removeSync(runRoot)
mkdirpSync(runRoot)

if (!pathExistsSync(mirrorfs)) {
	spawnSync('lrun-mirrorfs', [
    '--name', 'doj',
		'--setup', 'mirrorfs.cfg'
	])
}

function teardown() {
	spawnSync('lrun-mirrorfs', [
		'--name', 'doj',
		'--teardown', 'mirrorfs.cfg'
	])
}

process.on('beforeExit', teardown)
process.on('SIGINT', () => {
	teardown()
	process.exit()
})
