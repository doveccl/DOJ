import { run } from './run'

const p = run({
	cmd: 'ls',
	args: [ '/' ]
})

console.log(p)
