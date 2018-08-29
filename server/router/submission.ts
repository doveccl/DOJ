import * as Router from 'koa-router'

import { IUser } from '../model/user'
import Submission, { ISubmission, Status } from '../model/submission'

import { isGroup } from '../middleware/auth'
import { user, problem, contest, urlFetch } from '../middleware/fetch'
import { toStringCompare } from '../util/function'

const router = new Router()

async function parseSubmission(record: ISubmission, self: IUser) {
	const admin = isGroup(self, 'admin')
	const { uid, pid, cid } = record
	const u = await user(uid)
	const p = await problem(pid)
	const c = await contest(cid)
	if (!admin && !toStringCompare(self._id, u.id)) {
		/**
		 * if submission code is not open to others
		 * code should be invisible other common user
		 */
		if (!record.open) { delete record.code }
		/**
		 * for contest (before ending) submission code
		 * other common user also couldn't see it
		 */
		if (c && new Date() < c.endAt) { delete record.code }
	}
	if (!admin && c && new Date() < c.endAt) {
		/**
		 * if the scoreboard is frozen before a submission is submitted
		 * common user (include itself) couldn't see result before ending
		 */
		if (c.freezeAt <= record.createdAt) {
			delete record.cases
			record.result = {
				time: 0, memory: 0,
				status: Status.WAIT
			}
		}
	}
	const result = record.toJSON()
	/**
	 * attach additional data to record
	 */
	result.uname = u.name
	result.ptitle = p.title
	if (c) { result.ctitle = c.title }
	return result
}

router.get('/submission', async ctx => {
	let { page, size, uid, pid, cid } = ctx.query
	const condition: any = { uid, pid, cid }
	for (let k of ['uid', 'pid', 'cid']) {
		if (!condition[k]) { delete condition[k] }
	}

	page = parseInt(page) || 1
	size = parseInt(size) || 50
	const total = await Submission.countDocuments(condition)
	if (size === -1) { size = total }
	const arr = await Submission.find(condition)
		.select('-code -cases')
		.skip(size * (page - 1)).limit(size)
	const list: any[] = []
	for (let item of arr) {
		list.push(await parseSubmission(item, ctx.self))
	}
	ctx.body = { total, list }
})

router.get('/submission/:id', urlFetch('submission'), async ctx => {
	ctx.body = await parseSubmission(ctx.submission, ctx.self)
})

/**
 * user can only modify code visibility to others
 */
router.put('/submission/:id', urlFetch('submission'), async ctx => {
	const { open } = ctx.request.body
	ctx.body = await ctx.submission.update({ open }, { runValidators: true })
})

router.post('/submission', async ctx => {
	const { body } = ctx.request
	const p = await problem(body.pid)
	body.uid = ctx.self._id
	body.cid = body.result = body.cases = undefined

	if (!p) { throw new Error('invalid problem id') }
	if (p.contest) {
		const c = await contest(p.contest.id)
		if (new Date() < c.startAt) {
			throw new Error('invalid before contest start')
		} else if (new Date() < c.endAt) {
			body.cid = p.contest.id
		}
	}
	ctx.body = await Submission.create(ctx.request.body)
	// add submission id (ctx.body._id) to judge queue
})

export default router
