import { eq } from 'drizzle-orm'
import { closeDb, db, schema } from '../packages/db/src/client'

const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const adminUser = process.env.DOJ_ADMIN_NAME ?? 'admin'
const adminPassword = process.env.DOJ_ADMIN_PASSWORD ?? 'admin12345'
const runId = crypto.randomUUID()
let crcTable: Uint32Array | undefined

try {
  const [admin, user] = await Promise.all([loginAdmin(), registerUser()])
  const { problem, version } = await createProblem(admin.token)
  const upload = await uploadTestdata(admin.token, problem.id)
  if (upload.caseCount !== 2) throw new Error(`expected 2 cases, got ${upload.caseCount}`)

  const accepted = await submitAndJudge(
    user.token,
    problem.id,
    version.id,
    '#include <bits/stdc++.h>\nusing namespace std;\nint main(){ long long x; cin >> x; cout << x * 3 << "\\n"; }\n'
  )
  if (accepted.status !== 'AC')
    throw new Error(`expected AC, got ${accepted.status}: ${accepted.message}`)

  const wrong = await submitAndJudge(
    user.token,
    problem.id,
    version.id,
    '#include <bits/stdc++.h>\nusing namespace std;\nint main(){ cout << 0 << "\\n"; }\n'
  )
  if (wrong.status !== 'WA') throw new Error(`expected WA, got ${wrong.status}: ${wrong.message}`)

  console.log({
    problemId: problem.id,
    fileId: upload.file.id,
    caseCount: upload.caseCount,
    accepted: accepted.status,
    wrong: wrong.status
  })
} finally {
  await closeDb()
}

async function loginAdmin() {
  const response = await fetch(`${apiBase}/api/auth/login`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ user: adminUser, password: adminPassword })
  })
  if (!response.ok)
    throw new Error(`admin auth API failed: ${response.status} ${await response.text()}`)
  return (await response.json()) as { token: string }
}

async function registerUser() {
  const response = await fetch(`${apiBase}/api/auth/register`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({
      name: `zip_${runId.slice(0, 8)}`,
      email: `zip_${runId}@example.test`,
      password: 'password123'
    })
  })
  if (!response.ok) throw new Error(`auth API failed: ${response.status} ${await response.text()}`)
  return (await response.json()) as { token: string }
}

async function createProblem(token: string) {
  const response = await fetch(`${apiBase}/api/problems`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      authorization: `Bearer ${token}`
    },
    body: JSON.stringify({
      title: `Zip Testdata ${runId.slice(0, 8)}`,
      slug: `zip-testdata-${runId}`,
      statementMarkdown: '# Zip Testdata\n\nTriple the input integer.',
      timeLimitMs: 5000,
      memoryLimitBytes: 128 * 1024 * 1024,
      outputLimitBytes: 1024 * 1024
    })
  })
  if (!response.ok)
    throw new Error(`problem API failed: ${response.status} ${await response.text()}`)
  return (await response.json()) as { problem: { id: number }; version: { id: number } }
}

async function uploadTestdata(token: string, problemId: number) {
  const form = new FormData()
  form.set(
    'file',
    new File(
      [
        createStoreZip({
          'input1.txt': '7\n',
          'ans01.txt': '21\n',
          '2.in': '-4\n',
          'output2.txt': '-12\n'
        })
      ],
      'testdata.zip',
      { type: 'application/zip' }
    )
  )

  const response = await fetch(`${apiBase}/api/problems/${problemId}/testdata`, {
    method: 'POST',
    headers: {
      authorization: `Bearer ${token}`
    },
    body: form
  })
  if (!response.ok)
    throw new Error(`testdata API failed: ${response.status} ${await response.text()}`)
  return (await response.json()) as { caseCount: number; file: { id: number } }
}

async function submitAndJudge(
  token: string,
  problemId: number,
  problemVersionId: number,
  sourceCode: string
) {
  const response = await fetch(`${apiBase}/api/submissions`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      authorization: `Bearer ${token}`
    },
    body: JSON.stringify({
      problemId,
      problemVersionId,
      languageId: 'cpp',
      sourceCode
    })
  })
  if (!response.ok)
    throw new Error(`submission API failed: ${response.status} ${await response.text()}`)
  const submission = (await response.json()) as { id: number }
  return waitForJudgement(submission.id)
}

