import { execSync } from 'child_process'

const cfg = './mirrorfs.cfg'
const rand = Math.random().toString(16).substr(2)
const setup = `lrun-mirrorfs --name doj_${rand} --setup ${cfg}`
const teardown = `lrun-mirrorfs --name doj_${rand} --teardown ${cfg}`

export const runRoot = `/tmp/doj/run_${rand}`
export const dataRoot = `/etc/doj/data_${rand}`
export const mirrorfs = execSync(setup).toString().trim()

execSync('chmod -R 600 config/*')
process.on('beforeExit', () => {
	execSync(teardown)
	execSync('chmod -R 644 config/*')
})
process.on('SIGINT', () => {
	execSync(teardown)
	execSync('chmod -R 644 config/*')
	process.exit()
})
