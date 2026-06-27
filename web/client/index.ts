import createClient from 'openapi-fetch'

import type { components, paths } from './schema'

export type Home = components['schemas']['Home']
export type HomeProblem = components['schemas']['HomeProblem']
export type HomeAssignment = components['schemas']['HomeAssignment']
export type HomeContest = components['schemas']['HomeContest']
export type Site = components['schemas']['Site']
export type NoticeUpdate = components['schemas']['NoticeUpdate']
export type CreatedID = components['schemas']['CreatedID']
export type Language = components['schemas']['Lang']
export type UploadResult = components['schemas']['UploadResult']
export type ProblemListItem = components['schemas']['ProblemListItem']
export type ProblemListPage = components['schemas']['ProblemListPage']
export type Problem = components['schemas']['Problem']
export type ProblemAssets = components['schemas']['ProblemAssets']
export type AssetFile = components['schemas']['AssetFile']
export type AssetContent = components['schemas']['AssetContent']
export type Item = components['schemas']['Item']
export type HeatCell = components['schemas']['HeatCell']
export type Assignment = components['schemas']['Assignment']
export type AssignmentListItem = components['schemas']['AssignmentListItem']
export type AssignmentListPage = components['schemas']['AssignmentListPage']
export type ProblemRef = components['schemas']['ProblemRef']
export type AssignmentDetail = components['schemas']['AssignmentDetail']
export type AssignmentProgress = components['schemas']['AssignmentProgress']
export type Contest = components['schemas']['Contest']
export type ContestListPage = components['schemas']['ContestListPage']
export type ContestDetail = components['schemas']['ContestDetail']
export type Submission = components['schemas']['Submission']
export type SubmissionListItem = components['schemas']['SubmissionListItem']
export type SubmissionListPage = components['schemas']['SubmissionListPage']
export type SubmissionDetail = components['schemas']['SubmissionDetail']
export type Case = components['schemas']['Case']
export type RankUser = components['schemas']['RankUser']
export type RankProblem = components['schemas']['RankProblem']
export type UserProfile = components['schemas']['UserProfile']
export type UserActivity = components['schemas']['UserActivity']
export type SolvedProblem = components['schemas']['SolvedProblem']
export type SolvedProblemPage = components['schemas']['SolvedProblemPage']
export type PublicUser = components['schemas']['PublicUser']
export type UserOption = components['schemas']['UserOption']
export type Discussion = components['schemas']['Discussion']
export type DiscussionListPage = components['schemas']['DiscussionListPage']
export type DiscussionDetail = components['schemas']['DiscussionDetail']
export type DiscussionCreate = components['schemas']['DiscussionCreate']
export type Comment = components['schemas']['Comment']
export type CommentCreate = components['schemas']['CommentCreate']
export type Me = components['schemas']['Me']
export type PasswordUpdate = components['schemas']['PasswordUpdate']
export type AdminMembers = components['schemas']['AdminMembers']
export type AdminUserPage = components['schemas']['AdminUserPage']
export type AdminSettings = components['schemas']['AdminSettings']
export type AdminUserCreate = components['schemas']['AdminUserCreate']
export type AdminGroupPage = components['schemas']['AdminGroupPage']
export type AdminGroupUpdate = components['schemas']['AdminGroupUpdate']
export type AdminLang = components['schemas']['AdminLang']
export type AdminLangCreate = components['schemas']['AdminLangCreate']
export type AdminJudgers = components['schemas']['AdminJudgers']
export type BackupSettings = components['schemas']['BackupSettings']
export type BackupList = components['schemas']['BackupList']
export type BackupItem = components['schemas']['BackupItem']

function defaultBaseUrl() {
  return typeof window === 'undefined' ? 'http://localhost:7974' : window.location.origin
}

export const api = createClient<paths>({
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
  const url = apiUrl(path)
  const headers = authHeaders(undefined, init?.headers)
  addCSRFHeader(headers, init?.method ?? 'GET')
  return fetch(url, { ...init, credentials: 'include', headers })
}

export function apiUrl(path: string) {
  return new URL(path, defaultBaseUrl())
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

export async function apiData<T>(response: PromiseLike<{ data?: T; error?: unknown }>) {
  const { data, error } = await response
  return assertData(data, error)
}

export async function apiEmpty(response: PromiseLike<{ error?: unknown }>) {
  const { error } = await response
  if (error) {
    throw new Error(errorMessage(error))
  }
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

export async function downloadBackup(name: string) {
  const response = await apiFetch(`/api/admin/backups/${encodeURIComponent(name)}/download`)
  if (!response.ok) {
    throw new Error(await response.text())
  }
  return response.blob()
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

export async function downloadProblemAssets(id: number) {
  const response = await apiFetch(`/api/problems/${id}.zip`)
  if (!response.ok) {
    throw new Error(await response.text())
  }
  return response.blob()
}

export async function downloadProblemFile(id: number, section: 'data' | 'judge', name: string) {
  const path = name
    .split('/')
    .filter(Boolean)
    .map((part) => encodeURIComponent(part))
    .join('/')
  const response = await apiFetch(`/api/problems/${id}/${section}/${path}`)
  if (!response.ok) {
    throw new Error(await response.text())
  }
  return response.blob()
}
