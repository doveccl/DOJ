import { closeDb } from '../packages/db/src/client'
import { redisSetJson } from '../apps/server/src/redis'
import { ensureJudgeServices, stopSpawnedJudgeServices, waitForJudgement } from './judge-services'
import tar from 'tar-stream'

const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const adminUser = process.env.ADMIN_NAME ?? 'admin'
const adminPassword = process.env.ADMIN_PASSWORD ?? 'admin12345'
const password = 'password123'
const smokeClientIp = `10.255.${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 255)}`

type SmokeName =
  | 'auth'
  | 'settings'
  | 'members'
  | 'problem-assets'
  | 'judge-default'
  | 'judge-strict'
  | 'judge-custom'
  | 'submission-security'
  | 'realtime-progress'
  | 'redis-derived'
  | 'assignments'
  | 'contests'
  | 'limits-and-hash'
  | 'discussion'

type Auth = { token: string; user: { id: number; name: string; email: string; mustChangePassword?: boolean } }
type Problem = { id: number; statement: string }

const name = (Bun.argv[2] as SmokeName | undefined) ?? inferSmokeName(Bun.argv[1])
if (!name) throw new Error('missing smoke name')

try {
  await run(name)
} finally {
  await closeDb()
}

async function run(target: SmokeName) {
  if (target === 'auth') return authSmoke()
  if (target === 'settings') return settingsSmoke()
  if (target === 'members') return membersSmoke()
  if (target === 'problem-assets') return problemAssetsSmoke()
  if (target === 'discussion') return discussionSmoke()
  if (target === 'assignments') return assignmentsSmoke()
  if (target === 'contests') return contestsSmoke()
  if (target === 'submission-security') return submissionSecuritySmoke()
  if (target === 'redis-derived') return redisDerivedSmoke()
  if (target === 'limits-and-hash') return limitsAndHashSmoke()
  return judgeSmoke(target)
}

async function authSmoke() {
  const admin = await loginAdmin()
  await api('/api/admin/settings', {
    token: admin.token,
    method: 'PATCH',
    body: { general: { signup: false } }
  })
  await expectStatus('/api/auth/register', 403, {
    method: 'POST',
    body: {
      name: `closed_${shortId()}`,
      email: `closed_${shortId()}@example.test`,
      password,
      code: '000000'
    }
  })
  await enableSignup(admin.token)
  const user = await registerUser('auth')
  const nextEmail = `auth_next_${shortId()}@example.test`
  const changed = await api<{ email: string }>('/api/auth/email', {
    token: user.token,
    method: 'PATCH',
    body: { email: nextEmail }
  })
  if (changed.email !== nextEmail) throw new Error(`email update failed: ${JSON.stringify(changed)}`)
  await resetSmokeSettings(admin.token)
  console.log({ smoke: 'auth', admin: admin.user.name, user: user.user.id })
}

async function settingsSmoke() {
  const admin = await loginAdmin()
  const settings = await api<any>('/api/admin/settings', {
    token: admin.token,
    method: 'PATCH',
    body: {
      general: { publicCode: true, guestAccess: false },
      smtp: { enabled: true, _host: 'smtp.example.test', _password: 'secret', from: 'noreply@example.test' },
      ai: { enabled: true, _baseUrl: 'https://ai.example.test', _apiKey: 'secret' }
    }
  })
  if (settings.smtp.hostSet !== true || settings.smtp.passwordSet !== true || settings.ai.apiKeySet !== true) {
    throw new Error(`private settings leaked: ${JSON.stringify(settings)}`)
  }
  const config = await api<any>('/api/config')
  if (config.publicCode !== true || config.guestAccess !== false) {
    throw new Error(`public config mismatch: ${JSON.stringify(config)}`)
  }
  await resetSmokeSettings(admin.token)
  console.log({ smoke: 'settings', redacted: true })
}

