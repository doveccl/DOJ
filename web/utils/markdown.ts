import DOMPurify from 'dompurify'

export type MarkdownTrust = 'trusted' | 'ugc'

type MarkdownToken = {
  attrs?: [string, string][] | null
  attrGet?: (name: string) => string | null
  attrSet?: (name: string, value: string) => void
}

type MarkdownRendererSelf = {
  renderToken: (tokens: MarkdownToken[], idx: number, options: unknown) => string
}

type MarkdownRendererRule = (
  tokens: MarkdownToken[],
  idx: number,
  options: unknown,
  env: unknown,
  self: MarkdownRendererSelf
) => string

type MarkdownItLike = {
  renderer: {
    rules: Record<string, unknown>
  }
}

const markdownAssetBases = new Map<string, string | undefined>()

const passthroughMarkdown = (html: string) => html

export function sanitizerForTrust(trust: MarkdownTrust) {
  return (html: string) => {
    return trust === 'trusted' ? passthroughMarkdown(html) : sanitizeMarkdown(html)
  }
}

export function sanitizeMarkdown(html: string) {
  DOMPurify.addHook('uponSanitizeAttribute', allowKaTeXLayoutStyle)
  try {
    return DOMPurify.sanitize(html, {
      USE_PROFILES: { html: true, mathMl: true },
      ADD_ATTR: ['style'],
      FORBID_TAGS: ['script', 'style', 'iframe', 'object', 'embed', 'form'],
      RETURN_TRUSTED_TYPE: false
    })
  } finally {
    DOMPurify.removeHook('uponSanitizeAttribute')
  }
}

const katexLayoutStyleProps = new Set([
  'border-bottom-width',
  'height',
  'margin-left',
  'margin-right',
  'min-width',
  'padding-left',
  'top',
  'vertical-align',
  'width'
])

const katexNonNegativeStyleProps = new Set(['border-bottom-width', 'height', 'min-width', 'padding-left', 'width'])

function allowKaTeXLayoutStyle(node: Element, data: { attrName: string; attrValue: string; keepAttr: boolean }) {
  if (data.attrName !== 'style') {
    return
  }
  const style = isKaTeXNode(node) ? sanitizeKaTeXStyle(data.attrValue) : ''
  if (!style) {
    data.keepAttr = false
    return
  }
  data.attrValue = style
}

function isKaTeXNode(node: Element) {
  for (let current: Element | null = node; current; current = current.parentElement) {
    if (current.classList.contains('katex') || current.classList.contains('katex-display')) {
      return true
    }
  }
  return false
}

export function sanitizeKaTeXStyle(value: string) {
  const declarations = value
    .split(';')
    .map((part) => part.trim())
    .filter(Boolean)
    .map((part) => {
      const separator = part.indexOf(':')
      if (separator < 1) {
        return null
      }
      const prop = part.slice(0, separator).trim().toLowerCase()
      const next = part.slice(separator + 1).trim().toLowerCase()
      if (!katexLayoutStyleProps.has(prop) || !safeKaTeXStyleValue(prop, next)) {
        return null
      }
      return `${prop}: ${next}`
    })
    .filter((part): part is string => Boolean(part))
  return declarations.join('; ')
}

function safeKaTeXStyleValue(prop: string, value: string) {
  const match = value.match(/^(-?(?:\d+(?:\.\d+)?|\.\d+))(em|px|%)?$/)
  if (!match) {
    return false
  }
  const amount = Number(match[1])
  if (!Number.isFinite(amount)) {
    return false
  }
  if (katexNonNegativeStyleProps.has(prop) && amount < 0) {
    return false
  }
  const unit = match[2] ?? ''
  if (unit === '%' && Math.abs(amount) > 100) {
    return false
  }
  if (unit === 'px' && Math.abs(amount) > 10000) {
    return false
  }
  return unit === 'px' || Math.abs(amount) <= 100
}

export function rewriteAssetURL(value: string, assetBase?: string) {
  if (!assetBase || !value || /^[a-z][a-z0-9+.-]*:/i.test(value) || value.startsWith('//')) {
    return value
  }
  const match = value.match(/^\.?\/?assets\/(.+)$/)
  if (!match) {
    return value
  }
  const base = assetBase.endsWith('/') ? assetBase : `${assetBase}/`
  return `${base}${match[1]}`
}

export function problemAssetUploadMarkdownURL(value: string, problemID: number) {
  const prefix = `/api/problems/${problemID}/assets/`
  if (!value.startsWith(prefix)) {
    return value
  }
  return `./assets/${value.slice(prefix.length)}`
}

export function setMarkdownAssetBase(editorID: string, assetBase?: string) {
  markdownAssetBases.set(editorID, assetBase)
}

export function clearMarkdownAssetBase(editorID: string) {
  markdownAssetBases.delete(editorID)
}

export function configureMarkdownAssetRenderer(md: MarkdownItLike, editorID: string) {
  const renderToken = (tokens: MarkdownToken[], idx: number, options: unknown, self: MarkdownRendererSelf) =>
    self.renderToken(tokens, idx, options)
  const imageRule = md.renderer.rules.image as MarkdownRendererRule | undefined
  const linkOpenRule = md.renderer.rules.link_open as MarkdownRendererRule | undefined

  const imageRenderer: MarkdownRendererRule = (tokens, idx, options, env, self) => {
    rewriteTokenURL(tokens[idx], 'src', markdownAssetBases.get(editorID))
    return imageRule ? imageRule(tokens, idx, options, env, self) : renderToken(tokens, idx, options, self)
  }
  const linkOpenRenderer: MarkdownRendererRule = (tokens, idx, options, env, self) => {
    rewriteTokenURL(tokens[idx], 'href', markdownAssetBases.get(editorID))
    return linkOpenRule ? linkOpenRule(tokens, idx, options, env, self) : renderToken(tokens, idx, options, self)
  }
  md.renderer.rules.image = imageRenderer
  md.renderer.rules.link_open = linkOpenRenderer
}

function rewriteTokenURL(token: MarkdownToken | undefined, attr: string, assetBase?: string) {
  if (!token) {
    return
  }
  const value = token.attrGet?.(attr) ?? token.attrs?.find(([name]) => name === attr)?.[1]
  if (!value) {
    return
  }
  const next = rewriteAssetURL(value, assetBase)
  if (next === value) {
    return
  }
  if (token.attrSet) {
    token.attrSet(attr, next)
    return
  }
  if (!token.attrs) {
    token.attrs = []
  }
  const existing = token.attrs.find((item) => item[0] === attr)
  if (existing) {
    existing[1] = next
  } else {
    token.attrs.push([attr, next])
  }
}
