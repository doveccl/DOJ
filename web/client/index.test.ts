import { describe, expect, it } from 'vitest'

import { apiUrl } from './index'

describe('apiUrl', () => {
  it('uses the server origin when no browser origin exists', () => {
    expect(apiUrl('/api/home').toString()).toBe('http://localhost:7974/api/home')
  })
})
