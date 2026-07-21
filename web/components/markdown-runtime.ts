import { config, prefix as mdEditorPrefix } from 'md-editor-rt'

import { configureMarkdownAssetRenderer, trustedMarkdownID } from '../utils/markdown'
import { limits } from '../utils/limits'
import { markdownEditorExtensions } from './markdown-assets'

type MarkdownRuntime = {
  set: (options: { html?: boolean }) => void
  options: {
    highlight?: ((code: string, lang: string, attrs: string) => string) | null
  }
  utils: {
    escapeHtml: (value: string) => string
  }
}

type MentionState = {
  src: string
  pos: number
  posMax: number
  linkLevel?: number
  push: (type: string, tag: string, nesting: number) => { attrSet?: (name: string, value: string) => void; content: string }
}

type MentionMarkdown = {
  inline: {
    ruler: {
      before: (before: string, name: string, rule: (state: MentionState, silent: boolean) => boolean) => void
    }
  }
}

declare global {
  var hljs: { getLanguage: (lang: string) => unknown } | undefined
}

let markdownRuntimeConfigured = false

export function configureMarkdownRuntime() {
  if (markdownRuntimeConfigured) {
    return
  }
  config({
    editorExtensions: markdownEditorExtensions,
    markdownItConfig: (md, options) => {
      configureMarkdownHTML(md, options.editorId)
      configureMarkdownAssetRenderer(md, options.editorId)
      configurePlainCodeBlocks(md)
      if (options.editorId.startsWith('discussion-')) {
        configureMarkdownMentions(md as unknown as MentionMarkdown)
      }
    }
  })
  markdownRuntimeConfigured = true
}

export function configureMarkdownMentions(md: MentionMarkdown) {
  md.inline.ruler.before('emphasis', 'mention', (state, silent) => {
    const start = state.pos
    if (state.src[start] !== '@' || (state.linkLevel ?? 0) > 0 || start > 0 && mentionBoundaryChar(state.src[start - 1])) {
      return false
    }
    let end = start + 1
    while (end < state.posMax && mentionNameChar(state.src[end])) {
      end++
    }
    const name = state.src.slice(start + 1, end)
    if (name.length < limits.usernameMin || name.length > limits.username) {
      return false
    }
    if (!silent) {
      state.push('link_open', 'a', 1).attrSet?.('href', `/users/${encodeURIComponent(name)}`)
      state.push('text', '', 0).content = `@${name}`
      state.push('link_close', 'a', -1)
    }
    state.pos = end
    return true
  })
}

function mentionNameChar(value: string) {
  return /^[A-Za-z0-9_-]$/.test(value)
}

function mentionBoundaryChar(value: string) {
  return value === '@' || mentionNameChar(value)
}

export function configureMarkdownHTML(md: Pick<MarkdownRuntime, 'set'>, editorID: string) {
  if (!trustedMarkdownID(editorID)) {
    md.set({ html: false })
  }
}

export function configurePlainCodeBlocks(md: MarkdownRuntime) {
  const highlight = md.options.highlight ?? undefined
  md.options.highlight = (code, lang, attrs) => {
    if (supportedCodeLanguage(lang)) {
      return highlight?.(code, lang, attrs) ?? ''
    }
    const escaped = md.utils.escapeHtml(code).replace(/^\n+|\n+$/g, '')
    const language = md.utils.escapeHtml(lang.trim())
    return `<pre><code class="language-${language}" language="${language}"><span class="${mdEditorPrefix}-code-block">${escaped}</span></code></pre>`
  }
}

function supportedCodeLanguage(lang: string) {
  const key = lang.trim()
  if (!key || !/^[a-zA-Z0-9_+#.-]+$/.test(key)) {
    return false
  }
  return !globalThis.hljs || globalThis.hljs.getLanguage(key)
}
