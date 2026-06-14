const smokeTargets = {
  auth: 'scripts/auth-smoke.ts',
  settings: 'scripts/settings-smoke.ts',
  members: 'scripts/members-smoke.ts',
  'problem-assets': 'scripts/problem-assets-smoke.ts',
  'judge-default': 'scripts/judge-default-smoke.ts',
  'judge-strict': 'scripts/judge-strict-smoke.ts',
  'judge-custom': 'scripts/judge-custom-smoke.ts',
  'submission-security': 'scripts/submission-security-smoke.ts',
  'realtime-progress': 'scripts/realtime-progress-smoke.ts',
  'redis-derived': 'scripts/redis-derived-smoke.ts',
  assignments: 'scripts/assignments-smoke.ts',
  contests: 'scripts/contests-smoke.ts',
  'limits-and-hash': 'scripts/limits-and-hash-smoke.ts',
  discussion: 'scripts/discussion-smoke.ts'
} as const

const defaultSmokeTargets: Array<keyof typeof smokeTargets> = [
  'auth',
  'settings',
  'problem-assets',
  'judge-default',
  'submission-security',
  'assignments',
  'contests',
  'discussion'
]

const requested = Bun.argv.slice(2)
const targetNames = requested.length ? requested : defaultSmokeTargets
const unknown = targetNames.filter((name) => !(name in smokeTargets))

if (unknown.length) {
  console.error(`Unknown smoke target: ${unknown.join(', ')}`)
  console.error(`Available targets: ${Object.keys(smokeTargets).join(', ')}`)
  process.exit(1)
}

for (const name of targetNames) {
  const script = smokeTargets[name as keyof typeof smokeTargets]
  console.log(`\n==> smoke:${name}`)
  const result = Bun.spawnSync(['bun', 'run', script], {
    stdout: 'inherit',
    stderr: 'inherit'
  })
  if (result.exitCode !== 0) process.exit(result.exitCode)
}
