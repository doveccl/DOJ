const packages = [
  'packages/shared',
  'packages/db',
  'packages/runner',
  'apps/api',
  'apps/worker',
  'apps/web'
]

for (const cwd of packages) {
  console.log(`\n==> ${cwd}`)
  const result = Bun.spawnSync(['bun', 'run', '--cwd', cwd, 'check'], {
    stdout: 'inherit',
    stderr: 'inherit'
  })
  if (result.exitCode !== 0) process.exit(result.exitCode)
}
