import { Hono } from 'hono'
import { authMiddleware, requireAdmin } from '../auth'
import { getLastRepairStatus, repairDerivedStats } from '../services/stats'
import { registerAdminAgentRoutes } from './admin-agents'
import { registerAdminGroupRoutes } from './admin-groups'
import { registerAdminLanguageRoutes } from './admin-languages'
import { registerAdminUserRoutes } from './admin-users'

export function registerAdminCoreRoutes(app: Hono) {
  app.get('/api/admin/stats/repair', authMiddleware, async (c) => {
    const denied = await requireAdmin(c)
    if (denied) return denied
    return c.json(await getLastRepairStatus())
  })

  app.post('/api/admin/stats/repair', authMiddleware, async (c) => {
    const denied = await requireAdmin(c)
    if (denied) return denied
    return c.json(await repairDerivedStats())
  })

  registerAdminGroupRoutes(app)
  registerAdminUserRoutes(app)
  registerAdminLanguageRoutes(app)
  registerAdminAgentRoutes(app)
}
