export interface RunnerKeyed {
  key: string
}

export function parseRunnerKeyFilter(value = process.env.DOJ_RUNNER_KEYS) {
  const keys =
    value
      ?.split(',')
      .map((key) => key.trim())
      .filter(Boolean) ?? []

  return keys.length ? new Set(keys) : null
}

export function filterRunnerConfigs<T extends RunnerKeyed>(
  runners: T[],
  value = process.env.DOJ_RUNNER_KEYS
) {
  const keys = parseRunnerKeyFilter(value)
  if (!keys) return runners
  return runners.filter((runner) => keys.has(runner.key))
}

export function describeRunnerKeyFilter(value = process.env.DOJ_RUNNER_KEYS) {
  return [...(parseRunnerKeyFilter(value) ?? [])].join(', ')
}
