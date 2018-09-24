import { spawn, spawnSync, ChildProcess } from 'child_process'
import { join } from 'path'

const BLACKLIST = [
	'execve',
	'flock',
	'ptrace',
	'sync',
	'fdatasync',
	'fsync',
	'msync',
	'sync_file_range',
	'syncfs',
	'unshare',
	'setns',
	'clone',
	'query_module',
	'sysinfo',
	'syslog',
	'sysfs'
]

export interface RunOpts {
	cmd: string
	args?: string[]
	uid?: number
	gid?: number
	network?: boolean
	remountDev?: boolean
	passExitcode?: boolean
	maxCpuTime?: number
	maxRealTime?: number
	maxMemory?: number
	chroot?: string
	chdir?: string
	syscalls?: boolean
	stdin?: any
	stdout?: any
	stderr?: any
	[index: string]: any
}

const defaultOpts: Partial<RunOpts> = {
	args: [],
	uid: 65534,
	gid: 65534,
	network: false,
	remountDev: true,
	passExitcode: false,
	syscalls: false
}

const buildArgs = (o: RunOpts) => {
	for (const key in defaultOpts) {
		if (o[key] === undefined) {
			o[key] = defaultOpts[key]
		}
	}
	const builder = [ '--uid', o.uid, '--gid', o.gid, '--network', o.network ]
	builder.push('--remount-dev', o.remountDev, '--pass-exitcode', o.passExitcode)
	if (!o.chroot) { builder.push('--tmpfs', join(__dirname, '../config'), 0) }
	if (o.maxCpuTime) { builder.push('--max-cpu-time', o.maxCpuTime) }
	if (o.maxRealTime) { builder.push('--max-real-time', o.maxRealTime) }
	if (o.maxMemory) { builder.push('--max-memory', o.maxMemory) }
	if (o.chroot) { builder.push('--chroot', o.chroot) }
	if (o.chdir) { builder.push('--chdir', o.chdir) }
	if (o.syscalls) { builder.push('--syscalls', `!${BLACKLIST.join(',')}`) }
	return builder.concat(o.cmd, ...o.args).map(String)
}

export const lrunSync = (opts: RunOpts) => {
	const { stdin, stdout, stderr } = opts
	const stdio = [ stdin, stdout, stderr, 'pipe' ]
	const o = { env: {}, maxBuffer: 10240, stdio }
	return spawnSync('lrun', buildArgs(opts), o)
}

export const lrun = (opts: RunOpts) => {
	const { stdin, stdout, stderr } = opts
	const o = { env: {}, stdio: [ stdin, stdout, stderr, 'pipe' ] }
	return spawn('lrun', buildArgs(opts), o)
}

export enum ExceedType {
	CPU_TIME,
	REAL_TIME,
	MEMORY
}
export interface RunResult {
	memory: number
	cpuTime: number
	realTime: number
	exitCode: number
	signal: number
	exceed: null | ExceedType
}
const getExceedType = (str: string) => {
	switch (str) {
		case 'CPU_TIME': return ExceedType.CPU_TIME
		case 'REAL_TIME': return ExceedType.REAL_TIME
		case 'MEMORY': return ExceedType.MEMORY
		default: return null
	}
}
export const parseResult = (res: string): RunResult => ({
	memory: Number(res.match(/MEMORY\s+(\d+)/)[1]),
	cpuTime: Number(res.match(/CPUTIME\s+([0-9.]+)/)[1]),
	realTime: Number(res.match(/REALTIME\s+([0-9.]+)/)[1]),
	exitCode: Number(res.match(/EXITCODE\s+(\d+)/)[1]),
	signal: Number(res.match(/TERMSIG\s+(\d+)/)[1]),
	exceed: getExceedType(res.match(/EXCEED\s+(\w+)/)[1])
})

export const wait = (cp: ChildProcess) => new Promise<RunResult>((resolve, reject) => {
	let fd3 = ''
	cp.stdio[3].on('data', (r) => fd3 += r)
	cp.on('close', () => {
		try {
			resolve(parseResult(fd3))
		} catch (e) { reject(e) }
	})
})