async function membersSmoke() {
  const [admin, user] = await Promise.all([loginAdmin(), registerUser('member')])
  const group = await api<{ id: number }>('/api/admin/groups', {
    token: admin.token,
    method: 'POST',
    body: { name: `Smoke Group ${shortId()}` }
  })
  await api(`/api/admin/groups/${group.id}/users`, {
    token: admin.token,
    method: 'POST',
    body: { userId: user.user.id }
  })
  const members = await api<any>(`/api/admin/groups/${group.id}/users`, { token: admin.token })
  if (!members.items.some((item: { id: number }) => item.id === user.user.id)) {
    throw new Error(`member missing: ${JSON.stringify(members)}`)
  }
  console.log({ smoke: 'members', group: group.id, user: user.user.id })
}

async function problemAssetsSmoke() {
  const admin = await loginAdmin()
  await resetSmokeSettings(admin.token)
  const problem = await createProblem(admin.token, {
    title: `Problem Assets ${shortId()}`,
    mode: 'custom',
    statement: '# Asset Smoke\n\nStatement lives in S3.',
    assets: {
      'data/in1.txt': '2 5\n',
      'data/ans1.txt': '7\n',
      'assets/sample.txt': 'public asset\n',
      Dockerfile: 'FROM busybox:latest\nCOPY src/checker.sh /checker.sh\nCMD ["/checker.sh"]\n',
      'src/checker.sh': '#!/bin/sh\nexit 0\n'
    }
  })
  if (problem.statement !== '# Asset Smoke\n\nStatement lives in S3.') throw new Error('statement mismatch')
  const assets = await api<Array<{ path: string }>>(`/api/admin/problems/${problem.id}/assets`, {
    token: admin.token
  })
  for (const path of ['data/in1.txt', 'data/ans1.txt', 'assets/sample.txt', 'Dockerfile', 'src/checker.sh']) {
    if (!assets.some((asset) => asset.path === path)) throw new Error(`missing asset: ${path}`)
  }

  const publicAsset = await fetch(`${apiBase}/api/problems/${problem.id}/assets/sample.txt`)
  if (!publicAsset.ok || (await publicAsset.text()) !== 'public asset\n') {
    throw new Error(`public asset mismatch: ${publicAsset.status}`)
  }
  await expectStatus(`/api/problems/${problem.id}/assets/data%2Fin1.txt`, 404)
  await expectStatus(`/api/admin/problems/${problem.id}/assets`, 401)
  await expectStatus(`/api/agents/bundle/${problem.id}`, 401)

  const bundle = await fetch(`${apiBase}/api/agents/bundle/${problem.id}`, {
    headers: { authorization: `Bearer ${process.env.SECRET ?? 'local-dev-secret-change-me'}` }
  })
  const bundleHash = bundle.headers.get('x-bundle-hash')
  if (!bundle.ok || !bundleHash?.match(/^[a-f0-9]{64}$/)) throw new Error(`bad bundle: ${bundle.status}`)
  const bundlePaths = await readTarPaths(new Uint8Array(await bundle.arrayBuffer()))
  for (const path of ['data/in1.txt', 'data/ans1.txt', 'Dockerfile', 'src/checker.sh']) {
    if (!bundlePaths.includes(path)) throw new Error(`bundle missing ${path}: ${bundlePaths.join(', ')}`)
  }
  for (const path of ['statement.md', 'assets/sample.txt']) {
    if (bundlePaths.includes(path)) throw new Error(`bundle leaked ${path}`)
  }

  await api(`/api/admin/problems/${problem.id}`, {
    token: admin.token,
    method: 'PATCH',
    body: { visible: false }
  })
  await expectStatus(`/api/problems/${problem.id}/assets/sample.txt`, 404)
  await resetSmokeSettings(admin.token)
  console.log({ smoke: 'problem-assets', problem: problem.id, assets: assets.length, bundleHash })
}

