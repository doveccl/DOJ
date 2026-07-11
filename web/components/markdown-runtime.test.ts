import { afterEach, describe, expect, it } from 'vitest'

import { disableMarkdownHTML, configurePlainCodeBlocks } from './markdown-runtime'

describe('markdown runtime', () => {
  afterEach(() => {
    globalThis.hljs = undefined
  })

  it('disables raw html for every markdown document', () => {
    const options: { html?: boolean } = {}
    disableMarkdownHTML({ set: (next) => Object.assign(options, next) })

    expect(options).toEqual({ html: false })
  })

  it('renders unlabeled code blocks as plain text', () => {
    const md = runtime()
    configurePlainCodeBlocks(md)

    expect(md.options.highlight?.('-34 -14 -10', '', '')).toBe(
      '<pre><code class="language-" language=""><span class="md-editor-code-block">-34 -14 -10</span></code></pre>'
    )
  })

  it('renders unknown code languages as plain text', () => {
    globalThis.hljs = { getLanguage: (lang) => lang === 'cpp' }
    const md = runtime()
    configurePlainCodeBlocks(md)

    expect(md.options.highlight?.('if (ok) return;', '伪代码', '')).toBe(
      '<pre><code class="language-伪代码" language="伪代码"><span class="md-editor-code-block">if (ok) return;</span></code></pre>'
    )
    expect(md.options.highlight?.('wat', 'notalang', '')).toBe(
      '<pre><code class="language-notalang" language="notalang"><span class="md-editor-code-block">wat</span></code></pre>'
    )
  })

  it('keeps explicit language highlighting delegated to markdown editor', () => {
    globalThis.hljs = { getLanguage: (lang) => ['cpp', 'C++', 'cxx'].includes(lang) }
    const md = runtime()
    configurePlainCodeBlocks(md)

    expect(md.options.highlight?.('return 0;', 'cpp', 'data-x')).toBe('<em>cpp:return 0;:data-x</em>')
    expect(md.options.highlight?.('return 0;', 'C++', 'data-x')).toBe('<em>C++:return 0;:data-x</em>')
    expect(md.options.highlight?.('return 0;', 'cxx', 'data-x')).toBe('<em>cxx:return 0;:data-x</em>')
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
