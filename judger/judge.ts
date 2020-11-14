import fs from 'fs-extra'
import config from 'config'

import { logJudger } from './log'
import { prepareData } from './data'
import { Case, CE } from '../common/pack'
import { mirrorfs, runRoot } from './path'
import { ILanguage, IResult, Status } from '../common/interface'
import { interRun, lrun, lrunSync, wait, ExceedType, RunOpts, RunResult } from './run'

const languages: ILanguage[] = config.get('languages')

interface IJudge {
	_id: string
	code: string
	data: string
	language: number
	timeLimit: number
	memoryLimit: number
}

export async function judge(args: IJudge) {
	logJudger.info('judge submission:', args._id)
	const language = languages[args.language]
	const dataPath = await prepareData(args.data)
	const runPath = `${runRoot}/${args._id}`

	await fs.outputFile(`${runPath}/${language.source}`, args.code)
	fs.readdirSync(dataPath).forEach(file => {
		if (file.endsWith('.in')) return
		if (file.endsWith('.out')) return
		if (fs.existsSync(`${runPath}/${file}`)) return
		fs.copySync(`${dataPath}/${file}`, `${runPath}/${file}`)
	})

	// compile
	if (language.compile) {
		await fs.chmod(runPath, 0o777)
		const result = lrunSync({
			cmd: language.compile.cmd,
			args: language.compile.args,
			maxRealTime: language.compile.time,
			passExitcode: true,
			chdir: runPath
		})
		logJudger.debug('Compiler return:', result.status)
		if (result.error) {
			return CE(args._id, result.error)
		} else if (result.status !== 0) {
			return CE(args._id, result.stderr.toString())
		}
	}

	// judge
	const checker = `${dataPath}/checker`
	const interPath = `${dataPath}/interactor`
	const interactor = fs.existsSync(interPath) && interPath
	let ith = 0, cases = new Array<IResult>()
	do {
		let result: RunResult
		const inf = `${dataPath}/${ith}.in`
		const ansf = `${dataPath}/${ith}.out`
		const outf = `${runPath}/${ith}.output`

		const rate = language.run.ratio
		const conf: RunOpts = {
			cmd: language.run.cmd,
			args: language.run.args,
			maxCpuTime: rate * args.timeLimit,
			maxRealTime: 1.5 * rate * args.timeLimit,
			maxMemory: args.memoryLimit,
			maxStack: args.memoryLimit,
			chroot: mirrorfs,
			chdir: runPath,
			syscalls: true
		}

		if (interactor) {
			result = await interRun(conf, {
				cmd: interactor,
				args: [inf, outf],
				remountDev: false
			})
		} else {
			conf.stdin = fs.openSync(inf, 'r')
			conf.stdout = fs.openSync(outf, 'w')
			result = await wait(lrun(conf))
			fs.closeSync(conf.stdout)
			fs.closeSync(conf.stdin)
		}

		logJudger.debug(`#${ith} run result:`, result)
		const { exceed, cpuTime, memory, signal } = result
		if (exceed !== null) {
			switch (exceed) {
				case ExceedType.CPU_TIME:
				case ExceedType.REAL_TIME:
					cases.push(Case(Status.TLE, cpuTime, memory))
					break
				case ExceedType.MEMORY:
					cases.push(Case(Status.MLE, cpuTime, memory))
			}
		} else if (signal) {
			const e = `process signaled (number: ${signal})`
			cases.push(Case(Status.RE, cpuTime, memory, e))
		} else { // check output
			const res = lrunSync({
				cmd: checker,
				args: [inf, outf, ansf],
				passExitcode: true
			})
			logJudger.debug('checker result:', res)
			const { stdout, stderr, status } = res
			const e = stdout.toString() + stderr.toString()
			cases.push(Case(status ? Status.WA : Status.AC, cpuTime, memory, e))
		}
		ith++
	} while (
		fs.existsSync(`${dataPath}/${ith}.in`) &&
		fs.existsSync(`${dataPath}/${ith}.out`)
	)

	// pack result
	let t = 0, m = 0, st = Status.AC
	for (const cas of cases) {
		if (t < cas.time) t = cas.time
		if (m < cas.memory) m = cas.memory
		if (st === Status.AC) st = cas.status
	}
	logJudger.info('cases:', cases)
	return {
		cases, _id: args._id,
		result: Case(st, t, m)
	}
}
