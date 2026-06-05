export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly body: unknown
  ) {
    super(message)
  }
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
  const body = text ? JSON.parse(text) : null
  if (!response.ok) {
    const message =
      typeof body?.message === 'string' ? body.message : `${response.status} ${response.statusText}`
    throw new ApiError(message, response.status, body)
  }

  return body as T
}
