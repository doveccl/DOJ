import { config, prefix as mdEditorPrefix } from 'md-editor-rt'

import { configureMarkdownAssetRenderer, trustedMarkdownID } from '../utils/markdown'
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
    }
  })
  markdownRuntimeConfigured = true
}

export function configureMarkdownHTML(md: Pick<MarkdownRuntime, 'set'>, editorID: string) {
  if (!trustedMarkdownID(editorID)) {
    md.set({ html: false })
  }
}

export function configurePlainCodeBlocks(md: MarkdownRuntime) {
  const highlight = md.options.highlight ?? undefined
  md.options.highlight = (code, lang, attrs) => {
    if (lang.trim()) {
      return highlight?.(code, lang, attrs) ?? ''
    }
    const escaped = md.utils.escapeHtml(code).replace(/^\n+|\n+$/g, '')
    return `<pre><code class="language-" language=""><span class="${mdEditorPrefix}-code-block">${escaped}</span></code></pre>`
  }
}