async function waitForJudgement(submissionId: number) {
  for (let attempt = 0; attempt < 20; attempt += 1) {
    await runWorkerOnce()
    const [judged] = await db
      .select()
      .from(schema.submissions)
      .where(eq(schema.submissions.id, submissionId))
      .limit(1)
    if (!judged) throw new Error(`submission disappeared: ${submissionId}`)
    if (!['WAITING', 'JUDGING'].includes(judged.status)) return judged
    await Bun.sleep(200)
  }
  throw new Error(`submission did not finish judging: ${submissionId}`)
}

async function runWorkerOnce() {
  const worker = Bun.spawn(['bun', 'run', '--cwd', 'apps/worker', 'dev'], {
    env: {
      ...process.env,
      DOJ_WORKER_ONCE: '1'
    },
    stdout: 'pipe',
    stderr: 'pipe'
  })
  const exitCode = await worker.exited
  if (exitCode !== 0) {
    throw new Error(
      [
        `worker failed with exit ${exitCode}`,
        await new Response(worker.stdout).text(),
        await new Response(worker.stderr).text()
      ].join('\n')
    )
  }
}

function createStoreZip(files: Record<string, string>) {
  const encoder = new TextEncoder()
  const chunks: Uint8Array[] = []
  const central: Uint8Array[] = []
  let offset = 0

  for (const [name, content] of Object.entries(files)) {
    const nameBytes = encoder.encode(name)
    const data = encoder.encode(content)
    const crc = crc32(data)
    const local = new Uint8Array(30 + nameBytes.length)
    const view = new DataView(local.buffer)
    view.setUint32(0, 0x04034b50, true)
    view.setUint16(4, 20, true)
    view.setUint16(8, 0, true)
    view.setUint32(14, crc, true)
    view.setUint32(18, data.length, true)
    view.setUint32(22, data.length, true)
    view.setUint16(26, nameBytes.length, true)
    local.set(nameBytes, 30)
    chunks.push(local, data)

    const entry = new Uint8Array(46 + nameBytes.length)
    const entryView = new DataView(entry.buffer)
    entryView.setUint32(0, 0x02014b50, true)
    entryView.setUint16(4, 20, true)
    entryView.setUint16(6, 20, true)
    entryView.setUint32(16, crc, true)
    entryView.setUint32(20, data.length, true)
    entryView.setUint32(24, data.length, true)
    entryView.setUint16(28, nameBytes.length, true)
    entryView.setUint32(42, offset, true)
    entry.set(nameBytes, 46)
    central.push(entry)
    offset += local.length + data.length
  }

  const centralOffset = offset
  const centralSize = central.reduce((sum, item) => sum + item.length, 0)
  const end = new Uint8Array(22)
  const endView = new DataView(end.buffer)
  endView.setUint32(0, 0x06054b50, true)
  endView.setUint16(8, central.length, true)
  endView.setUint16(10, central.length, true)
  endView.setUint32(12, centralSize, true)
  endView.setUint32(16, centralOffset, true)
  chunks.push(...central, end)

  const output = new Uint8Array(chunks.reduce((sum, item) => sum + item.length, 0))
  let cursor = 0
  for (const chunk of chunks) {
    output.set(chunk, cursor)
    cursor += chunk.length
  }
  return output
}

function crc32(bytes: Uint8Array) {
  const table = getCrcTable()
  let crc = 0xffffffff
  for (const byte of bytes) {
    crc = (crc >>> 8) ^ table[(crc ^ byte) & 0xff]
  }
  return (crc ^ 0xffffffff) >>> 0
}

function getCrcTable() {
  crcTable ??= Uint32Array.from({ length: 256 }, (_value, index) => {
    let crc = index
    for (let bit = 0; bit < 8; bit += 1) {
      crc = crc & 1 ? 0xedb88320 ^ (crc >>> 1) : crc >>> 1
    }
    return crc >>> 0
  })
  return crcTable
}
