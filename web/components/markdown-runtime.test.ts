import { afterEach, describe, expect, it } from 'vitest'

import { configureMarkdownHTML, configureMarkdownMentions, configurePlainCodeBlocks } from './markdown-runtime'

describe('markdown runtime', () => {
  afterEach(() => {
    globalThis.hljs = undefined
  })

  it('disables raw html for user content', () => {
    const options: { html?: boolean } = {}
    configureMarkdownHTML({ set: (next) => Object.assign(options, next) }, 'md-1')

    expect(options).toEqual({ html: false })
  })

  it('allows raw html in admin-managed content', () => {
    const options: { html?: boolean } = {}
    configureMarkdownHTML({ set: (next) => Object.assign(options, next) }, 'P1098')
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

  it('links standalone discussion mentions without rewriting emails or links', () => {
    const render = mentionRule()

    expect(runMention(render, '@Alice')).toEqual({ matched: true, position: 6, href: '/users/Alice', text: '@Alice' })
    expect(runMention(render, 'a@example.com', 1).matched).toBe(false)
    expect(runMention(render, '@Alice', 0, 1).matched).toBe(false)
  })
})

function mentionRule() {
  let rule: ((state: MentionState, silent: boolean) => boolean) | undefined
  configureMarkdownMentions({
    inline: {
      ruler: {
        before: (_before, _name, next) => {
          rule = next
        }
      }
    }
  })
  if (!rule) {
    throw new Error('mention rule was not registered')
  }
  return rule
}

type MentionState = {
  src: string
  pos: number
  posMax: number
  linkLevel?: number
  push: (type: string, tag: string, nesting: number) => { attrSet: (name: string, value: string) => void; content: string }
}

function runMention(render: ReturnType<typeof mentionRule>, source: string, position = 0, linkLevel = 0) {
  let href = ''
  let text = ''
  const state: MentionState = {
    src: source,
    pos: position,
    posMax: source.length,
    linkLevel,
    push: (type) => ({
      attrSet: (name, value) => {
        if (type === 'link_open' && name === 'href') {
          href = value
        }
      },
      get content() {
        return text
      },
      set content(value: string) {
        if (type === 'text') {
          text = value
        }
      }
    })
  }
  return { matched: render(state, false), position: state.pos, href, text }
}

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