async function judgeSmoke(target: SmokeName) {
  await ensureJudgeServices()
  try {
    const [admin, user] = await Promise.all([loginAdmin(), registerUser(target.replace('-', '_'))])
    const mode = target === 'judge-strict' ? 'strict' : target === 'judge-custom' ? 'custom' : 'default'
    const assets =
      mode === 'custom'
        ? {
            'data/in1.txt': '',
            'data/ans1.txt': '',
            Dockerfile: 'FROM busybox:latest\nCOPY judge.sh /judge.sh\nRUN chmod +x /judge.sh\nCMD ["/judge.sh"]\n',
            'judge.sh': '#!/bin/sh\necho 6\nread ans\n[ "$ans" = "6" ] && exit 0\necho wrong >&2\nexit 1\n'
          }
        : {
            'data/in1.txt': mode === 'strict' ? '' : '1 2\n',
            'data/ans1.txt': mode === 'strict' ? 'hello world\n' : '3\n'
          }
    const problem = await createProblem(admin.token, {
      title: `${target} ${shortId()}`,
      mode,
      assets
    })
    const code =
      mode === 'custom'
        ? '#include <bits/stdc++.h>\nusing namespace std; int main(){ long long x; if(cin>>x) cout << x << "\\n"; }'
        : mode === 'strict'
          ? '#include <bits/stdc++.h>\nusing namespace std; int main(){ cout << "hello    world\\n"; }'
          : '#include <bits/stdc++.h>\nusing namespace std; int main(){ long long a,b; cin>>a>>b; cout << a+b << "\\n"; }'
    const submission = await submit(user.token, problem.id, code)
    const judged = await waitForJudgement(submission.id)
    const expected = mode === 'strict' ? 'PE' : 'AC'
    if (judged.status !== expected) throw new Error(`expected ${expected}, got ${judged.status}: ${judged.message}`)
    console.log({ smoke: target, problem: problem.id, submission: submission.id, status: judged.status })
  } finally {
    await stopSpawnedJudgeServices()
  }
}

async function submissionSecuritySmoke() {
  await ensureJudgeServices()
  try {
    const [admin, owner, stranger] = await Promise.all([
      loginAdmin(),
      registerUser('sec_owner'),
      registerUser('sec_other')
    ])
    const problem = await createProblem(admin.token, {
      title: `Submission Security ${shortId()}`,
      assets: { 'data/in1.txt': '', 'data/ans1.txt': 'ok\n' }
    })
    const privateSubmission = await submit(
      owner.token,
      problem.id,
      '#include <bits/stdc++.h>\nusing namespace std; int main(){ cout << "ok\\n"; }',
      { public: false }
    )
    await waitForJudgement(privateSubmission.id)
    const hidden = await api<{ code: string | null }>(`/api/submissions/${privateSubmission.id}`, {
      token: stranger.token
    })
    if (hidden.code !== null) throw new Error('private source leaked')
    console.log({ smoke: 'submission-security', submission: privateSubmission.id })
  } finally {
    await stopSpawnedJudgeServices()
  }
}

async function redisDerivedSmoke() {
  await ensureJudgeServices()
  try {
    const [admin, user] = await Promise.all([loginAdmin(), registerUser('redis')])
    const problem = await createProblem(admin.token, {
      title: `Redis Derived ${shortId()}`,
      assets: { 'data/in1.txt': '', 'data/ans1.txt': 'ok\n' }
    })
    const submission = await submit(user.token, problem.id, '#include <bits/stdc++.h>\nusing namespace std; int main(){ cout << "ok\\n"; }')
    await waitForJudgement(submission.id)
    const repair = await api<any>('/api/admin/stats/repair', { token: admin.token, method: 'POST' })
    if (repair.status !== 'DONE') throw new Error(`stats repair failed: ${JSON.stringify(repair)}`)
    const ranking = await api<any>('/api/ranking?pageSize=100')
    if (!ranking.items.some((row: any) => row.user.id === user.user.id && row.solved >= 1)) {
      throw new Error(`ranking missing solved user: ${JSON.stringify(ranking)}`)
    }
    console.log({ smoke: 'redis-derived', user: user.user.id })
  } finally {
    await stopSpawnedJudgeServices()
  }
}

async function limitsAndHashSmoke() {
  await ensureJudgeServices()
  try {
    const [admin, user] = await Promise.all([loginAdmin(), registerUser('limits')])
    const problem = await createProblem(admin.token, {
      title: `Limits Hash ${shortId()}`,
      timeLimit: 500,
      assets: { 'data/in1.txt': '', 'data/ans1.txt': 'ok\n' }
    })
    const bundle = await fetch(`${apiBase}/api/agents/bundle/${problem.id}`, {
      headers: { authorization: `Bearer ${process.env.SECRET ?? 'local-dev-secret-change-me'}` }
    })
    const bundleHash = bundle.headers.get('x-bundle-hash')
    if (!bundle.ok || !bundleHash?.match(/^[a-f0-9]{64}$/)) throw new Error(`bad bundle hash: ${bundle.status}`)
    const submission = await submit(user.token, problem.id, '#include <bits/stdc++.h>\nusing namespace std; int main(){ while(true){} }')
    const judged = await waitForJudgement(submission.id)
    if (judged.status !== 'TLE') throw new Error(`expected TLE, got ${judged.status}`)
    console.log({ smoke: 'limits-and-hash', bundleHash, status: judged.status })
  } finally {
    await stopSpawnedJudgeServices()
  }
}

