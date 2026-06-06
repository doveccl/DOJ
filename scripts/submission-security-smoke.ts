const apiBase = process.env.DOJ_API_BASE ?? 'http://localhost:7974'
const adminUser = process.env.DOJ_ADMIN_NAME ?? 'admin'
const adminPassword = process.env.DOJ_ADMIN_PASSWORD ?? 'admin12345'
const runId = crypto.randomUUID()

const admin = await login(adminUser, adminPassword)
const user = await registerUser('security')
const intruder = await registerUser('intruder')
const hiddenProblem = await createProblem(admin.token, `Hidden ${runId.slice(0, 8)}`)
const visibleProblem = await createProblem(admin.token, `Visible ${runId.slice(0, 8)}`)

await api(`/api/problems/${hiddenProblem.problem.id}`, {
  method: 'PATCH',
  headers: jsonAuth(admin.token),
  body: JSON.stringify({ visible: false })
})

const group = (await api('/api/groups', {
  method: 'POST',
  headers: jsonAuth(admin.token),
  body: JSON.stringify({
    key: `security_${runId.slice(0, 8)}`,
    name: 'Submission Security Smoke',
    description: 'Created by submission security smoke.'
  })
})) as { id: number }

await api(`/api/groups/${group.id}/users`, {
  method: 'POST',
  headers: jsonAuth(admin.token),
  body: JSON.stringify({ userId: user.user.id, manager: false })
})

const assignment = (await api('/api/assignments', {
  method: 'POST',
  headers: jsonAuth(admin.token),
  body: JSON.stringify({
    title: `Submission Security ${runId.slice(0, 8)}`,
    groupIds: [group.id],
    problems: [{ problemId: visibleProblem.problem.id, score: 100 }],
    allowLate: true
  })
})) as { assignment: { id: number } }

const hiddenStatus = await submitStatus(user.token, {
  problemId: hiddenProblem.problem.id
})
if (hiddenStatus !== 404) {
  throw new Error(`expected hidden problem 404, got ${hiddenStatus}`)
}

const assignmentStatus = await submitStatus(user.token, {
  problemId: visibleProblem.problem.id,
  assignmentId: assignment.assignment.id
})
if (assignmentStatus !== 201) {
  throw new Error(`expected member assignment submission 201, got ${assignmentStatus}`)
}

const nonMemberAssignmentStatus = await submitStatus(intruder.token, {
  problemId: visibleProblem.problem.id,
  assignmentId: assignment.assignment.id
})
if (nonMemberAssignmentStatus !== 404) {
  throw new Error(`expected non-member assignment 404, got ${nonMemberAssignmentStatus}`)
}

const sourceMarker = `LIST_LEAK_${runId}`
const assignmentSubmission = await submit(user.token, {
  problemId: visibleProblem.problem.id,
  assignmentId: assignment.assignment.id,
  sourceCode: `#include <bits/stdc++.h>\n// ASSIGNMENT_${sourceMarker}\nint main(){return 0;}\n`,
  open: true
})
const anonymousAssignmentDetail = await fetch(
  `${apiBase}/api/submissions/${assignmentSubmission.id}`
)
if (anonymousAssignmentDetail.status !== 404) {
  throw new Error(
    `anonymous assignment submission detail leaked: ${anonymousAssignmentDetail.status}`
  )
}
const intruderAssignmentDetail = await fetch(
  `${apiBase}/api/submissions/${assignmentSubmission.id}`,
  {
    headers: { authorization: `Bearer ${intruder.token}` }
  }
)
if (intruderAssignmentDetail.status !== 404) {
  throw new Error(
    `non-member assignment submission detail leaked: ${intruderAssignmentDetail.status}`
  )
}
const ownerAssignmentDetail = (await api(`/api/submissions/${assignmentSubmission.id}`, {
  headers: { authorization: `Bearer ${user.token}` }
})) as { sourceCode: string; restricted: boolean }
if (
  ownerAssignmentDetail.restricted ||
  !ownerAssignmentDetail.sourceCode.includes(`ASSIGNMENT_${sourceMarker}`)
) {
  throw new Error(`owner could not inspect own assignment submission`)
}

const normalSubmission = await submit(user.token, {
  problemId: visibleProblem.problem.id,
  sourceCode: `#include <bits/stdc++.h>\n// ${sourceMarker}\nint main(){return 0;}\n`
})
const privateDetail = (await api(`/api/submissions/${normalSubmission.id}`)) as {
  sourceCode: string
  sourceRestricted: boolean
  restricted: boolean
}
if (privateDetail.restricted || !privateDetail.sourceRestricted || privateDetail.sourceCode) {
  throw new Error(`private source leaked: ${JSON.stringify(privateDetail)}`)
}
const openSubmission = await submit(user.token, {
  problemId: visibleProblem.problem.id,
  sourceCode: `#include <bits/stdc++.h>\n// OPEN_${sourceMarker}\nint main(){return 0;}\n`,
  open: true
})
const openDetail = (await api(`/api/submissions/${openSubmission.id}`)) as {
  sourceCode: string
  sourceRestricted: boolean
}
if (openDetail.sourceRestricted || !openDetail.sourceCode.includes(`OPEN_${sourceMarker}`)) {
  throw new Error(`open source was not public: ${JSON.stringify(openDetail)}`)
}
const submissionList = (await api('/api/submissions')) as {
  list: Array<Record<string, unknown> & { id: number }>
}
const listedSubmission = submissionList.list.find((item) => item.id === normalSubmission.id)
if (!listedSubmission) {
  throw new Error(`created submission missing from public list: ${normalSubmission.id}`)
}
if ('sourceCode' in listedSubmission || JSON.stringify(listedSubmission).includes(sourceMarker)) {
  throw new Error(`submission list leaked source code: ${JSON.stringify(listedSubmission)}`)
}
if (listedSubmission.message) {
  throw new Error(
    `submission list leaked judge message to anonymous viewer: ${listedSubmission.message}`
  )
}
if (submissionList.list.some((item) => item.id === assignmentSubmission.id)) {
  throw new Error(`assignment submission leaked in anonymous list: ${assignmentSubmission.id}`)
}
const intruderList = (await api('/api/submissions', {
  headers: { authorization: `Bearer ${intruder.token}` }
})) as { list: Array<{ id: number }> }
if (intruderList.list.some((item) => item.id === assignmentSubmission.id)) {
  throw new Error(`assignment submission leaked in non-member list: ${assignmentSubmission.id}`)
}
const ownerListSubmission = (
  (await api('/api/submissions', {
    headers: { authorization: `Bearer ${user.token}` }
  })) as { list: Array<{ id: number; message?: string }> }
).list.find((item) => item.id === normalSubmission.id)
if (!ownerListSubmission || typeof ownerListSubmission.message !== 'string') {
  throw new Error(`owner could not read own submission message from list`)
}
const ownerAssignmentListSubmission = (
  (await api('/api/submissions', {
    headers: { authorization: `Bearer ${user.token}` }
  })) as { list: Array<{ id: number }> }
).list.find((item) => item.id === assignmentSubmission.id)
if (!ownerAssignmentListSubmission) {
  throw new Error(`owner could not see own assignment submission in list`)
}
const dashboardBeforeHide = (await api('/api/dashboard', {
  headers: { authorization: `Bearer ${admin.token}` }
})) as {
  stats: { submissions: number }
}

