import createClient from 'openapi-fetch'

import type { components, paths } from './schema'

export type Home = components['schemas']['Home']
export type Site = components['schemas']['Site']
export type NoticeUpdate = components['schemas']['NoticeUpdate']
export type Language = components['schemas']['Lang']
export type UploadResult = components['schemas']['UploadResult']
export type Problem = components['schemas']['Problem']
export type ProblemCreate = components['schemas']['ProblemCreate']
export type ProblemUpdate = components['schemas']['ProblemUpdate']
export type ProblemAssets = components['schemas']['ProblemAssets']
export type AssetFile = components['schemas']['AssetFile']
export type AssetCaseCreate = components['schemas']['AssetCaseCreate']
export type AssetContent = components['schemas']['AssetContent']
export type AssetContentUpdate = components['schemas']['AssetContentUpdate']
export type Item = components['schemas']['Item']
export type HeatCell = components['schemas']['HeatCell']
export type Assignment = components['schemas']['Assignment']
export type AssignmentCreate = components['schemas']['AssignmentCreate']
export type AssignmentUpdate = components['schemas']['AssignmentUpdate']
export type AssignmentDetail = components['schemas']['AssignmentDetail']
export type Contest = components['schemas']['Contest']
export type ContestCreate = components['schemas']['ContestCreate']
export type ContestUpdate = components['schemas']['ContestUpdate']
export type ContestDetail = components['schemas']['ContestDetail']
export type Submission = components['schemas']['Submission']
export type SubmissionDetail = components['schemas']['SubmissionDetail']
export type SubmitRequest = components['schemas']['SubmitRequest']
export type Case = components['schemas']['Case']
export type RankUser = components['schemas']['RankUser']
export type UserProfile = components['schemas']['UserProfile']
export type PublicUser = components['schemas']['PublicUser']
export type Discussion = components['schemas']['Discussion']
export type DiscussionDetail = components['schemas']['DiscussionDetail']
export type DiscussionCreate = components['schemas']['DiscussionCreate']
export type DiscussionUpdate = components['schemas']['DiscussionUpdate']
export type Comment = components['schemas']['Comment']
export type CommentCreate = components['schemas']['CommentCreate']
export type Me = components['schemas']['Me']
export type MeUpdate = components['schemas']['MeUpdate']
export type PasswordUpdate = components['schemas']['PasswordUpdate']
export type LoginRequest = components['schemas']['LoginRequest']
export type RegisterRequest = components['schemas']['RegisterRequest']
export type AdminOverview = components['schemas']['AdminOverview']
export type AdminSettings = components['schemas']['AdminSettings']
export type AdminUserCreate = components['schemas']['AdminUserCreate']
export type AdminUserUpdate = components['schemas']['AdminUserUpdate']
export type PasswordReset = components['schemas']['PasswordReset']
export type AdminGroup = components['schemas']['AdminGroup']
export type AdminGroupUpdate = components['schemas']['AdminGroupUpdate']
export type AdminLangUpdate = components['schemas']['AdminLangUpdate']
export type AdminLangCreate = components['schemas']['AdminLangCreate']
export type AdminJudgerUpdate = components['schemas']['AdminJudgerUpdate']
export type AdminJudgerCreate = components['schemas']['AdminJudgerCreate']

function defaultBaseUrl() {
  const configured = import.meta.env.VITE_API_BASE
  if (configured) {
    return absoluteApiBase(configured)
  }
  if (typeof window === 'undefined') {
    return 'http://localhost:7974'
  }
  return `${window.location.protocol}//${window.location.hostname}:7974`
}

function absoluteApiBase(value: string) {
  if (/^https?:\/\//i.test(value)) {
    return normalizeApiBase(value)
  }
  if (typeof window === 'undefined') {
    return normalizeApiBase(value)
  }
  return normalizeApiBase(new URL(value, window.location.origin).toString().replace(/\/$/, ''))
}

export function normalizeApiBase(value: string) {
  return value.replace(/\/api\/?$/, '')
}

const client = createClient<paths>({
  baseUrl: defaultBaseUrl(),
  fetch: (input: RequestInfo | URL, init?: RequestInit) => {
    const nextInit = init ?? {}
    const headers = authHeaders(input instanceof Request ? input.headers : undefined, nextInit.headers)
    addCSRFHeader(headers, methodOf(input, nextInit))
    if (!headers.has('Content-Type') && hasJsonBody(nextInit.body)) {
      headers.set('Content-Type', 'application/json')
    }
    return fetch(input, { ...nextInit, credentials: 'include', headers })
  }
})

