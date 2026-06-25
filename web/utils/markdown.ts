import DOMPurify from 'dompurify'

export type MarkdownTrust = 'trusted' | 'ugc'

const passthroughMarkdown = (html: string) => html

export function sanitizerForTrust(trust: MarkdownTrust, options: { assetBase?: string } = {}) {
  return (html: string) => {
    const next = trust === 'trusted' ? passthroughMarkdown(html) : sanitizeMarkdown(html)
    return rewriteAssetHTML(next, options.assetBase)
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

function rewriteAssetHTML(html: string, assetBase?: string) {
  if (!assetBase || !globalThis.DOMParser) {
    return html
  }
  const document = new DOMParser().parseFromString(html, 'text/html')
  document.querySelectorAll('img[src], a[href]').forEach((node) => {
    const attr = node instanceof HTMLImageElement ? 'src' : 'href'
    const value = node.getAttribute(attr)
    if (value) {
      node.setAttribute(attr, rewriteAssetURL(value, assetBase))
    }
  })
  return document.body.innerHTML
}
