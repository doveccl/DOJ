import { scheduleJob } from 'node-schedule'

import { Status } from '../../common/interface'
import { clearCache } from '../middleware/fetch'
import { Problem } from '../model/problem'
import { Submission } from '../model/submission'
import { User } from '../model/user'
import { logServer } from './log'

export const startCron = () => scheduleJob(
	'update solve and submit count',
	'0 1 * * *',
	async () => {
		logServer.info('daily update')
		const users = await User.find()
		const problems = await Problem.find()
		for (const user of users) {
			await user.update({
				solve: await Submission.find({
					'uid': user._id,
					'result.status': Status.AC
				}).countDocuments(),
				submit: await Submission.find({
					uid: user._id
				}).countDocuments()
			})
		}
		for (const problem of problems) {
			await problem.update({
				solve: await Submission.find({
					'pid': problem._id,
					'result.status': Status.AC
				}).countDocuments(),
				submit: await Submission.find({
					pid: problem._id
				}).countDocuments()
			})
		}
		clearCache()
	}
)
