const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const adminUser = process.env.DOJ_ADMIN_NAME ?? 'admin'
const adminPassword = process.env.DOJ_ADMIN_PASSWORD ?? 'admin12345'
const runId = crypto.randomUUID()

async function api(path: string, init: RequestInit = {}) {
  const response = await fetch(`${apiBase}${path}`, init)
  if (!response.ok) {
    throw new Error(`${path} failed: ${response.status} ${await response.text()}`)
  }
  return response.json()
}

const student = (await api('/api/auth/register', {
  method: 'POST',
  headers: {
    'content-type': 'application/json'
  },
  body: JSON.stringify({
    name: `student_${runId.slice(0, 8)}`,
    email: `student_${runId}@example.test`,
    password: 'password123'
  })
})) as { token: string }

const admin = (await api('/api/auth/login', {
  method: 'POST',
  headers: {
    'content-type': 'application/json'
  },
  body: JSON.stringify({ user: adminUser, password: adminPassword })
})) as { token: string }

const adminHeaders = {
  'content-type': 'application/json',
  authorization: `Bearer ${admin.token}`
}

const groups = (await api('/api/groups', {
  headers: adminHeaders
})) as { list: Array<{ id: number; key: string }> }

const userGroup = groups.list.find((group) => group.key === 'user')
if (!userGroup) throw new Error('user group missing')

const { problem } = (await api('/api/problems', {
  method: 'POST',
  headers: {
    'content-type': 'application/json'
  },
  body: JSON.stringify({
    title: `Student Assignment Problem ${runId.slice(0, 8)}`,
    slug: `student-assignment-${runId}`,
    statementMarkdown: '# Student Assignment\n\nVisible to users.'
  })
})) as { problem: { id: number } }

const created = (await api('/api/assignments', {
  method: 'POST',
  headers: adminHeaders,
  body: JSON.stringify({
    title: `Student Assignment ${runId.slice(0, 8)}`,
    groupIds: [userGroup.id],
    problems: [{ problemId: problem.id, score: 100 }]
  })
})) as { assignment: { id: number; title: string } }

const mine = (await api('/api/my/assignments', {
  headers: {
    authorization: `Bearer ${student.token}`
  }
})) as { list: Array<{ id: number; title: string }> }

if (!mine.list.some((assignment) => assignment.id === created.assignment.id)) {
  throw new Error(`student assignment missing: ${created.assignment.id}`)
}

const detail = (await api(`/api/my/assignments/${created.assignment.id}`, {
  headers: {
    authorization: `Bearer ${student.token}`
  }
})) as { problems: Array<{ id: number }> }

if (detail.problems.length !== 1) {
  throw new Error(`student assignment detail missing problems: ${JSON.stringify(detail)}`)
}

const problemDetail = (await api(`/api/problems/${detail.problems[0].id}`)) as {
  version: { id: number }
}

const submission = (await api('/api/submissions', {
  method: 'POST',
  headers: {
    'content-type': 'application/json',
    authorization: `Bearer ${student.token}`
  },
  body: JSON.stringify({
    problemId: detail.problems[0].id,
    problemVersionId: problemDetail.version.id,
    assignmentId: created.assignment.id,
    languageId: 'sh',
    sourceCode: '#!/bin/sh\necho accepted\n'
  })
})) as { assignmentId: number | null }

if (submission.assignmentId !== created.assignment.id) {
  throw new Error(`submission assignment id mismatch: ${submission.assignmentId}`)
}

console.log({
  assignmentId: created.assignment.id,
  title: created.assignment.title,
  visibleCount: mine.list.length
})
