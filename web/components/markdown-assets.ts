import highlightScript from '@highlightjs/cdn-assets/highlight.min.js?url'
import githubDarkStyle from '@highlightjs/cdn-assets/styles/github-dark.min.css?url'
import githubStyle from '@highlightjs/cdn-assets/styles/github.min.css?url'
import Cropper from 'cropperjs'
import screenfull from 'screenfull'
import katex from 'katex'
import type { GlobalConfig } from 'md-editor-rt'
import 'cropperjs/dist/cropper.css'
import 'katex/dist/katex.min.css'
import 'md-editor-rt/lib/style.css'

const highlightStyles = {
  github: {
    light: githubStyle,
    dark: githubDarkStyle
  }
}

export const markdownEditorExtensions = {
  highlight: {
    js: highlightScript,
    css: highlightStyles
  },
  cropper: {
    instance: Cropper
  },
  screenfull: {
    instance: screenfull
  },
  katex: {
    instance: katex
  }
} satisfies GlobalConfig['editorExtensions']

export function markdownEditorExtensionURLs() {
  const urls = [
    markdownEditorExtensions.highlight.js,
    ...Object.values(markdownEditorExtensions.highlight.css).flatMap((theme) => [theme.light, theme.dark]),
  ]
  return urls.filter(Boolean)
}