async function assignmentsSmoke() {
  const [admin, user] = await Promise.all([loginAdmin(), registerUser('assign')])
  const problem = await createProblem(admin.token, {
    title: `Assignment Problem ${shortId()}`,
    assets: { 'data/in1.txt': '', 'data/ans1.txt': 'ok\n' }
  })
  const assignment = await api<any>('/api/admin/assignments', {
    token: admin.token,
    method: 'POST',
    body: {
      title: `Assignment ${shortId()}`,
      endAt: new Date(Date.now() + 86400000).toISOString(),
      userIds: [user.user.id],
      problemIds: [problem.id]
    }
  })
  const mine = await api<any>('/api/my/assignments', { token: user.token })
  if (!mine.items.some((item: any) => item.id === assignment.assignment.id)) throw new Error('assignment missing')
  console.log({ smoke: 'assignments', assignment: assignment.assignment.id })
}

async function contestsSmoke() {
  const [admin, user] = await Promise.all([loginAdmin(), registerUser('contest')])
  const problem = await createProblem(admin.token, {
    title: `Contest Problem ${shortId()}`,
    assets: { 'data/in1.txt': '', 'data/ans1.txt': 'ok\n' }
  })
  const now = Date.now()
  const contest = await api<any>('/api/admin/contests', {
    token: admin.token,
    method: 'POST',
    body: {
      title: `ICPC ${shortId()}`,
      type: 'ICPC',
      startAt: new Date(now - 3600000).toISOString(),
      freezeAt: new Date(now - 60000).toISOString(),
      endAt: new Date(now + 3600000).toISOString(),
      problemIds: [problem.id]
    }
  })
  await submit(user.token, problem.id, '#include <bits/stdc++.h>\nusing namespace std; int main(){ cout << "ok\\n"; }', {
    contestId: contest.contest.id
  })
  const board = await api<any>(`/api/contests/${contest.contest.id}/scoreboard`)
  const full = await api<any>(`/api/admin/contests/${contest.contest.id}/scoreboard/full`, {
    token: admin.token
  })
  if (board.contest.mode !== 'public' || full.contest.mode !== 'full') throw new Error('scoreboard modes mismatch')
  console.log({ smoke: 'contests', contest: contest.contest.id })
}

async function discussionSmoke() {
  const user = await registerUser('discuss')
  const topic = await api<any>('/api/discussion/topics', {
    token: user.token,
    method: 'POST',
    body: { title: `Discussion ${shortId()}`, content: 'Smoke topic body.', tags: ['smoke'] }
  })
  const reply = await api<any>(`/api/discussion/topics/${topic.id}/posts`, {
    token: user.token,
    method: 'POST',
    body: { content: 'Smoke reply.' }
  })
  const detail = await api<any>(`/api/discussion/topics/${topic.id}`)
  if (!detail.posts.some((post: any) => post.id === reply.id)) throw new Error('reply missing')
  console.log({ smoke: 'discussion', topic: topic.id, reply: reply.id })
}

async function loginAdmin() {
  const admin = await api<Auth>('/api/auth/login', {
    method: 'POST',
    body: { user: adminUser, password: adminPassword }
  })
  if (admin.user.mustChangePassword) {
    await api('/api/auth/self', {
      token: admin.token,
      method: 'PATCH',
      body: { currentPassword: adminPassword, password: adminPassword }
    })
  }
  return admin
}

async function registerUser(prefix: string) {
  const admin = await loginAdmin()
  await enableSignup(admin.token)
  const userName = `${prefix}_${shortId()}`.slice(0, 32)
  const email = `${userName}@example.test`
  await setEmailCode('register', email, '123456', null)
  return api<Auth>('/api/auth/register', {
    method: 'POST',
    body: { name: userName, email, password, code: '123456' }
  })
}