function authHeaders(base?: HeadersInit, extra?: HeadersInit) {
  const headers = new Headers(base)
  new Headers(extra).forEach((value, key) => headers.set(key, value))
  return headers
}

async function apiFetch(path: string, init?: RequestInit) {
  const url = new URL(path, defaultBaseUrl())
  const headers = authHeaders(undefined, init?.headers)
  addCSRFHeader(headers, init?.method ?? 'GET')
  return fetch(url, { ...init, credentials: 'include', headers })
}

function hasJsonBody(body: BodyInit | null | undefined) {
  if (body == null) {
    return false
  }
  if (typeof FormData !== 'undefined' && body instanceof FormData) {
    return false
  }
  if (typeof URLSearchParams !== 'undefined' && body instanceof URLSearchParams) {
    return false
  }
  if (typeof Blob !== 'undefined' && body instanceof Blob) {
    return false
  }
  if (body instanceof ArrayBuffer || ArrayBuffer.isView(body)) {
    return false
  }
  return true
}

function methodOf(input: RequestInfo | URL, init: RequestInit) {
  if (init.method) {
    return init.method
  }
  if (input instanceof Request) {
    return input.method
  }
  return 'GET'
}

function addCSRFHeader(headers: Headers, method: string) {
  if (safeMethod(method) || headers.has('X-DOJ-CSRF')) {
    return
  }
  const token = readCookie('doj_csrf')
  if (token) {
    headers.set('X-DOJ-CSRF', token)
  }
}

function safeMethod(method: string) {
  return ['GET', 'HEAD', 'OPTIONS'].includes(method.toUpperCase())
}

function readCookie(name: string) {
  if (typeof document === 'undefined') {
    return ''
  }
  const prefix = `${name}=`
  const item = document.cookie.split('; ').find((part) => part.startsWith(prefix))
  return item ? decodeURIComponent(item.slice(prefix.length)) : ''
}

function assertData<T>(data: T | undefined, error: unknown): T {
  if (error) {
    throw new Error(errorMessage(error))
  }
  if (data === undefined) {
    throw new Error('Empty response')
  }
  return data
}

function errorMessage(error: unknown) {
  if (error instanceof Error) {
    return error.message
  }
  if (typeof error === 'object' && error !== null && 'message' in error) {
    const message = (error as { message?: unknown }).message
    if (typeof message === 'string') {
      return message
    }
  }
  if (typeof error === 'string') {
    return error
  }
  return JSON.stringify(error)
}

export async function getHome() {
  const { data, error } = await client.GET('/api/home')
  return assertData(data, error)
}

export async function getSite() {
  const { data, error } = await client.GET('/api/site')
  return assertData(data, error)
}

export async function updateNotice(body: NoticeUpdate) {
  const { data, error } = await client.PATCH('/api/home/notice', { body })
  return assertData(data, error)
}

export async function getMe() {
  const { data, error } = await client.GET('/api/me')
  return assertData(data, error)
}

export async function updateMe(body: MeUpdate) {
  const { data, error } = await client.PATCH('/api/me', { body })
  return assertData(data, error)
}

export async function updatePassword(body: PasswordUpdate) {
  const { error } = await client.PATCH('/api/me/password', { body })
  if (error) {
    throw new Error(errorMessage(error))
  }
}

export async function login(body: LoginRequest) {
  const { data, error } = await client.POST('/api/auth/login', { body })
  return assertData(data, error)
}

export async function register(body: RegisterRequest) {
  const { data, error } = await client.POST('/api/auth/register', { body })
  return assertData(data, error)
}

export async function logout() {
  const { error } = await client.POST('/api/auth/logout')
  if (error) {
    throw new Error(errorMessage(error))
  }
}

export async function getLangs() {
  const { data, error } = await client.GET('/api/languages')
  return assertData(data, error)
}

export async function uploadImage(file: File) {
  const body = new FormData()
  body.set('file', file)
  const response = await apiFetch('/api/uploads/images', { method: 'POST', body })
  if (!response.ok) {
    throw new Error(await response.text())
  }
  const data = (await response.json()) as UploadResult
  return data.url
}

export async function uploadProblemImage(id: number, file: File) {
  const body = new FormData()
  body.set('file', file)
  const response = await apiFetch(`/api/problems/${id}/assets/images`, { method: 'POST', body })
  if (!response.ok) {
    throw new Error(await response.text())
  }
  const data = (await response.json()) as UploadResult
  return data.url
}

export async function getAdmin() {
  const { data, error } = await client.GET('/api/admin')
  return assertData(data, error)
}

