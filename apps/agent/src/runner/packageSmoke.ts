import { DockerRunner } from './dockerRunner'

const runner = new DockerRunner()

const cacheKey = `smoke-${crypto.randomUUID().replace(/-/g, '')}`
try {
  const files = {
    Dockerfile: 'FROM busybox:latest\nCMD ["true"]\n'
  }
  const firstBuild = await runner.buildPackage({
    scopeId: `pkg-cache-a-${crypto.randomUUID()}`,
    files,
    trusted: true,
    cacheKey
  })
  const secondBuild = await runner.buildPackage({
    scopeId: `pkg-cache-b-${crypto.randomUUID()}`,
    files,
    trusted: true,
    cacheKey
  })
  if (!firstBuild.ok || firstBuild.cached || !secondBuild.ok || !secondBuild.cached) {
    throw new Error(
      `expected package image cache miss then hit, got ${JSON.stringify({ firstBuild, secondBuild })}`
    )
  }
  console.log({
    name: 'package-cache',
    firstCached: firstBuild.cached,
    secondCached: secondBuild.cached
  })
} finally {
  await runner.cleanupPackageCache(cacheKey)
}

console.log('package engine smoke passed')
