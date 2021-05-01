import { spawn, spawnSync, ChildProcess } from 'child_process'

const env = {
	HOME: '/tmp',
	PATH: '/bin:/usr/bin:/usr/local/bin'
}

const BLACKLIST = [
	'clone[a&268435456==268435456]',
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
	maxStack?: number
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

const buildArgs = (args: RunOpts) => {
	const o = Object.assign({}, defaultOpts, args)
	const builder = ['--uid', o.uid, '--gid', o.gid, '--network', o.network]
	builder.push('--remount-dev', o.remountDev, '--pass-exitcode', o.passExitcode)
	if (o.syscalls) builder.push('--syscalls', `!${BLACKLIST.join(',')}`)
	if (o.maxRealTime) builder.push('--max-real-time', o.maxRealTime)
	if (o.maxCpuTime) builder.push('--max-cpu-time', o.maxCpuTime)
	if (o.maxMemory) builder.push('--max-memory', o.maxMemory)
	if (o.maxStack) builder.push('--max-stack', o.maxStack)
	if (o.chroot) builder.push('--chroot', o.chroot)
	if (o.chdir) builder.push('--chdir', o.chdir)
	return builder.concat(o.cmd, o.args || []).map(String)
}

export const lrunSync = (opts: RunOpts) => {
	const { stdin, stdout, stderr } = opts
	const stdio = [stdin, stdout, stderr, 'pipe']
	return spawnSync('lrun', buildArgs(opts), { maxBuffer: 10240, stdio, env })
}

export const lrun = (opts: RunOpts) => {
	const { stdin, stdout, stderr } = opts
	const stdio = [stdin, stdout, stderr, 'pipe']
	return spawn('lrun', buildArgs(opts), { stdio, env })
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
	error?: string // user program stderr output
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
	let fd2 = '', fd3 = ''
	cp.stdio[2].on('data', c => fd2 += c)
	cp.stdio[3].on('data', c => fd3 += c)
	cp.on('close', () => {
		try {
			const res = parseResult(fd3)
			if (fd2) res.error = fd2
			resolve(res)
		} catch {
			reject(fd2 + fd3)
		}
	})
})

export const interRun = (test: RunOpts, inter: RunOpts) => new Promise<RunResult>((resolve) => {
	let t3 = '', i3 = '', work = 2
	const t = lrun(test), i = lrun(inter)
	const ret = () => resolve(parseResult(t3 || i3))
	t.stdio[3].on('data', (r) => t3 += r)
	i.stdio[3].on('data', (r) => i3 += r)
	t.on('close', () => --work || ret())
	i.on('close', () => --work || ret())
	t.stdin.on('error', () => t.kill())
	i.stdin.on('error', () => i.kill())
	t.stdout.pipe(i.stdin)
	i.stdout.pipe(t.stdin)
})