export async function updateAdminSettings(body: AdminSettings) {
  const { data, error } = await client.PATCH('/api/admin/settings', { body })
  return assertData(data, error)
}

export async function createAdminUser(body: AdminUserCreate) {
  const { data, error } = await client.POST('/api/admin/users', { body })
  return assertData(data, error)
}

export async function updateAdminUser(name: string, body: AdminUserUpdate) {
  const { data, error } = await client.PATCH('/api/admin/users/{name}', {
    params: { path: { name } },
    body
  })
  return assertData(data, error)
}

export async function deleteAdminUser(name: string) {
  const { data, error } = await client.DELETE('/api/admin/users/{name}', {
    params: { path: { name } }
  })
  return assertData(data, error)
}

export async function resetAdminUserPassword(name: string) {
  const { data, error } = await client.POST('/api/admin/users/{name}/password', {
    params: { path: { name } }
  })
  return assertData(data, error)
}

export async function createAdminGroup(body: AdminGroupUpdate) {
  const { data, error } = await client.POST('/api/admin/groups', { body })
  return assertData(data, error)
}

export async function updateAdminGroup(id: number, body: AdminGroupUpdate) {
  const { data, error } = await client.PATCH('/api/admin/groups/{id}', {
    params: { path: { id } },
    body
  })
  return assertData(data, error)
}

export async function deleteAdminGroup(id: number) {
  const { data, error } = await client.DELETE('/api/admin/groups/{id}', {
    params: { path: { id } }
  })
  return assertData(data, error)
}

export async function updateAdminLang(id: string, body: AdminLangUpdate) {
  const { data, error } = await client.PATCH('/api/admin/languages/{id}', {
    params: { path: { id } },
    body
  })
  return assertData(data, error)
}

export async function deleteAdminLang(id: string) {
  const { data, error } = await client.DELETE('/api/admin/languages/{id}', {
    params: { path: { id } }
  })
  return assertData(data, error)
}

export async function createAdminLang(body: AdminLangCreate) {
  const { data, error } = await client.POST('/api/admin/languages', { body })
  return assertData(data, error)
}

export async function updateAdminJudger(id: number, body: AdminJudgerUpdate) {
  const { data, error } = await client.PATCH('/api/admin/judgers/{id}', {
    params: { path: { id } },
    body
  })
  return assertData(data, error)
}

export async function deleteAdminJudger(id: number) {
  const { data, error } = await client.DELETE('/api/admin/judgers/{id}', {
    params: { path: { id } }
  })
  return assertData(data, error)
}

export async function createAdminJudger(body: AdminJudgerCreate) {
  const { data, error } = await client.POST('/api/admin/judgers', { body })
  return assertData(data, error)
}

export async function getProblems(params: { q?: string; tag?: string } = {}) {
  const { data, error } = await client.GET('/api/problems', {
    params: { query: params }
  })
  return assertData(data, error)
}

export async function getProblem(id: number) {
  const { data, error } = await client.GET('/api/problems/{id}', {
    params: { path: { id } }
  })
  return assertData(data, error)
}

export async function createProblem(body: ProblemCreate) {
  const { data, error } = await client.POST('/api/problems', { body })
  return assertData(data, error)
}

export async function updateProblem(id: number, body: ProblemUpdate) {
  const { data, error } = await client.PATCH('/api/problems/{id}', {
    params: { path: { id } },
    body
  })
  return assertData(data, error)
}

export async function deleteProblem(id: number) {
  const { error } = await client.DELETE('/api/problems/{id}', {
    params: { path: { id } }
  })
  if (error) {
    throw new Error(errorMessage(error))
  }
}

export async function getProblemAssets(id: number) {
  const { data, error } = await client.GET('/api/problems/{id}/assets', {
    params: { path: { id } }
  })
  return assertData(data, error)
}

export async function uploadProblemAsset(id: number, section: 'data' | 'judge' | 'assets', file: File) {
  const body = new FormData()
  body.set('section', section)
  body.set('file', file)
  const response = await apiFetch(`/api/problems/${id}/assets/files`, { method: 'POST', body })
  if (!response.ok) {
    throw new Error(await response.text())
  }
  return (await response.json()) as ProblemAssets
}

export async function deleteProblemAsset(id: number, key: string) {
  const { data, error } = await client.DELETE('/api/problems/{id}/assets/files', {
    params: { path: { id }, query: { key } }
  })
  return assertData(data, error)
}

export async function getProblemAssetContent(id: number, key: string) {
  const { data, error } = await client.GET('/api/problems/{id}/assets/files/content', {
    params: { path: { id }, query: { key } }
  })
  return assertData(data, error)
}

