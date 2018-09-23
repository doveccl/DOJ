import { spawnSync } from 'child_process'

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

export const run = spawnSync

export const lrun = ({
	cmd = undefined as string,
	args = [] as any[],
	uid = 65534,
	gid = 65534,
	network = false,
	remountDev = true,
	passExitcode = false,
	maxCpuTime = undefined as number,
	maxRealTime = undefined as number,
	maxMemory = undefined as number,
	chroot = undefined as string,
	chdir = undefined as string,
	syscalls = false,
	stdin = undefined as any,
	stdout = undefined as any,
	stderr = undefined as any,
	env = {} as NodeJS.ProcessEnv,
	maxBuffer = 10 * 1024
}) => {
	// return spawnSync(cmd, args) // for mac test
	const builder = [ '--uid', uid, '--gid', gid, '--network', network ]
	builder.push('--remount-dev', remountDev, '--pass-exitcode', passExitcode)
	if (maxCpuTime) { builder.push('--max-cpu-time', maxCpuTime) }
	if (maxRealTime) { builder.push('--max-real-time', maxRealTime) }
	if (maxMemory) { builder.push('--max-memory', maxMemory) }
	if (chroot) { builder.push('--chroot', chroot) }
	if (chdir) { builder.push('--chdir', chdir) }
	if (syscalls) { builder.push('--syscalls', `!${BLACKLIST.join(',')}`) }
	return spawnSync('lrun', builder.concat(cmd, ...args).map(String), {
		env, maxBuffer, stdio: [ stdin, stdout, stderr, 'pipe' ]
	})
}
