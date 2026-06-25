import { describe, expect, it } from 'vitest'

import { rewriteAssetURL } from './markdown'

describe('markdown assets', () => {
  it('maps only relative problem asset urls', () => {
    expect(rewriteAssetURL('./assets/a.png', '/api/problems/1000/assets/')).toBe('/api/problems/1000/assets/a.png')
    expect(rewriteAssetURL('assets/a.png#v', '/api/problems/1000/assets')).toBe('/api/problems/1000/assets/a.png#v')
    expect(rewriteAssetURL('/assets/a.png', '/api/problems/1000/assets')).toBe('/api/problems/1000/assets/a.png')
    expect(rewriteAssetURL('https://example.com/assets/a.png', '/api/problems/1000/assets')).toBe('https://example.com/assets/a.png')
    expect(rewriteAssetURL('./data/1.in', '/api/problems/1000/assets')).toBe('./data/1.in')
  })
})
