import { describe, expect, it } from 'vitest'

import { discussionDraftKey, submissionDraftKey } from './draft'

describe('draft keys', () => {
  it('separates users, activity contexts, and languages', () => {
    const practice = submissionDraftKey('alice', 1000, 'cpp')
    expect(submissionDraftKey('bob', 1000, 'cpp')).not.toBe(practice)
    expect(submissionDraftKey('alice', 1000, 'python')).not.toBe(practice)
    expect(submissionDraftKey('alice', 1000, 'cpp', 1)).not.toBe(practice)
    expect(submissionDraftKey('alice', 1000, 'cpp', undefined, 1)).not.toBe(practice)
  })

  it('separates discussion authors and editor contexts', () => {
    const reply = discussionDraftKey('alice', 'reply', 1)
    expect(discussionDraftKey('bob', 'reply', 1)).not.toBe(reply)
    expect(discussionDraftKey('alice', 'reply', 2)).not.toBe(reply)
    expect(discussionDraftKey('alice', 'edit', 1)).not.toBe(reply)
    expect(discussionDraftKey('alice', 'new')).not.toBe(reply)
  })
})
