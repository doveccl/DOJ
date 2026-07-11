import { config, prefix as mdEditorPrefix } from 'md-editor-rt'

import { configureMarkdownAssetRenderer } from '../utils/markdown'
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
      disableMarkdownHTML(md)
      configureMarkdownAssetRenderer(md, options.editorId)
      configurePlainCodeBlocks(md)
    }
  })
  markdownRuntimeConfigured = true
}

export function disableMarkdownHTML(md: Pick<MarkdownRuntime, 'set'>) {
  md.set({ html: false })
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
