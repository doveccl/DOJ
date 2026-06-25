import { describe, expect, it } from 'vitest'

import {
  clearMarkdownAssetBase,
  configureMarkdownAssetRenderer,
  rewriteAssetURL,
  setMarkdownAssetBase
} from './markdown'

describe('markdown assets', () => {
  it('maps only relative problem asset urls', () => {
    expect(rewriteAssetURL('./assets/a.png', '/api/problems/1000/assets/')).toBe('/api/problems/1000/assets/a.png')
    expect(rewriteAssetURL('assets/a.png#v', '/api/problems/1000/assets')).toBe('/api/problems/1000/assets/a.png#v')
    expect(rewriteAssetURL('/assets/a.png', '/api/problems/1000/assets')).toBe('/api/problems/1000/assets/a.png')
    expect(rewriteAssetURL('https://example.com/assets/a.png', '/api/problems/1000/assets')).toBe('https://example.com/assets/a.png')
    expect(rewriteAssetURL('./data/1.in', '/api/problems/1000/assets')).toBe('./data/1.in')
  })

  it('rewrites image and link tokens during markdown rendering', () => {
    const md: { renderer: { rules: Record<string, unknown> } } = {
      renderer: {
        rules: {}
      }
    }
    setMarkdownAssetBase('editor-1', '/api/problems/1000/assets')
    configureMarkdownAssetRenderer(md, 'editor-1')
    const self = {
      renderToken: () => ''
    }
    const image = token('src', './assets/a.png')
    const link = token('href', '/assets/b.png')

    const imageRule = md.renderer.rules.image as TestRule
    const linkRule = md.renderer.rules.link_open as TestRule
    imageRule([image], 0, {}, {}, self)
    linkRule([link], 0, {}, {}, self)

    expect(image.attrGet('src')).toBe('/api/problems/1000/assets/a.png')
    expect(link.attrGet('href')).toBe('/api/problems/1000/assets/b.png')
    clearMarkdownAssetBase('editor-1')
  })
})

function token(name: string, value: string) {
  const attrs: [string, string][] = [[name, value]]
  return {
    attrs,
    attrGet: (key: string) => attrs.find(([attr]) => attr === key)?.[1] ?? null,
    attrSet: (key: string, next: string) => {
      const found = attrs.find(([attr]) => attr === key)
      if (found) {
        found[1] = next
      } else {
        attrs.push([key, next])
      }
    }
  }
}

type TestToken = ReturnType<typeof token>
type TestRule = (tokens: TestToken[], idx: number, options: unknown, env: unknown, self: { renderToken: () => string }) => string