async function enableSignup(token: string) {
  return api('/api/admin/settings', {
    token,
    method: 'PATCH',
    body: {
      general: { signup: true, guestAccess: true },
      smtp: { enabled: true, _host: 'smtp.example.test', _port: 587, from: 'noreply@example.test' }
    }
  })
}

async function resetSmokeSettings(token: string) {
  return api('/api/admin/settings', {
    token,
    method: 'PATCH',
    body: {
      general: { signup: false, guestAccess: true, publicCode: false },
      smtp: { enabled: false, _host: '', _port: 587, _user: '', _password: '', from: '' },
      ai: { enabled: false, _baseUrl: '', _model: '', _apiKey: '' }
    }
  })
}

async function createProblem(token: string, input: {
  title: string
  statement?: string
  mode?: string
  timeLimit?: number
  assets: Record<string, string>
}) {
  const problem = await api<Problem>('/api/admin/problems', {
    token,
    method: 'POST',
    body: {
      title: input.title,
      mode: input.mode ?? 'default',
      visible: true,
      timeLimit: input.timeLimit ?? 2000,
      tags: ['smoke']
    }
  })
  await api(`/api/admin/problems/${problem.id}/statement`, {
    token,
    method: 'PUT',
    body: { markdown: input.statement ?? `# ${input.title}\n` }
  })
  for (const [path, content] of Object.entries(input.assets)) {
    await api(`/api/admin/problems/${problem.id}/assets/content`, {
      token,
      method: 'PUT',
      body: { path, content, encoding: 'utf8', contentType: 'text/plain; charset=utf-8' }
    })
  }
  return api<Problem>(`/api/admin/problems/${problem.id}`, { token })
}

async function submit(token: string, problemId: number, code: string, extra: Record<string, unknown> = {}) {
  return api<{ id: number }>('/api/submissions', {
    token,
    method: 'POST',
    body: { problemId, languageId: 'cpp', code, ...extra }
  })
}

async function readTarPaths(bytes: Uint8Array) {
  const extract = tar.extract()
  const paths: string[] = []
  const done = new Promise<void>((resolve, reject) => {
    extract.on('entry', (header, stream, next) => {
      paths.push(header.name)
      stream.resume()
      stream.on('end', next)
    })
    extract.on('finish', resolve)
    extract.on('error', reject)
  })
  extract.end(Buffer.from(bytes))
  await done
  return paths
}

async function api<T>(path: string, options: { token?: string; method?: string; body?: unknown } = {}) {
  const headers: Record<string, string> = { 'x-forwarded-for': smokeClientIp }
  if (options.token) headers.authorization = `Bearer ${options.token}`
  if (options.body !== undefined) headers['content-type'] = 'application/json'
  const response = await fetch(`${apiBase}${path}`, {
    method: options.method ?? 'GET',
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body)
  })
  if (!response.ok) throw new Error(`${path} failed: ${response.status} ${await response.text()}`)
  return (await response.json()) as T
}

async function expectStatus(path: string, status: number, options: { method?: string; body?: unknown } = {}) {
  const headers: Record<string, string> = { 'x-forwarded-for': smokeClientIp }
  if (options.body !== undefined) headers['content-type'] = 'application/json'
  const response = await fetch(`${apiBase}${path}`, {
    method: options.method ?? 'GET',
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body)
  })
  if (response.status !== status) throw new Error(`${path} expected ${status}, got ${response.status}`)
}

async function setEmailCode(purpose: 'register', email: string, code: string, userId: number | null) {
  await redisSetJson(
    `emailCode:${purpose}:${email.toLowerCase()}`,
    { code, userId, newEmail: email.toLowerCase(), createdAt: new Date().toISOString() },
    600
  )
}

function shortId() {
  return crypto.randomUUID().slice(0, 8)
}

function inferSmokeName(scriptPath: string | undefined) {
  const basename = scriptPath?.split('/').at(-1) ?? ''
  const match = basename.match(/^(.+)-smoke\.ts$/)
  return match?.[1] as SmokeName | undefined
}
