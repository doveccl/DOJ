import { Hono } from 'hono'
import { logger } from 'hono/logger'
import { ZodError } from 'zod'
import { config } from './config'
import { registerAdminCoreRoutes } from './routes/admin-core'
import { registerAssignmentRoutes } from './routes/assignments'
import { registerAuthRoutes } from './routes/auth'
import { registerBbsRoutes } from './routes/bbs'
import { registerContestRoutes } from './routes/contests'
import { registerProblemRoutes } from './routes/problems'
import { registerPublicRoutes } from './routes/public'
import { registerSubmissionRoutes } from './routes/submissions'
import { getRuntimeSettings } from './settings'

const app = new Hono()

app.use('*', logger())

app.onError((error, c) => {
  if (error instanceof ZodError) {
    return c.json(
      {
        code: 'BAD_REQUEST',
        message: 'Invalid request payload',
        issues: error.issues
      },
      400
    )
  }

  console.error(error)
  return c.json(
    {
      code: 'INTERNAL_SERVER_ERROR',
      message: error instanceof Error ? error.message : String(error)
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
registerBbsRoutes(app)

Bun.serve({
  port: config.port,
  fetch: app.fetch
})

console.log(`DOJ API listening on http://localhost:${config.port}`)
