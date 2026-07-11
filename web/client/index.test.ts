import { describe, expect, it } from 'vitest'

import { APIError, apiData, apiUrl, shouldRetryQuery } from './index'

describe('apiUrl', () => {
  it('uses the server origin when no browser origin exists', () => {
    expect(apiUrl('/api/home').toString()).toBe('http://localhost:7974/api/home')
  })
})

describe('API errors', () => {
  it('keeps HTTP status and retries only transient query failures once', async () => {
    await expect(apiData(Promise.resolve({ error: { message: 'missing' }, response: new Response('', { status: 404 }) }))).rejects.toMatchObject({ status: 404 })
    expect(shouldRetryQuery(0, new APIError('server', 500))).toBe(true)
    expect(shouldRetryQuery(0, new APIError('missing', 404))).toBe(false)
    expect(shouldRetryQuery(1, new Error('network'))).toBe(false)
  })
})
