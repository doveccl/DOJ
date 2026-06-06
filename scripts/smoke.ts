const smokeTargets = {
  auth: 'scripts/auth-smoke.ts',
  admin: 'scripts/admin-smoke.ts',
  users: 'scripts/users-smoke.ts',
  'problem-update': 'scripts/problem-update-smoke.ts',
  languages: 'scripts/languages-smoke.ts',
  agents: 'scripts/agents-smoke.ts',
  'group-membership': 'scripts/group-membership-smoke.ts',
  assignment: 'scripts/assignment-smoke.ts',
  'my-assignments': 'scripts/my-assignments-smoke.ts',
  contest: 'scripts/contest-smoke.ts',
  discussion: 'scripts/discussion-smoke.ts',
  rank: 'scripts/rank-smoke.ts',
  queue: 'scripts/queue-smoke.ts',
  runner: 'packages/runner/src/smoke.ts',
  'runner-package': 'packages/runner/src/package-smoke.ts',
  e2e: 'scripts/e2e-smoke.ts',
  'submission-security': 'scripts/submission-security-smoke.ts',
  'rate-limit': 'scripts/rate-limit-smoke.ts',
  settings: 'scripts/settings-smoke.ts',
  testcases: 'scripts/testcases-smoke.ts',
  testdata: 'scripts/testdata-zip-smoke.ts',
  ai: 'scripts/ai-coach-smoke.ts'
} as const

const requested = Bun.argv.slice(2)
const targetNames = requested.length ? requested : Object.keys(smokeTargets)
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
