import { describe, expect, it } from 'vitest'

import { markdownEditorExtensions } from './markdown-assets'

describe('markdown editor runtime assets', () => {
  it('keeps md-editor dynamic extension urls same-origin', () => {
    const urls = [
      markdownEditorExtensions.highlight.js,
      ...Object.values(markdownEditorExtensions.highlight.css).flatMap((theme) => [theme.light, theme.dark])
    ]
    for (const url of urls) {
      expect(url).not.toMatch(/^https?:\/\//)
      expect(url).not.toMatch(/^\/\//)
    }
  })
})
