import { lazy, Suspense } from 'react'

import './markdown.css'

type MarkdownEditorProps = {
  id?: string
  value?: string
  height?: number
  readOnly?: boolean
  upload?: (file: File) => Promise<string>
  variant?: 'editor' | 'preview'
  onChange?: (value: string) => void
}

const LazyMarkdownEditor = lazy(() => import('./mdeditor').then((mod) => ({ default: mod.MarkdownEditor })))

export function MarkdownEditor(props: MarkdownEditorProps) {
  return (
    <Suspense fallback={<div className="markdownShell markdownFallback" />}>
      <LazyMarkdownEditor {...props} />
    </Suspense>
  )
}

export function MarkdownPreview({ id, value }: { id?: string; value: string }) {
  if (!value.trim()) {
    return null
  }

  return <MarkdownEditor id={id} value={value} readOnly variant="preview" />
}
