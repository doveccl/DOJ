import highlightScript from '@highlightjs/cdn-assets/highlight.min.js?url'
import githubDarkStyle from '@highlightjs/cdn-assets/styles/github-dark.min.css?url'
import githubStyle from '@highlightjs/cdn-assets/styles/github.min.css?url'
import Cropper from 'cropperjs'
import screenfull from 'screenfull'
import katex from 'katex'
import type { GlobalConfig } from 'md-editor-rt'
import disabledExtensionStyle from '../assets/md-editor-disabled.css?url'
import disabledExtensionScript from '../assets/md-editor-disabled.js?url'
import 'cropperjs/dist/cropper.css'
import 'katex/dist/katex.min.css'
import 'md-editor-rt/lib/style.css'

const disabledPrettier = {
  format() {
    throw new Error('Markdown prettier is disabled')
  }
}

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
  prettier: {
    prettierInstance: disabledPrettier,
    parserMarkdownInstance: {},
    standaloneJs: disabledExtensionScript,
    parserMarkdownJs: disabledExtensionScript
  },
  cropper: {
    instance: Cropper,
    js: disabledExtensionScript,
    css: disabledExtensionStyle
  },
  screenfull: {
    instance: screenfull,
    js: disabledExtensionScript
  },
  mermaid: {
    js: disabledExtensionScript,
    enableZoom: false
  },
  katex: {
    instance: katex,
    js: disabledExtensionScript,
    css: disabledExtensionStyle
  },
  echarts: {
    js: disabledExtensionScript,
    parseOption: () => ({})
  }
} satisfies GlobalConfig['editorExtensions']

export function markdownEditorExtensionURLs() {
  const urls = [
    markdownEditorExtensions.highlight.js,
    ...Object.values(markdownEditorExtensions.highlight.css).flatMap((theme) => [theme.light, theme.dark]),
    markdownEditorExtensions.prettier.standaloneJs,
    markdownEditorExtensions.prettier.parserMarkdownJs,
    markdownEditorExtensions.cropper.js,
    markdownEditorExtensions.cropper.css,
    markdownEditorExtensions.screenfull.js,
    markdownEditorExtensions.mermaid.js,
    markdownEditorExtensions.katex.js,
    markdownEditorExtensions.katex.css,
    markdownEditorExtensions.echarts.js
  ]
  return urls.filter(Boolean)
}
