import * as config from 'config'
import * as fs from 'fs-extra'

import { ILanguage, IResult, Status } from '../common/interface'
import { Case, CE, Pack } from '../common/pack'
import { prepareData } from './data'
import { logJudger } from './log'
import { interRun, lrun, lrunSync, wait, ExceedType, RunOpts, RunResult } from './run'

const mirrorfs = '/run/lrun/mirrorfs/doj'
const languages: ILanguage[] = config.get('languages')

export const judge = async (s: any): Promise<Pack> => {
	logJudger.info('judge submission:', s)
	const { _id, language, code, data, timeLimit, memoryLimit } = s
	const dataDir = await prepareData(data)
	const judgeDir = `/doj_tmp/${_id}`
	const lan = languages[language]
	await fs.outputFile(`${judgeDir}/${lan.source}`, code)
	await fs.chmod(judgeDir, 0o777)
	/**
	 * compile source code
	 */
	if (lan.compile) {
		const result = lrunSync({
			cmd: lan.compile.cmd,
			args: lan.compile.args,
			maxRealTime: lan.compile.time,
			passExitcode: true,
			chdir: judgeDir
		})
		logJudger.debug('compile result:', result)
		if (result.status !== 0) {
			const { stdout, stderr, output } = result
			const e = stdout.toString() + stderr.toString()
			if (e.trim()) { return CE(_id, e) }
			if (output[3]) { return CE(_id, output[3].toString()) }
			return CE(_id)
		}
	}
	/**
	 * run test data
	 */
	const checker = `${dataDir}/checker`
	const interPath = `${dataDir}/interactor`
	const interactor = fs.existsSync(interPath) && interPath
	const cases: IResult[] = []
	let ith = 0
	do {
		let result: RunResult
		const inf = `${dataDir}/${ith}.in`
		const ansf = `${dataDir}/${ith}.out`
		const ouf = `${judgeDir}/output`
		const maxCpuTime = lan.run.ratio * timeLimit
		const maxRealTime = 1.5 * maxCpuTime
		const conf: RunOpts = {
			cmd: lan.run.cmd,
			args: lan.run.args,
			maxCpuTime, maxRealTime,
			maxMemory: memoryLimit,
			chroot: mirrorfs,
			chdir: judgeDir,
			syscalls: true
		}
		if (interactor) {
			result = await interRun(conf, {
				cmd: interactor,
				args: [ inf, ouf ],
				remountDev: false
			})
		} else {
			conf.stdin = fs.openSync(inf, 'r')
			conf.stdout = fs.openSync(ouf, 'w')
			result = await wait(lrun(conf))
			fs.closeSync(conf.stdin)
			fs.closeSync(conf.stdout)
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
		} else {
			const res = lrunSync({
				cmd: checker,
				args: [ inf, ouf, ansf ],
				passExitcode: true
			})
			logJudger.debug('checker result:', res)
			const { stdout, stderr, status } = res
			const e = stdout.toString() + stderr.toString()
			cases.push(Case(status ? Status.WA : Status.AC, cpuTime, memory, e))
		}
		ith++
	} while (
		fs.existsSync(`${dataDir}/${ith}.in`) &&
		fs.existsSync(`${dataDir}/${ith}.out`)
	)
	/**
	 * wrap result
	 */
	let t = 0
	let m = 0
	let st = Status.AC
	for (const cas of cases) {
		if (t < cas.time) { t = cas.time }
		if (m < cas.memory) { m = cas.memory }
		if (st !== Status.AC || cas.status === Status.AC) { continue }
		st = cas.status
	}
	logJudger.info('cases:', cases)
	return { _id, cases, result: Case(st, t, m) }
}
