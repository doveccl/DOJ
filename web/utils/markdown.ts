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

export function problemMarkdownID(problemID: number) {
  return `P${problemID}`
}

export function trustedMarkdownID(editorID: string) {
  return editorID === 'home-notice' || /^P\d+$/.test(editorID)
}

export function codeBlock(value: string, lang: string) {
  return `\`\`\`${lang}\n${value.replaceAll('```', '`\\`\\`')}\n\`\`\``
}

export function rewriteAssetURL(value: string, editorID?: string) {
  const problemID = editorID?.match(/^P(\d+)$/)?.[1]
  const problemBase = problemID ? `/api/problems/${problemID}/` : undefined
  if (!problemBase || !value || /^[a-z][a-z0-9+.-]*:/i.test(value) || value.startsWith('//')) {
    return value
  }
  const match = value.match(/^\.?\/?(assets\/.+)$/)
  if (!match) {
    return value
  }
  return `${problemBase}${match[1]}`
}

export function problemAssetUploadMarkdownURL(value: string, problemID: number) {
  const prefix = `/api/problems/${problemID}/assets/`
  if (!value.startsWith(prefix)) {
    return value
  }
  return `./assets/${value.slice(prefix.length)}`
}

export function configureMarkdownAssetRenderer(md: MarkdownItLike, editorID: string) {
  const renderToken = (tokens: MarkdownToken[], idx: number, options: unknown, self: MarkdownRendererSelf) =>
    self.renderToken(tokens, idx, options)
  const imageRule = md.renderer.rules.image as MarkdownRendererRule | undefined
  const linkOpenRule = md.renderer.rules.link_open as MarkdownRendererRule | undefined

  const imageRenderer: MarkdownRendererRule = (tokens, idx, options, env, self) => {
    rewriteTokenURL(tokens[idx], 'src', editorID)
    return imageRule ? imageRule(tokens, idx, options, env, self) : renderToken(tokens, idx, options, self)
  }
  const linkOpenRenderer: MarkdownRendererRule = (tokens, idx, options, env, self) => {
    const href = rewriteTokenURL(tokens[idx], 'href', editorID)
    if (href && isExternalURL(href)) {
      setTokenAttr(tokens[idx], 'target', '_blank')
      setTokenAttr(tokens[idx], 'rel', 'noopener noreferrer')
    }
    return linkOpenRule ? linkOpenRule(tokens, idx, options, env, self) : renderToken(tokens, idx, options, self)
  }
  md.renderer.rules.image = imageRenderer
  md.renderer.rules.link_open = linkOpenRenderer
}

function rewriteTokenURL(token: MarkdownToken | undefined, attr: string, editorID: string) {
  if (!token) {
    return ''
  }
  const value = token.attrGet?.(attr) ?? token.attrs?.find(([name]) => name === attr)?.[1]
  if (!value) {
    return ''
  }
  const next = rewriteAssetURL(value, editorID)
  if (next === value) {
    return value
  }
  setTokenAttr(token, attr, next)
  return next
}

function setTokenAttr(token: MarkdownToken | undefined, attr: string, value: string) {
  if (!token) {
    return
  }
  if (token.attrSet) {
    token.attrSet(attr, value)
    return
  }
  if (!token.attrs) {
    token.attrs = []
  }
  const existing = token.attrs.find((item) => item[0] === attr)
  if (existing) {
    existing[1] = value
  } else {
    token.attrs.push([attr, value])
  }
}

function isExternalURL(value: string) {
  return /^https?:\/\//i.test(value) || value.startsWith('//')
}
