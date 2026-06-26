import { describe, expect, it } from 'vitest'

import { markdownEditorExtensionURLs } from './markdown-assets'

describe('markdown editor runtime assets', () => {
  it('keeps md-editor dynamic extension urls same-origin', () => {
    for (const url of markdownEditorExtensionURLs()) {
      expect(url).not.toMatch(/^https?:\/\//)
      expect(url).not.toMatch(/^\/\//)
    }
  })
})
