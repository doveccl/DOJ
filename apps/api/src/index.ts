import { Hono } from 'hono'
import { logger } from 'hono/logger'
import { ZodError } from 'zod'
import { config } from './config'
import { ApiHttpError, apiError, validationIssues } from './errors'
import { browserWebSocketHandlers, handleBrowserUpgrade, type BrowserSocketData } from './browserWs'
import { registerAdminCoreRoutes } from './routes/admin-core'
import { agentWebSocketHandlers, handleAgentUpgrade, registerAgentRoutes, type AgentSocketData } from './routes/agents'
import { registerAssignmentRoutes } from './routes/assignments'
import { registerAuthRoutes } from './routes/auth'
import { registerContestRoutes } from './routes/contests'
import { registerDiscussionRoutes } from './routes/discussion'
import { registerProblemRoutes } from './routes/problems'
import { registerPublicRoutes } from './routes/public'
import { registerSubmissionRoutes } from './routes/submissions'
import { getRuntimeSettings } from './settings'
import { startJudgeScheduler } from './judge'
import { startStatsRepairCron } from './services/stats'

const app = new Hono()

app.use('*', logger())

app.onError((error, c) => {
  if (error instanceof ZodError) {
    return apiError(c, 400, 'VALIDATION_ERROR', 'Invalid request payload', validationIssues(error.issues))
  }

  if (error instanceof ApiHttpError) {
    return apiError(c, error.status, error.code, error.message, error.issues)
  }

  console.error(error)
  return apiError(c, 500, 'INTERNAL_SERVER_ERROR', 'Internal server error')
})

app.notFound((c) => apiError(c, 404, 'NOT_FOUND', 'Resource not found'))

app.get('/health', (c) =>
  c.json({
    ok: true,
    service: 'doj-api'
  })
)

app.get('/api/config', async (c) => {
  const settings = await getRuntimeSettings()
  return c.json({
    signup: settings.general.signup,
    guestAccess: settings.general.guestAccess,
    publicCode: settings.general.publicCode,
    smtpConfigured: settings.smtp.enabled && Boolean(settings.smtp._host && settings.smtp.from),
    aiEnabled: settings.ai.enabled,
    notice: settings.general.notice
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
registerAgentRoutes(app)

type ApiSocketData = BrowserSocketData | AgentSocketData

const apiServer = Bun.serve<ApiSocketData>({
  port: config.port,
  fetch(request, server) {
    const url = new URL(request.url)
    if (url.pathname === '/api/agents/connect') {
      return handleAgentUpgrade(request, server as unknown as Bun.Server<AgentSocketData>) ?? new Response(null, { status: 101 })
    }
    if (url.pathname === '/api/ws') {
      return handleBrowserUpgrade(request, server as unknown as Bun.Server<BrowserSocketData>) ?? new Response(null, { status: 101 })
    }
    return app.fetch(request)
  },
  websocket: {
    open(ws) {
      if (ws.data?.kind === 'browser') {
        browserWebSocketHandlers.open?.(ws as Bun.ServerWebSocket<BrowserSocketData>)
      } else {
        agentWebSocketHandlers.open?.(ws as Bun.ServerWebSocket<AgentSocketData>)
      }
    },
    message(ws, message) {
      if (ws.data?.kind === 'browser') {
        void browserWebSocketHandlers.message?.(ws as Bun.ServerWebSocket<BrowserSocketData>, message)
      } else {
        agentWebSocketHandlers.message?.(ws as Bun.ServerWebSocket<AgentSocketData>, message)
      }
    },
    close(ws, code, reason) {
      if (ws.data?.kind === 'browser') {
        browserWebSocketHandlers.close?.(ws as Bun.ServerWebSocket<BrowserSocketData>, code, reason)
      } else {
        agentWebSocketHandlers.close?.(ws as Bun.ServerWebSocket<AgentSocketData>, code, reason)
      }
    }
  }
})

console.log(`DOJ API listening on http://localhost:${apiServer.port}`)
startJudgeScheduler()
startStatsRepairCron()
