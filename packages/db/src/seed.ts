import { closeDb, db, schema } from './client'

const builtinGroups = [
  { key: 'admin', name: 'Admin', description: 'System administrators', builtin: true },
  { key: 'user', name: 'User', description: 'Registered users', builtin: true },
  { key: 'guest', name: 'Guest', description: 'Anonymous or low-trust users', builtin: true }
]

await db
  .insert(schema.groups)
  .values(builtinGroups)
  .onConflictDoNothing({ target: schema.groups.key })

console.log('Seeded builtin groups.')

await closeDb()
