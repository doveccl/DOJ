import fs from 'fs'
import WebSocket from 'ws'
import { config } from '../util/config'
import { logJudger } from '../util/log'
import { prepareData } from './data'
import { mirrorfs, runRoot } from './path'
import { Case, CE } from '../../common/pack'
import { IResult, Status } from '../../common/interface'
import { interRun, lrun, lrunSync, wait, ExceedType, RunOpts, RunResult } from './run'

interface IJudge {
	_id: string
	code: string
	data: string
	language: number
	timeLimit: number
	memoryLimit: number
}

export async function judge(args: IJudge, ws?: WebSocket) {
	const step = (pending?: string, cases: IResult[] = []) => {
		ws?.send(JSON.stringify({ _id: args._id, pending, cases }))
	}

	logJudger.info('judge submission:', args._id)
	step('Prepare problem data and checker ...')
	const language = config.languages[args.language]
	const dataPath = await prepareData(args.data)
	const runPath = `${runRoot}/${args._id}`

	await fs.promises.mkdir(runPath, { recursive: true })
	await fs.promises.writeFile(`${runPath}/${language.source}`, args.code)
	await fs.promises.chmod(runPath, 0o777)
	fs.readdirSync(dataPath).forEach(file => {
		if (file.endsWith('.in')) return
		if (file.endsWith('.out')) return
		if (fs.existsSync(`${runPath}/${file}`)) return
		fs.promises.copyFile(`${dataPath}/${file}`, `${runPath}/${file}`)
	})

	// compile
	if (language.compile) {
		step('Compiling code ...')
		const result = lrunSync({
			cmd: language.compile.cmd,
			args: language.compile.args,
			maxRealTime: language.compile.time,
			passExitcode: true,
			chdir: runPath
		})
		logJudger.debug('Compiler return:', result.status)
		const { stdout, stderr } = result
		if (result.error) {
			return CE(args._id, result.error)
		} else if (result.status !== 0) {
			return CE(args._id, stdout + '' + stderr)
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
		step(`Judging case #${ith} ...`, cases)

		const rate = language.run.ratio
		const conf: RunOpts = {
			cmd: language.run.cmd,
			args: language.run.args,
			maxCpuTime: rate * args.timeLimit,
			maxRealTime: 2 * rate * args.timeLimit,
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

		logJudger.debug(`#${ith} run result:`, JSON.stringify(result))
		const { exceed, realTime, memory, signal, exitCode, error } = result
		if (exceed !== null) {
			switch (exceed) {
				case ExceedType.CPU_TIME:
				case ExceedType.REAL_TIME:
					cases.push(Case(Status.TLE, realTime, memory))
					break
				case ExceedType.MEMORY:
					cases.push(Case(Status.MLE, realTime, memory))
			}
		} else if (exitCode || error || signal) {
			let e = `Error (code=${exitCode}, signal=${signal})`
			if (error) e += ':\n' + error.trimEnd()
			cases.push(Case(Status.RE, realTime, memory, e))
		} else { // check output
			const res = lrunSync({
				cmd: checker,
				args: [inf, outf, ansf],
				passExitcode: true
			})
			const { stdout, stderr, status } = res
			const e = (stdout.toString() + stderr.toString()).trimEnd()
			logJudger.debug(`#${ith} checker result:`, JSON.stringify(e))
			cases.push(Case(status ? Status.WA : Status.AC, realTime, memory, e))
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
	return {
		cases, _id: args._id,
		result: Case(st, t, m)
	}
}
