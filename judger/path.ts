import { execSync } from 'child_process'
import { mkdtempSync } from 'fs-extra'

const cfg = './mirrorfs.cfg'
const name = `doj_${Math.random().toString(16).substr(2)}`
const setup = `lrun-mirrorfs --name ${name} --setup ${cfg}`
const teardown = `lrun-mirrorfs --name ${name} --teardown ${cfg}`

export const runRoot = mkdtempSync('/tmp/doj/run/')
export const dataRoot = mkdtempSync('/etc/doj/data_')
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
