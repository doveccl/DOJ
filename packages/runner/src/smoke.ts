import { DockerRunner } from './docker-runner'
import { judgePayload } from './judge'

const runner = new DockerRunner()

async function runCase(
  name: string,
  script: string,
  expected: string,
  timeMs = 2000,
  outputBytes = 1024 * 1024
) {
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
// CPU-time limit: a busy loop burns CPU and must trip TLE.
await runCase('tle', '#!/bin/sh\nwhile :; do :; done\n', 'TLE', 300)
// Sleeping burns no CPU time, so it is AC under CPU-time judging even though
// wall-clock time exceeds the limit.
await runCase('sleep-ac', '#!/bin/sh\nsleep 1\n', 'AC', 300)
await runCase('ole', '#!/bin/sh\nyes x | head -c 2048\n', 'OLE', 2000, 128)

await runCheckerCase()

// Special judge: a checker that accepts any answer within +/-1 of the expected
// value. The submission prints answer+1, which exact-compare would reject but
// the checker must accept.
async function runCheckerCase() {
  const scopeId = `smoke-checker-${crypto.randomUUID()}`
  const result = await judgePayload(runner, {
    submissionId: 0,
    scopeId,
    sourceCode:
      '#include <cstdio>\nint main(){int a,b;scanf("%d %d",&a,&b);printf("%d\\n",a+b+1);}',
    language: {
      id: 'cc',
      sourceFile: 'main.cc',
      dockerfile: [
        'FROM gcc:latest',
        'WORKDIR /workspace',
        'COPY main.cc /workspace/main.cc',
        'RUN g++ -std=c++20 -O2 -pipe -o main main.cc',
        'CMD ["/workspace/main"]'
      ].join('\n'),
      command: ['/workspace/main']
    },
    limits: { timeMs: 2000, memoryBytes: 128 * 1024 * 1024, outputBytes: 1024 * 1024 },
    testCases: [{ name: '1', input: '1 2\n', output: '3\n' }],
    checker: {
      sourceCode: [
        '#include <cstdio>',
        '#include <cstdlib>',
        'int main(int argc, char** argv){',
        '  FILE* fo = fopen(argv[2], "r");',
        '  FILE* fa = fopen(argv[3], "r");',
        '  long out, ans;',
        '  if (fscanf(fo, "%ld", &out) != 1) return 1;',
        '  if (fscanf(fa, "%ld", &ans) != 1) return 1;',
        '  long d = out - ans; if (d < 0) d = -d;',
        '  return d <= 1 ? 0 : 1;',
        '}'
      ].join('\n')
    }
  })
  await runner.cleanup({ scopeId })
  if (result.status !== 'AC') {
    throw new Error(`case checker: expected AC, got ${result.status} (${result.message})`)
  }
  console.log({ name: 'checker', status: result.status, message: result.message })
}
