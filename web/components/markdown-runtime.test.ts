import { describe, expect, it } from 'vitest'

import { configureMarkdownHTML, configurePlainCodeBlocks } from './markdown-runtime'

describe('markdown runtime', () => {
  it('disables raw html for untrusted markdown ids', () => {
    const options: { html?: boolean } = {}
    configureMarkdownHTML({ set: (next) => Object.assign(options, next) }, 'md-1')

    expect(options).toEqual({ html: false })
  })

  it('keeps raw html enabled for trusted markdown ids', () => {
    const options: { html?: boolean } = {}
    configureMarkdownHTML({ set: (next) => Object.assign(options, next) }, 'P1000')
    configureMarkdownHTML({ set: (next) => Object.assign(options, next) }, 'home-notice')

    expect(options).toEqual({})
  })

  it('renders unlabeled code blocks as plain text', () => {
    const md = runtime()
    configurePlainCodeBlocks(md)

    expect(md.options.highlight?.('-34 -14 -10', '', '')).toBe(
      '<pre><code class="language-" language=""><span class="md-editor-code-block">-34 -14 -10</span></code></pre>'
    )
  })

  it('keeps explicit language highlighting delegated to markdown editor', () => {
    const md = runtime()
    configurePlainCodeBlocks(md)

    expect(md.options.highlight?.('return 0;', 'cpp', 'data-x')).toBe('<em>cpp:return 0;:data-x</em>')
  })
})

function runtime() {
  return {
    set: () => undefined,
    options: {
      highlight: (code: string, lang: string, attrs: string) => `<em>${lang}:${code}:${attrs}</em>`
    },
    utils: {
      escapeHtml: (value: string) => value.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;')
    }
  }
}
