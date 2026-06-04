import { DockerRunner } from './docker-runner'

const runner = new DockerRunner()

async function runCase(name: string, script: string, expected: string, timeMs = 2000, outputBytes = 1024 * 1024) {
  const scopeId = `smoke-${name}-${crypto.randomUUID()}`
  const build = await runner.build({
    scopeId,
    dockerfile: [
      'FROM alpine:3.20',
      'WORKDIR /work',
      'COPY main.sh /work/main.sh',
      'RUN chmod +x /work/main.sh',
      'CMD ["/work/main.sh"]'
    ].join('\n'),
    files: {
      'main.sh': script
    },
    limits: {
      timeMs: 10_000,
      memoryBytes: 128 * 1024 * 1024
    }
  })

  if (!build.ok || !build.imageId) {
    throw new Error(`build failed:\n${build.logs}`)
  }

  const result = await runner.run({
    scopeId,
    imageId: build.imageId,
    limits: {
      timeMs,
      memoryBytes: 128 * 1024 * 1024,
      outputBytes
    }
  })

  await runner.cleanup({ scopeId })

  if (result.status !== expected) {
    throw new Error(`case ${name}: expected ${expected}, got ${result.status}`)
  }

  console.log({
    name,
    status: result.status,
    exitCode: result.exitCode,
    stdout: Buffer.from(result.stdout).toString('utf8').trim(),
    stderr: result.stderr.trim(),
    timeMs: result.timeMs,
    memoryBytes: result.memoryBytes
  })
}

await runCase('ac', '#!/bin/sh\necho "hello from runner"\nsleep 0.3\n', 'AC')
await runCase('re', '#!/bin/sh\necho "runtime boom" >&2\nexit 42\n', 'RE')
await runCase('tle', '#!/bin/sh\nsleep 2\n', 'TLE', 300)
await runCase('ole', '#!/bin/sh\nyes x | head -c 2048\n', 'OLE', 2000, 128)
