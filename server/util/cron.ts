import { Status } from '../../common/interface'
import { clearCache } from '../middleware/fetch'
import { Problem } from '../model/problem'
import { Submission } from '../model/submission'
import { User } from '../model/user'
import { logServer } from './log'

async function updateRatio() {
  logServer.info('update ac ratio')
  const users = await User.find()
  const problems = await Problem.find()
  for (const user of users) {
    await user.updateOne({
      solve: await Submission.count({ uid: user._id, 'result.status': Status.AC }),
      submit: await Submission.count({ uid: user._id })
    })
  }
  for (const problem of problems) {
    await problem.updateOne({
      solve: await Submission.count({ pid: problem._id, 'result.status': Status.AC }),
      submit: await Submission.count({ pid: problem._id })
    })
  }
}

export function startCron() {
  setInterval(updateRatio, 60 * 60 * 1e3)
  setInterval(clearCache, 4 * 60 * 60 * 1e3)
}
