export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly body: unknown
  ) {
    super(message)
  }
}

export const DEFAULT_PAGE_SIZE = 50
export const PAGE_SIZE_OPTIONS = [20, 50, 100] as const

export interface PublicConfig {
  signup: boolean
  guestAccess: boolean
  publicCode: boolean
  smtpConfigured: boolean
  aiEnabled: boolean
  notice: string
}

export async function apiFetch<T>(path: string, init: RequestInit = {}) {
  const token = localStorage.getItem('doj.token')
  const headers = new Headers(init.headers)
  if (!headers.has('content-type') && init.body && !(init.body instanceof FormData)) {
    headers.set('content-type', 'application/json')
  }
  if (token) headers.set('authorization', `Bearer ${token}`)

  const response = await fetch(path, {
    ...init,
    headers
  })

  const text = await response.text()
  const body = text ? parseResponseBody(text) : null
  if (!response.ok) {
    const message =
      typeof body?.error?.message === 'string'
        ? body.error.message
        : typeof body?.message === 'string'
          ? body.message
          : `${response.status} ${response.statusText}`
    throw new ApiError(message, response.status, body)
  }

  return body as T
}

function parseResponseBody(text: string) {
  try {
    return JSON.parse(text)
  } catch {
    return { message: text }
  }
}

export interface Paged<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
}

export function getItems<T>(data: Paged<T>) {
  return data.items
}

export function errorMessage(cause: unknown) {
  return cause instanceof Error ? cause.message : String(cause)
}

export function isUnauthorized(cause: unknown) {
  return cause instanceof ApiError && cause.status === 401
}

export function openApiWebSocket() {
  const token = localStorage.getItem('doj.token')
  const url = new URL('/api/ws', window.location.origin)
  url.protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return token ? new WebSocket(url, `doj-auth.${token}`) : null
}
