import * as Router from 'koa-router'

import { compare } from '../../common/function'
import { ContestType, Group, Status } from '../../common/interface'
import { diffGroup, ensureGroup } from '../../common/user'
import { group, token } from '../middleware/auth'
import { contest, fetch, problem, user } from '../middleware/fetch'
import { DSubmission, Submission } from '../model/submission'
import { DUser, User } from '../model/user'
import { doJudge } from '../socket/judger'

const router = new Router()

router.use('/submission', token())

async function parseSubmission(record: DSubmission, self: DUser) {
	const admin = diffGroup(self, Group.admin)
	const { uid, pid, cid } = record
	const json = record.toJSON()
	const u = await user(uid) || { name: 'unknown', id: '' }
	const p = await problem(pid) || { title: 'unknown' }
	const c = cid && await contest(cid)
	if (!admin && !compare(self._id, u.id)) {
		/**
		 * if submission code is not open to others
		 * code should be invisible other common user
		 */
		if (!record.open) { delete json.code }
		/**
		 * for contest (before ending) submission code
		 * other common user also couldn't see it
		 */
		if (c && new Date() < c.endAt) { delete json.code }
	}
	if (!admin && c && new Date() < c.endAt) {
		/**
		 * if the scoreboard is frozen before a submission is submitted
		 * common user couldn't see result before ending
		 */
		if (c.type !== ContestType.ICPC || !compare(self._id, u.id)) {
			if (c.freezeAt <= record.createdAt) {
				delete json.cases
				json.result = {
					time: 0, memory: 0,
					status: Status.FREEZE
				}
			}
		}
	}
	/**
	 * attach additional data to record
	 */
	json.uname = u.name
	json.ptitle = p.title
	return json
}

router.get('/submission', async (ctx) => {
	const { uname, pid, cid } = ctx.query
	let { page, size } = ctx.query

	const condition: any = { uname, pid, cid }
	for (const k of ['uname', 'pid', 'cid']) {
		if (!condition[k]) { delete condition[k] }
	}
	if (uname) {
		const u = await User.findOne({ name: uname })
		if (u) { condition.uid = u._id, delete condition.uname }
	}

	page = parseInt(page, 10) || 1
	size = parseInt(size, 10) || 50
	const total = await Submission.countDocuments(condition)
	if (size === -1) { size = total }
	const arr = await Submission.find(condition)
		.select('-code').sort('-_id')
		.skip(size * (page - 1)).limit(size)
	const list: any[] = []
	for (const item of arr) {
		list.push(await parseSubmission(item, ctx.self))
	}
	ctx.body = { total, list }
})

router.get('/submission/:id', fetch('submission'), async (ctx) => {
	ctx.body = await parseSubmission(ctx.submission, ctx.self)
})

router.put('/submission/rejudge', group(Group.admin), async (ctx) => {
	ctx.body = await Submission.find(ctx.request.body)
	ctx.body.forEach(doJudge)
})

/**
 * user can only modify code visibility to others
 */
router.put('/submission/:id', fetch('submission'), async (ctx) => {
	const { self, submission } = ctx
	const { open } = ctx.request.body
	if (!compare(self._id, submission.uid)) { ensureGroup(self, Group.admin) }
	ctx.body = await submission.update({ open }, { runValidators: true })
})

router.post('/submission', async (ctx) => {
	const { body } = ctx.request
	body.uid = ctx.self._id
	body.cid = body.result = body.cases = undefined

	const p = await problem(body.pid)
	if (!p) { throw new Error('invalid problem id') }
	if (!p.data) { throw new Error('no data for this problem') }

	let freeze = false
	if (p.contest) {
		const c = await contest(p.contest.id)
		if (new Date() < c.startAt) {
			throw new Error('invalid before contest start')
		} else if (new Date() < c.endAt) {
			body.cid = p.contest.id
			if (c.type === ContestType.OI) {
				freeze = true
			}
		}
	}
	ctx.body = await Submission.create(body)
	if (freeze) { ctx.body.result.status = Status.FREEZE }
	doJudge(ctx.body)
})

export default router
