import fs from 'fs'
import { execSync } from 'child_process'

const now = Date.now()
const cfg = 'mirrorfs.cfg'
const setup = `lrun-mirrorfs --name doj_${now} --setup ${cfg}`
const teardown = `lrun-mirrorfs --name doj_${now} --teardown ${cfg}`

export const runRoot = `/tmp/doj/run_${now}`
export const dataRoot = `/etc/doj/data_${now}`
export const mirrorfs = execSync(setup).toString().trim()

function chmodConfig(mode: number) {
	const f = 'config.json'
	fs.existsSync(f) && fs.chmodSync(f, mode)
}

chmodConfig(0o600)
fs.mkdirSync(runRoot, { recursive: true })
fs.mkdirSync(dataRoot, { recursive: true })

function beforeExit() {
	execSync(teardown)
	fs.rmSync('/tmp/doj', { recursive: true })
	fs.rmSync('/etc/doj', { recursive: true })
}

process.on('beforeExit', beforeExit)
process.on('SIGINT', () => {
	beforeExit()
	process.exit()
})
