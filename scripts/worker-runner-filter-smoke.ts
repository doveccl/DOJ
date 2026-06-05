import { describeRunnerKeyFilter, filterRunnerConfigs } from '../apps/worker/src/runner-filter'

const runners = [
  { key: 'local-docker', value: 1 },
  { key: 'judge-a', value: 2 },
  { key: 'judge-b', value: 3 }
]

const all = filterRunnerConfigs(runners, undefined).map((runner) => runner.key)
if (all.join(',') !== 'local-docker,judge-a,judge-b') {
  throw new Error(`empty filter should keep all runners: ${all.join(',')}`)
}

const selected = filterRunnerConfigs(runners, ' judge-a, judge-b ,, ').map((runner) => runner.key)
if (selected.join(',') !== 'judge-a,judge-b') {
  throw new Error(`runner key filter mismatch: ${selected.join(',')}`)
}

const missing = filterRunnerConfigs(runners, 'missing')
if (missing.length) {
  throw new Error(`missing filter should not match runners: ${JSON.stringify(missing)}`)
}

const described = describeRunnerKeyFilter(' judge-a, judge-b ,, ')
if (described !== 'judge-a, judge-b') {
  throw new Error(`filter description mismatch: ${described}`)
}

console.log({
  all,
  selected,
  missing: missing.length,
  described
})
