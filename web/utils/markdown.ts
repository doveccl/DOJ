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
  return DOMPurify.sanitize(html, {
    USE_PROFILES: { html: true, mathMl: true },
    FORBID_TAGS: ['script', 'style', 'iframe', 'object', 'embed', 'form'],
    FORBID_ATTR: ['style'],
    RETURN_TRUSTED_TYPE: false
  })
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
