import { closeDb, db, schema } from './client'
import { eq } from 'drizzle-orm'

const builtinGroups = [
  { key: 'admin', name: 'Admin', description: 'System administrators', builtin: true },
  { key: 'user', name: 'User', description: 'Registered users', builtin: true },
  { key: 'guest', name: 'Guest', description: 'Anonymous or low-trust users', builtin: true }
]

await db
  .insert(schema.groups)
  .values(builtinGroups)
  .onConflictDoNothing({ target: schema.groups.key })

const adminName = process.env.DOJ_ADMIN_NAME ?? 'admin'
const adminEmail = process.env.DOJ_ADMIN_EMAIL ?? 'admin@example.test'
const adminPassword = process.env.DOJ_ADMIN_PASSWORD ?? 'admin12345'

const [adminGroup] = await db.select().from(schema.groups).where(eq(schema.groups.key, 'admin')).limit(1)
if (!adminGroup) throw new Error('admin group missing after seed')

const [existingAdmin] = await db.select().from(schema.users).where(eq(schema.users.name, adminName)).limit(1)
if (!existingAdmin) {
  const [admin] = await db
    .insert(schema.users)
    .values({
      name: adminName,
      email: adminEmail,
      passwordHash: await Bun.password.hash(adminPassword, {
        algorithm: 'argon2id',
        memoryCost: 19456,
        timeCost: 2
      })
    })
    .returning()

  await db.insert(schema.userGroups).values({ userId: admin.id, groupId: adminGroup.id, manager: true })
}

console.log('Seeded builtin groups and admin user.')

await closeDb()