await api(`/api/problems/${visibleProblem.problem.id}`, {
  method: 'PATCH',
  headers: jsonAuth(admin.token),
  body: JSON.stringify({ visible: false })
})
const listAfterHide = (await api('/api/submissions')) as {
  list: Array<{ id: number }>
}
if (listAfterHide.list.some((item) => item.id === normalSubmission.id)) {
  throw new Error(`hidden problem submission leaked in public list: ${normalSubmission.id}`)
}
const dashboardAfterHide = (await api('/api/dashboard', {
  headers: { authorization: `Bearer ${admin.token}` }
})) as {
  stats: { submissions: number }
  recentSubmissions: Array<{ id: number }>
}
if (dashboardAfterHide.recentSubmissions.some((item) => item.id === normalSubmission.id)) {
  throw new Error(`hidden problem submission leaked in dashboard: ${normalSubmission.id}`)
}
if (dashboardAfterHide.stats.submissions !== dashboardBeforeHide.stats.submissions - 4) {
  throw new Error(
    `hidden problem submission counted in dashboard stats: before ${dashboardBeforeHide.stats.submissions}, after ${dashboardAfterHide.stats.submissions}`
  )
}

const hiddenDetailStatus = await fetch(`${apiBase}/api/submissions/${normalSubmission.id}`)
if (hiddenDetailStatus.status !== 404) {
  throw new Error(`hidden problem submission detail leaked: ${hiddenDetailStatus.status}`)
}
const ownerHiddenDetail = (await api(`/api/submissions/${normalSubmission.id}`, {
  headers: { authorization: `Bearer ${user.token}` }
})) as { sourceCode: string; restricted: boolean }
if (ownerHiddenDetail.restricted || !ownerHiddenDetail.sourceCode.includes(sourceMarker)) {
  throw new Error(`owner could not inspect hidden problem submission`)
}

console.log({
  hiddenStatus,
  assignmentStatus,
  nonMemberAssignmentStatus,
  listedSubmissionId: normalSubmission.id,
  privateSourceHidden: true,
  openSourceVisible: true,
  assignmentDetailLeak: false,
  assignmentListLeak: false,
  hiddenListLeak: false,
  hiddenDashboardLeak: false,
  hiddenDetailLeak: false
})

async function login(user: string, password: string) {
  return api('/api/auth/login', {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ user, password })
  }) as Promise<{ token: string }>
}

async function registerUser(prefix: string) {
  return api('/api/auth/register', {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({
      name: `${prefix}_${runId.slice(0, 8)}`,
      email: `${prefix}_${runId}@example.test`,
      password: 'password123'
    })
  }) as Promise<{ token: string; user: { id: number } }>
}

async function createProblem(token: string, title: string) {
  return api('/api/problems', {
    method: 'POST',
    headers: jsonAuth(token),
    body: JSON.stringify({
      title,
      statementMarkdown: '# Security Smoke\n\nReturn zero.',
      testCases: [{ input: '', output: '', hidden: false }]
    })
  }) as Promise<{ problem: { id: number } }>
}

async function submitStatus(token: string, body: { problemId: number; assignmentId?: number }) {
  const response = await fetch(`${apiBase}/api/submissions`, {
    method: 'POST',
    headers: jsonAuth(token),
    body: JSON.stringify({
      ...body,
      languageId: 'cc',
      sourceCode: '#include <bits/stdc++.h>\nint main(){return 0;}\n'
    })
  })
  return response.status
}

async function submit(
  token: string,
  body: { problemId: number; assignmentId?: number; sourceCode: string; open?: boolean }
) {
  return api('/api/submissions', {
    method: 'POST',
    headers: jsonAuth(token),
    body: JSON.stringify({
      ...body,
      languageId: 'cc'
    })
  }) as Promise<{ id: number }>
}

async function api(path: string, init: RequestInit = {}) {
  const response = await fetch(`${apiBase}${path}`, init)
  if (!response.ok) throw new Error(`${path} failed: ${response.status} ${await response.text()}`)
  return response.json()
}

function jsonAuth(token: string) {
  return {
    'content-type': 'application/json',
    authorization: `Bearer ${token}`
  }
}