export async function updateProblemAssetContent(id: number, body: AssetContentUpdate) {
  const { data, error } = await client.PATCH('/api/problems/{id}/assets/files/content', {
    params: { path: { id } },
    body
  })
  return assertData(data, error)
}

export async function createProblemCase(id: number, body: AssetCaseCreate) {
  const { data, error } = await client.POST('/api/problems/{id}/assets/cases', {
    params: { path: { id } },
    body
  })
  return assertData(data, error)
}

export async function fillJudgeTemplate(id: number) {
  const { data, error } = await client.POST('/api/problems/{id}/assets/template', {
    params: { path: { id } }
  })
  return assertData(data, error)
}

export async function downloadProblemAssets(id: number) {
  const response = await apiFetch(`/api/problems/${id}/assets.zip`)
  if (!response.ok) {
    throw new Error(await response.text())
  }
  return response.blob()
}

export async function getAssignments() {
  const { data, error } = await client.GET('/api/assignments')
  return assertData(data, error)
}

export async function createAssignment(body: AssignmentCreate) {
  const { data, error } = await client.POST('/api/assignments', { body })
  return assertData(data, error)
}

export async function updateAssignment(id: number, body: AssignmentUpdate) {
  const { data, error } = await client.PATCH('/api/assignments/{id}', {
    params: { path: { id } },
    body
  })
  return assertData(data, error)
}

export async function deleteAssignment(id: number) {
  const { error } = await client.DELETE('/api/assignments/{id}', {
    params: { path: { id } }
  })
  if (error) {
    throw new Error(errorMessage(error))
  }
}

export async function getAssignment(id: number) {
  const { data, error } = await client.GET('/api/assignments/{id}', {
    params: { path: { id } }
  })
  return assertData(data, error)
}

export async function getContests() {
  const { data, error } = await client.GET('/api/contests')
  return assertData(data, error)
}

export async function createContest(body: ContestCreate) {
  const { data, error } = await client.POST('/api/contests', { body })
  return assertData(data, error)
}

export async function updateContest(id: number, body: ContestUpdate) {
  const { data, error } = await client.PATCH('/api/contests/{id}', {
    params: { path: { id } },
    body
  })
  return assertData(data, error)
}

export async function deleteContest(id: number) {
  const { error } = await client.DELETE('/api/contests/{id}', {
    params: { path: { id } }
  })
  if (error) {
    throw new Error(errorMessage(error))
  }
}

export async function getContest(id: number) {
  const { data, error } = await client.GET('/api/contests/{id}', {
    params: { path: { id } }
  })
  return assertData(data, error)
}

export async function getSubmissions(params: { problem?: string; status?: string } = {}) {
  const { data, error } = await client.GET('/api/submissions', {
    params: { query: params }
  })
  return assertData(data, error)
}

export async function getSubmission(id: number) {
  const { data, error } = await client.GET('/api/submissions/{id}', {
    params: { path: { id } }
  })
  return assertData(data, error)
}

export async function submitCode(body: SubmitRequest) {
  const { data, error } = await client.POST('/api/submissions', { body })
  return assertData(data, error)
}

export async function getRank() {
  const { data, error } = await client.GET('/api/rank')
  return assertData(data, error)
}

export async function getUser(name: string) {
  const { data, error } = await client.GET('/api/users/{name}', {
    params: { path: { name } }
  })
  return assertData(data, error)
}

export async function getDiscussions(params: { tags?: string } = {}) {
  const { data, error } = await client.GET('/api/discussion', {
    params: { query: params }
  })
  return assertData(data, error)
}

export async function createDiscussion(body: DiscussionCreate) {
  const { data, error } = await client.POST('/api/discussion', { body })
  return assertData(data, error)
}

export async function getDiscussion(id: number) {
  const { data, error } = await client.GET('/api/discussion/{id}', {
    params: { path: { id } }
  })
  return assertData(data, error)
}

export async function updateDiscussion(id: number, body: DiscussionUpdate) {
  const { data, error } = await client.PATCH('/api/discussion/{id}', {
    params: { path: { id } },
    body
  })
  return assertData(data, error)
}

export async function deleteDiscussion(id: number) {
  const { error } = await client.DELETE('/api/discussion/{id}', {
    params: { path: { id } }
  })
  if (error) {
    throw new Error(errorMessage(error))
  }
}

export async function createComment(id: number, body: CommentCreate) {
  const { data, error } = await client.POST('/api/discussion/{id}/comments', {
    params: { path: { id } },
    body
  })
  return assertData(data, error)
}
