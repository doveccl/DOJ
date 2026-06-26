import { config } from 'md-editor-rt'

import { configureMarkdownAssetRenderer } from '../utils/markdown'
import { markdownEditorExtensions } from './markdown-assets'

let markdownRuntimeConfigured = false

export function configureMarkdownRuntime() {
  if (markdownRuntimeConfigured) {
    return
  }
  config({
    editorExtensions: markdownEditorExtensions,
    markdownItConfig: (md, options) => {
      configureMarkdownAssetRenderer(md, options.editorId)
    }
  })
  markdownRuntimeConfigured = true
}
