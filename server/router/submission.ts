import * as Router from 'koa-router'

import { ensureGroup, isGroup, token } from '../middleware/auth'
import { contest, problem, urlFetch, user } from '../middleware/fetch'
import { ISubmission, Status, Submission } from '../model/submission'
import { IUser, User } from '../model/user'
import { toStringCompare } from '../util/function'

const router = new Router()

router.use('/submission', token())

async function parseSubmission(record: ISubmission, self: IUser) {
	const admin = isGroup(self, 'admin')
	const { uid, pid, cid } = record
	const json = record.toJSON()
	const u = await user(uid)
	const p = await problem(pid)
	const c = await contest(cid)
	if (!admin && !toStringCompare(self._id, u.id)) {
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
		 * common user (include itself) couldn't see result before ending
		 */
		if (c.freezeAt <= record.createdAt) {
			delete json.cases
			json.result = {
				time: 0, memory: 0,
				status: Status.WAIT
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
		.select('-code -cases').sort('-_id')
		.skip(size * (page - 1)).limit(size)
	const list: any[] = []
	for (const item of arr) {
		list.push(await parseSubmission(item, ctx.self))
	}
	ctx.body = { total, list }
})

router.get('/submission/:id', urlFetch('submission'), async (ctx) => {
	ctx.body = await parseSubmission(ctx.submission, ctx.self)
})

/**
 * user can only modify code visibility to others
 */
router.put('/submission/:id', urlFetch('submission'), async (ctx) => {
	const { self, submission } = ctx
	const { open } = ctx.request.body
	if (!toStringCompare(self._id, submission.uid)) { ensureGroup(self, 'admin') }
	ctx.body = await submission.update({ open }, { runValidators: true })
})

router.post('/submission', async (ctx) => {
	const { body } = ctx.request
	body.uid = ctx.self._id
	body.cid = body.result = body.cases = undefined

	const p = await problem(body.pid)
	if (!p) { throw new Error('invalid problem id') }
	if (p.contest) {
		const c = await contest(p.contest.id)
		if (new Date() < c.startAt) {
			throw new Error('invalid before contest start')
		} else if (new Date() < c.endAt) {
			body.cid = p.contest.id
		}
	}
	ctx.body = await Submission.create(body)
	// add submission id (ctx.body._id) to judge queue
})

export default router
