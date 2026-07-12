import { describe, expect, it } from 'vitest'

import {
  configureMarkdownAssetRenderer,
  problemAssetUploadMarkdownURL,
  problemMarkdownID,
  rewriteAssetURL
} from './markdown'

describe('markdown assets', () => {
  it('maps only relative problem asset urls', () => {
    expect(rewriteAssetURL('./assets/a.png', 'P1000')).toBe('/api/problems/1000/assets/a.png')
    expect(rewriteAssetURL('assets/a.png#v', 'P1000')).toBe('/api/problems/1000/assets/a.png#v')
    expect(rewriteAssetURL('/assets/a.png', 'P1000')).toBe('/api/problems/1000/assets/a.png')
    expect(rewriteAssetURL('https://example.com/assets/a.png', 'P1000')).toBe('https://example.com/assets/a.png')
    expect(rewriteAssetURL('./data/1.in', 'P1000')).toBe('./data/1.in')
    expect(rewriteAssetURL('./assets/a.png', 'md-1')).toBe('./assets/a.png')
  })

  it('uses relative urls for newly uploaded problem assets in markdown', () => {
    expect(problemAssetUploadMarkdownURL('/api/problems/1000/assets/a.png', 1000)).toBe('./assets/a.png')
    expect(problemAssetUploadMarkdownURL('/api/problems/1000/assets/nested/a.png', 1000)).toBe('./assets/nested/a.png')
    expect(problemAssetUploadMarkdownURL('/api/problems/1001/assets/a.png', 1000)).toBe('/api/problems/1001/assets/a.png')
  })

  it('builds problem markdown ids', () => {
    expect(problemMarkdownID(1000)).toBe('P1000')
  })

  it('rewrites image and link tokens during markdown rendering', () => {
    const md: { renderer: { rules: Record<string, unknown> } } = {
      renderer: {
        rules: {}
      }
    }
    configureMarkdownAssetRenderer(md, 'P1000')
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
  })

  it('opens external markdown links in a new tab', () => {
    const md: { renderer: { rules: Record<string, unknown> } } = {
      renderer: {
        rules: {}
      }
    }
    configureMarkdownAssetRenderer(md, 'P1000')
    const self = {
      renderToken: () => ''
    }
    const external = token('href', 'https://example.com')
    const asset = token('href', './assets/a.pdf')

    const linkRule = md.renderer.rules.link_open as TestRule
    linkRule([external], 0, {}, {}, self)
    linkRule([asset], 0, {}, {}, self)

    expect(external.attrGet('target')).toBe('_blank')
    expect(external.attrGet('rel')).toBe('noopener noreferrer')
    expect(asset.attrGet('href')).toBe('/api/problems/1000/assets/a.pdf')
    expect(asset.attrGet('target')).toBeNull()
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
