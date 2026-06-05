import { eq } from 'drizzle-orm'
import { db, schema } from '../packages/db/src/client'

const spawnedServices: Bun.Subprocess[] = []

export async function ensureJudgeServices() {
  if (!(await workerHealthOk())) {
    spawnedServices.push(spawnService(['bun', 'run', '--cwd', 'apps/worker', 'dev']))
    await waitForWorkerHealth()
  }

  if (!(await workerHasLocalAgent())) {
    spawnedServices.push(spawnService(['bun', 'run', '--cwd', 'apps/agent', 'dev']))
    if (!(await waitForWorkerLocalAgent())) {
      throw new Error('local judge agent did not connect to worker')
    }
  }
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

async function drain(stream: ReadableStream<Uint8Array> | null, label: string) {
  if (!stream) return
  for await (const chunk of stream) {
    if (process.env.DOJ_SMOKE_LOG_SERVICES === '1') {
      process.stdout.write(`[${label}] ${Buffer.from(chunk).toString('utf8')}`)
    }
  }
}

async function workerHealthOk() {
  return (await readWorkerHealth())?.ok === true
}

async function readWorkerHealth() {
  try {
    const response = await fetch('http://localhost:7975/health')
    if (!response.ok) return null
    return (await response.json()) as {
      ok: boolean
      agents?: Array<{ key: string }>
    }
  } catch {
    return null
  }
}

async function waitForWorkerHealth() {
  for (let attempt = 0; attempt < 40; attempt += 1) {
    if (await workerHealthOk()) return
    await Bun.sleep(250)
  }
  throw new Error('worker agent server did not become healthy')
}

async function workerHasLocalAgent() {
  return (await readWorkerHealth())?.agents?.some((agent) => agent.key === 'local-agent') === true
}

async function waitForWorkerLocalAgent() {
  for (let attempt = 0; attempt < 32; attempt += 1) {
    if (await workerHasLocalAgent()) return true
    await Bun.sleep(250)
  }
  return false
}
