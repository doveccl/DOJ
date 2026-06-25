import { lazy, Suspense } from 'react'

import type { MarkdownTrust } from '../utils/markdown'

type MarkdownEditorProps = {
  value?: string
  minHeight?: number
  readOnly?: boolean
  trust?: MarkdownTrust
  assetBase?: string
  upload?: (file: File) => Promise<string>
  variant?: 'editor' | 'preview'
  onChange?: (value: string) => void
}

const LazyMarkdownEditor = lazy(() => import('./mdeditor').then((mod) => ({ default: mod.MarkdownEditor })))

export function MarkdownEditor(props: MarkdownEditorProps) {
  return (
    <Suspense fallback={<div className="markdownShell markdownFallback" style={{ minHeight: props.minHeight ?? 260 }} />}>
      <LazyMarkdownEditor {...props} />
    </Suspense>
  )
}

export function MarkdownPreview({ value, trust = 'ugc', assetBase }: { value: string; trust?: MarkdownTrust; assetBase?: string }) {
  if (!value.trim()) {
    return null
  }

  return <MarkdownEditor value={value} minHeight={0} readOnly trust={trust} assetBase={assetBase} variant="preview" />
}
