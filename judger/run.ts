import { spawn, spawnSync } from 'child_process'

const BLACKLIST = [
	'getdents',
	'execve',
	'ptrace',
	'clone',
	'fork',
	'vfork'
]

export const run = ({
	cmd = undefined as string,
	args = [] as any[],
	uid = 65534,
	gid = 65534,
	network = false,
	maxCpuTime = 1,
	maxRealTime = 1.5,
	maxMemory = 32 * 1024 * 1024,
	mirrorfs = undefined as string,
	syscalls = false
}) => {
	const builder: any[] = []
	if (uid) { builder.push('--uid', uid) }
	if (gid) { builder.push('--gid', gid) }
	if (maxCpuTime) { builder.push('--max-cpu-time', maxCpuTime) }
	if (maxRealTime) { builder.push('--max-real-time', maxRealTime) }
	if (maxMemory) { builder.push('--max-memory', maxMemory) }
	if (syscalls) { builder.push('--syscalls', `!${BLACKLIST.join(',')}`) }
	if (network !== undefined) { builder.push('--network', network) }
	if (mirrorfs) {
		spawnSync('lrun-mirrorfs', [ '--name', mirrorfs, '--setup', 'config/mirrorfs' ])
		builder.push('--chroot', `/run/lrun/mirrorfs/${mirrorfs}`)
	}
	// return await spawn('lrun', builder)
	return spawnSync(
		'docker',
		[
			'run', '--rm', '--privileged', 'doj',
			'lrun', ...builder,
			'--', cmd, ...args
		],
		{
			stdio: [ null, null, null, 'pipe' ]
		}
	)
}
