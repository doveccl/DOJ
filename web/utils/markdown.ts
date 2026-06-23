import DOMPurify from 'dompurify'

export type MarkdownTrust = 'trusted' | 'ugc'

const passthroughMarkdown = (html: string) => html

export function sanitizerForTrust(trust: MarkdownTrust) {
  return trust === 'trusted' ? passthroughMarkdown : sanitizeMarkdown
}

export function sanitizeMarkdown(html: string) {
  return DOMPurify.sanitize(html, {
    USE_PROFILES: { html: true, mathMl: true },
    FORBID_TAGS: ['script', 'style', 'iframe', 'object', 'embed', 'form'],
    FORBID_ATTR: ['style'],
    RETURN_TRUSTED_TYPE: false
  })
}
