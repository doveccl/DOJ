import Router from '@koa/router'

import { Group, Status } from '../../common/interface'
import { ensureGroup } from '../../common/user'
import { group, token } from '../middleware/auth'
import { contest, fetch } from '../middleware/fetch'
import { File } from '../model/file'
import { DProblem, Problem } from '../model/problem'
import { Submission } from '../model/submission'
import { DUser } from '../model/user'
import type { FilterQuery } from 'mongoose'

const router = new Router<any, {
  self: DUser
  problem: DProblem
}>()

router.use('/problem', token())

router.get('/problem', async (ctx) => {
  const { all, cid, search, page: p, size: s } = ctx.query
  const page = typeof p === 'string' ? +p : 1
  const size = typeof s === 'string' ? +s : 50

  all && ensureGroup(ctx.self, Group.admin)

  const condition: FilterQuery<DProblem> = {}
  if (typeof cid === 'string') condition['contest.id'] = cid
  if (typeof search === 'string') {
    const esc = search.replace(/[-\/\\^$*+?.()|[\]{}]/g, '\\$&')
    const searchRegExp = new RegExp(esc, 'i')
    const $elemMatch = { $eq: search }
    condition.$or = [
      { tags: { $elemMatch } },
      { title: searchRegExp },
      { content: searchRegExp }
    ]
  }

  const total = await Problem.count(condition)
  const arr = await Problem.find(condition)
    .sort(all ? '-_id' : '_id')
    .select(all ? '' : '-content -data')
    .skip(size * (page - 1)).limit(size)
  const list: any[] = []
  for (const item of arr) {
    if (!all && item.contest) {
      const c = await contest(item.contest.id)
      const enableAt = cid ? c?.startAt : c?.endAt
      if (enableAt && new Date() < enableAt) continue
    }
    list.push({
      ...item.toJSON(),
      solved: await Submission.count({
        pid: item._id,
        uid: ctx.self._id,
        'result.status': Status.AC
      })
    })
  }
  ctx.body = { total, list }
})

router.get('/problem/:id', fetch('problem'), async (ctx) => {
  if (ctx.problem.contest) {
    const c = await contest(ctx.problem.contest.id)
    if (c && new Date() < c.startAt) ensureGroup(ctx.self, Group.admin)
  }
  ctx.body = ctx.problem.toJSON()
  delete ctx.body.data
})

router.post('/problem', group(Group.admin), async (ctx) => {
  ctx.body = await Problem.create(ctx.request.body)
  await File.findByIdAndUpdate(ctx.request.body.data, { metadata: { type: 'data' } })
})

router.put('/problem/:id', group(Group.admin), fetch('problem'), async (ctx) => {
  await File.findByIdAndUpdate(ctx.request.body.data, { metadata: { type: 'data' } })
  ctx.body = await ctx.problem.updateOne(ctx.request.body, { runValidators: true })
  ctx.problem.set(ctx.request.body)
})

router.del('/problem/:id', group(Group.admin), fetch('problem'), async (ctx) => {
  ctx.body = await ctx.problem.deleteOne()
})

export default router
