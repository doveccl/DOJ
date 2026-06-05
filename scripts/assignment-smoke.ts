const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const user = process.env.DOJ_ADMIN_NAME ?? 'admin'
const password = process.env.DOJ_ADMIN_PASSWORD ?? 'admin12345'
const runId = crypto.randomUUID()

async function api(path: string, init: RequestInit = {}) {
  const response = await fetch(`${apiBase}${path}`, init)
  if (!response.ok) {
    throw new Error(`${path} failed: ${response.status} ${await response.text()}`)
  }
  return response.json()
}

const login = (await api('/api/auth/login', {
  method: 'POST',
  headers: {
    'content-type': 'application/json'
  },
  body: JSON.stringify({ user, password })
})) as { token: string }

const headers = {
  'content-type': 'application/json',
  authorization: `Bearer ${login.token}`
}

const group = (await api('/api/groups', {
  method: 'POST',
  headers,
  body: JSON.stringify({
    key: `assignment_${runId.slice(0, 8)}`,
    name: 'Assignment Smoke Group',
    description: 'Created by assignment smoke.'
  })
})) as { id: number }

const { problem } = (await api('/api/problems', {
  method: 'POST',
  headers,
  body: JSON.stringify({
    title: `Assignment Smoke Problem ${runId.slice(0, 8)}`,
    statementMarkdown: '# Assignment Smoke\n\nSolve the assigned task.'
  })
})) as { problem: { id: number } }

const created = (await api('/api/assignments', {
  method: 'POST',
  headers,
  body: JSON.stringify({
    title: `Assignment Smoke ${runId.slice(0, 8)}`,
    description: 'Created by assignment smoke.',
    groupIds: [group.id],
    problems: [{ problemId: problem.id, score: 100 }],
    allowLate: true,
    aiCoachingEnabled: true
  })
})) as {
  assignment: { id: number; title: string }
  groups: Array<{ id: number }>
  problems: Array<{ id: number }>
}

if (created.groups.length !== 1 || created.problems.length !== 1) {
  throw new Error(`assignment detail did not include mappings: ${JSON.stringify(created)}`)
}

const detail = (await api(`/api/assignments/${created.assignment.id}`, {
  headers
})) as typeof created

if (detail.assignment.id !== created.assignment.id) {
  throw new Error(`assignment detail id mismatch: ${detail.assignment.id}`)
}

const list = (await api('/api/assignments', {
  headers
})) as { list: Array<{ id: number }> }

if (!list.list.some((assignment) => assignment.id === created.assignment.id)) {
  throw new Error(`created assignment missing from list: ${created.assignment.id}`)
}

console.log({
  assignmentId: created.assignment.id,
  title: created.assignment.title,
  groups: created.groups.length,
  problems: created.problems.length
})
