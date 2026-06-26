import highlightScript from '@highlightjs/cdn-assets/highlight.min.js?url'
import a11yDarkStyle from '@highlightjs/cdn-assets/styles/a11y-dark.min.css?url'
import a11yLightStyle from '@highlightjs/cdn-assets/styles/a11y-light.min.css?url'
import atomDarkStyle from '@highlightjs/cdn-assets/styles/atom-one-dark.min.css?url'
import atomLightStyle from '@highlightjs/cdn-assets/styles/atom-one-light.min.css?url'
import gradientDarkStyle from '@highlightjs/cdn-assets/styles/gradient-dark.min.css?url'
import gradientLightStyle from '@highlightjs/cdn-assets/styles/gradient-light.min.css?url'
import githubDarkStyle from '@highlightjs/cdn-assets/styles/github-dark.min.css?url'
import githubStyle from '@highlightjs/cdn-assets/styles/github.min.css?url'
import kimbieDarkStyle from '@highlightjs/cdn-assets/styles/kimbie-dark.min.css?url'
import kimbieLightStyle from '@highlightjs/cdn-assets/styles/kimbie-light.min.css?url'
import paraisoDarkStyle from '@highlightjs/cdn-assets/styles/paraiso-dark.min.css?url'
import paraisoLightStyle from '@highlightjs/cdn-assets/styles/paraiso-light.min.css?url'
import qtcreatorDarkStyle from '@highlightjs/cdn-assets/styles/qtcreator-dark.min.css?url'
import qtcreatorLightStyle from '@highlightjs/cdn-assets/styles/qtcreator-light.min.css?url'
import stackoverflowDarkStyle from '@highlightjs/cdn-assets/styles/stackoverflow-dark.min.css?url'
import stackoverflowLightStyle from '@highlightjs/cdn-assets/styles/stackoverflow-light.min.css?url'
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
  a11y: {
    light: a11yLightStyle,
    dark: a11yDarkStyle
  },
  atom: {
    light: atomLightStyle,
    dark: atomDarkStyle
  },
  github: {
    light: githubStyle,
    dark: githubDarkStyle
  },
  gradient: {
    light: gradientLightStyle,
    dark: gradientDarkStyle
  },
  kimbie: {
    light: kimbieLightStyle,
    dark: kimbieDarkStyle
  },
  paraiso: {
    light: paraisoLightStyle,
    dark: paraisoDarkStyle
  },
  qtcreator: {
    light: qtcreatorLightStyle,
    dark: qtcreatorDarkStyle
  },
  stackoverflow: {
    light: stackoverflowLightStyle,
    dark: stackoverflowDarkStyle
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
