import { Hono } from 'hono'
import { logger } from 'hono/logger'
import { ZodError } from 'zod'
import { config } from './config'
import { registerAdminCoreRoutes } from './routes/admin-core'
import { registerAssignmentRoutes } from './routes/assignments'
import { registerAuthRoutes } from './routes/auth'
import { registerContestRoutes } from './routes/contests'
import { registerDiscussionRoutes } from './routes/discussion'
import { registerMediaRoutes } from './routes/media'
import { registerProblemRoutes } from './routes/problems'
import { registerPublicRoutes } from './routes/public'
import { registerSubmissionRoutes } from './routes/submissions'
import { getRuntimeSettings } from './settings'

const app = new Hono()

app.use('*', logger())

app.onError((error, c) => {
  if (error instanceof ZodError) {
    const issue = error.issues[0]
    const path = issue?.path.join('.')
    const detail = issue ? `${path ? `${path}: ` : ''}${issue.message}` : 'Invalid request payload'
    return c.json(
      {
        code: 'BAD_REQUEST',
        message: detail,
        issues: error.issues
      },
      400
    )
  }

  console.error(error)
  return c.json(
    {
      code: 'INTERNAL_SERVER_ERROR',
      message: 'Internal server error'
    },
    500
  )
})

app.get('/health', (c) =>
  c.json({
    ok: true,
    service: 'doj-api'
  })
)

app.get('/api/config', async (c) => {
  const settings = await getRuntimeSettings()
  return c.json({
    registration: settings.registrationEnabled,
    registrationInviteRequired: settings.registrationInviteCode.length > 0,
    aiCoachingEnabled: settings.aiCoachingEnabled,
    guestProblemsetVisible: settings.guestProblemsetVisible,
    sourceOpenDefault: settings.sourceOpenDefault
  })
})

registerAuthRoutes(app)
registerAdminCoreRoutes(app)
registerPublicRoutes(app)
registerProblemRoutes(app)
registerAssignmentRoutes(app)
registerContestRoutes(app)
registerSubmissionRoutes(app)
registerDiscussionRoutes(app)
registerMediaRoutes(app)

Bun.serve({
  port: config.port,
  fetch: app.fetch
})

console.log(`DOJ API listening on http://localhost:${config.port}`)
