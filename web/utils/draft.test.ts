import { describe, expect, it } from 'vitest'

import { submissionDraftKey } from './draft'

describe('submission draft key', () => {
  it('separates users, activity contexts, and languages', () => {
    const practice = submissionDraftKey('alice', 1000, 'cpp')
    expect(submissionDraftKey('bob', 1000, 'cpp')).not.toBe(practice)
    expect(submissionDraftKey('alice', 1000, 'python')).not.toBe(practice)
    expect(submissionDraftKey('alice', 1000, 'cpp', 1)).not.toBe(practice)
    expect(submissionDraftKey('alice', 1000, 'cpp', undefined, 1)).not.toBe(practice)
  })
})
