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

export const trustedMarkdownIDs = new Set(['home-notice'])

export function problemMarkdownID(problemID: number) {
  return `P${problemID}`
}

export function trustedMarkdownID(editorID: string) {
  return trustedMarkdownIDs.has(editorID) || /^P\d+$/.test(editorID)
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
    rewriteTokenURL(tokens[idx], 'href', editorID)
    return linkOpenRule ? linkOpenRule(tokens, idx, options, env, self) : renderToken(tokens, idx, options, self)
  }
  md.renderer.rules.image = imageRenderer
  md.renderer.rules.link_open = linkOpenRenderer
}

function rewriteTokenURL(token: MarkdownToken | undefined, attr: string, editorID: string) {
  if (!token) {
    return
  }
  const value = token.attrGet?.(attr) ?? token.attrs?.find(([name]) => name === attr)?.[1]
  if (!value) {
    return
  }
  const next = rewriteAssetURL(value, editorID)
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
