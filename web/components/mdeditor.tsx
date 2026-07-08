import { MdEditor, MdPreview } from 'md-editor-rt'
import { useId } from 'react'

import { uploadImage } from '../client'
import { useColor } from '../color'
import { localeCode, useLocale } from '../locale'
import { rewriteAssetURL } from '../utils/markdown'
import { configureMarkdownRuntime } from './markdown-runtime'

type MarkdownEditorProps = {
  id?: string
  value?: string
  readOnly?: boolean
  upload?: (file: File) => Promise<string>
  variant?: 'editor' | 'preview'
  onChange?: (value: string) => void
}

const toolbars = [
  'bold',
  'italic',
  'strikeThrough',
  '-',
  'title',
  'quote',
  'unorderedList',
  'orderedList',
  'task',
  '-',
  'codeRow',
  'code',
  'link',
  'image',
  'table',
  'katex',
  '=',
  'preview',
  'previewOnly',
  'fullscreen'
] as const

configureMarkdownRuntime()

export function MarkdownEditor({
  id,
  value = '',
  readOnly = false,
  upload: uploadFile = uploadImage,
  variant = 'editor',
  onChange
}: MarkdownEditorProps) {
  const { color } = useColor()
  const { lang } = useLocale()
  const generatedID = `md-${useId().replace(/[^a-zA-Z0-9_-]/g, '')}`
  const editorID = id ?? generatedID
  const language = localeCode(lang)
  const theme = color === 'dark' ? 'dark' : 'light'
  const common = {
    id: editorID,
    value,
    theme,
    language,
    previewTheme: 'github',
    codeTheme: 'github',
    showCodeRowNumber: false,
    transformImgUrl: (url: string) => rewriteAssetURL(url, editorID),
    noMermaid: true,
    noEcharts: true,
    noImgZoomIn: true,
    codeFoldable: false
  } as const

  if (readOnly || variant === 'preview') {
    return (
      <div className="markdownPreviewShell">
        <MdPreview {...common} />
      </div>
    )
  }

  return (
    <div className="markdownShell">
      <MdEditor
        {...common}
        onChange={(next) => onChange?.(next)}
        onUploadImg={(files, callback) => {
          void uploadFiles(files, uploadFile).then(callback)
        }}
        toolbars={[...toolbars]}
        noPrettier
        style={{ height: 500 }}
      />
    </div>
  )
}

async function uploadFiles(files: File[], upload: (file: File) => Promise<string>) {
  const urls = await Promise.all(files.map((file) => upload(file)))
  return urls.map((url, index) => ({
    url,
    alt: files[index]?.name ?? '',
    title: files[index]?.name ?? ''
  }))
}
