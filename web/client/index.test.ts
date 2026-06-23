import { describe, expect, it } from 'vitest'

import { normalizeApiBase } from './index'

describe('normalizeApiBase', () => {
  it('keeps the base before OpenAPI /api paths', () => {
    expect(normalizeApiBase('http://example.test/api')).toBe('http://example.test')
    expect(normalizeApiBase('http://example.test/judge/api')).toBe('http://example.test/judge')
    expect(normalizeApiBase('http://example.test')).toBe('http://example.test')
  })
})
