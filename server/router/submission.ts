import * as Router from 'koa-router'

const router = new Router()

router.get('/submission', async ctx => {
})

router.get('/submission/:id', async ctx => {
})

/**
 * 1. rejudge submission
 * 2. modify code visibility to others
 */
router.put('/submission/:id', async ctx => {
})

router.post('/submission', async ctx => {
})

export default router
