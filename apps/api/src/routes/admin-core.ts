import { Hono } from 'hono'
import { registerAdminAgentRoutes } from './admin-agents'
import { registerAdminGroupRoutes } from './admin-groups'
import { registerAdminLanguageRoutes } from './admin-languages'
import { registerAdminUserRoutes } from './admin-users'

export function registerAdminCoreRoutes(app: Hono) {
  registerAdminGroupRoutes(app)
  registerAdminUserRoutes(app)
  registerAdminLanguageRoutes(app)
  registerAdminAgentRoutes(app)
}
