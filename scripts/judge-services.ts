import { eq } from 'drizzle-orm'
import { mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { db, schema } from '../packages/db/src/client'

const spawnedServices: Bun.Subprocess[] = []
const prebuiltJudgeImage = 'doveccl/doj:judger'

export async function ensureJudgeServices() {
  await ensurePrebuiltJudgeImage()
  spawnedServices.push(spawnService(['bun', 'run', '--cwd', 'apps/agent', 'dev']))
  await Bun.sleep(1500)
}

export async function stopSpawnedJudgeServices() {
  for (const service of spawnedServices.splice(0)) {
    service.kill()
    await service.exited.catch(() => {})
  }
}

export async function waitForJudgement(submissionId: number, attempts = 80) {
  for (let attempt = 0; attempt < attempts; attempt += 1) {
    const [judged] = await db
      .select()
      .from(schema.submissions)
      .where(eq(schema.submissions.id, submissionId))
      .limit(1)
    if (!judged) throw new Error(`submission disappeared: ${submissionId}`)
    if (!['WAITING', 'JUDGING'].includes(judged.status)) return judged
    await Bun.sleep(250)
  }
  throw new Error(`submission did not finish judging: ${submissionId}`)
}

function spawnService(cmd: string[]) {
  const service = Bun.spawn(cmd, {
    env: {
      ...process.env
    },
    stdout: 'pipe',
    stderr: 'pipe'
  })
  void drain(service.stdout, cmd.join(' '))
  void drain(service.stderr, cmd.join(' '))
  return service
}

async function ensurePrebuiltJudgeImage() {
  const inspect = Bun.spawnSync(['docker', 'image', 'inspect', prebuiltJudgeImage], {
    stdout: 'ignore',
    stderr: 'ignore'
  })
  if (inspect.exitCode === 0) return

  const dir = await mkdtemp(join(tmpdir(), 'doj-judger-'))
  try {
    await writeFile(
      join(dir, 'Dockerfile'),
      ['FROM busybox:latest', 'COPY judge.sh /judge.sh', 'RUN chmod +x /judge.sh', 'CMD ["/judge.sh"]'].join('\n')
    )
    await writeFile(
      join(dir, 'judge.sh'),
      [
        '#!/bin/sh',
        'cat "$INPUT" &',
        'writer="$!"',
        'exec 1>/dev/null',
        'actual="$(cat)"',
        'wait "$writer"',
        'answer="$(cat "$OUT")"',
        'trimmed_actual="$(printf "%s" "$actual" | sed "s/^[[:space:]]*//;s/[[:space:]]*$//")"',
        'trimmed_answer="$(printf "%s" "$answer" | sed "s/^[[:space:]]*//;s/[[:space:]]*$//")"',
        'if [ "$trimmed_actual" = "$trimmed_answer" ]; then',
        '  if [ "$actual" = "$answer" ] || [ "$CHECK" = "trim" ]; then exit 0; fi',
        '  exit 2',
        'fi',
        'if [ "$CHECK" = "pe" ]; then',
        '  normalized_actual="$(printf "%s" "$actual" | tr -s "[:space:]" " " | sed "s/^ //;s/ $//")"',
        '  normalized_answer="$(printf "%s" "$answer" | tr -s "[:space:]" " " | sed "s/^ //;s/ $//")"',
        '  [ "$normalized_actual" = "$normalized_answer" ] && exit 2',
        'fi',
        'printf "expected %s, got %s\\n" "$answer" "$actual" >&2',
        'exit 1',
        ''
      ].join('\n')
    )
    const build = Bun.spawnSync(['docker', 'build', '-t', prebuiltJudgeImage, dir], {
      stdout: 'inherit',
      stderr: 'inherit'
    })
    if (build.exitCode !== 0) throw new Error(`failed to build ${prebuiltJudgeImage}`)
  } finally {
    await rm(dir, { recursive: true, force: true })
  }
}

async function drain(stream: ReadableStream<Uint8Array> | null, label: string) {
  if (!stream) return
  for await (const chunk of stream) {
    if (process.env.DOJ_SMOKE_LOG_SERVICES === '1') {
      process.stdout.write(`[${label}] ${Buffer.from(chunk).toString('utf8')}`)
    }
  }
}
