import { DockerRunner } from './docker-runner'

const runner = new DockerRunner()
const scopeId = `smoke-${crypto.randomUUID()}`

try {
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
      'main.sh': '#!/bin/sh\necho "hello from runner"\nsleep 0.3\n'
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
      timeMs: 5_000,
      memoryBytes: 128 * 1024 * 1024,
      outputBytes: 1024 * 1024
    }
  })

  console.log({
    status: result.status,
    exitCode: result.exitCode,
    stdout: Buffer.from(result.stdout).toString('utf8').trim(),
    stderr: result.stderr.trim(),
    timeMs: result.timeMs,
    memoryBytes: result.memoryBytes
  })
} finally {
  await runner.cleanup({ scopeId })
}
