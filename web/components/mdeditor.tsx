import { MdEditor, MdPreview } from 'md-editor-rt'
import { useEffect, useId } from 'react'

import { uploadImage } from '../client'
import { useColor } from '../color'
import { useLocale } from '../locale'
import { clearMarkdownAssetBase, rewriteAssetURL, sanitizerForTrust, setMarkdownAssetBase } from '../utils/markdown'
import type { MarkdownTrust } from '../utils/markdown'
import { configureMarkdownRuntime } from './markdown-runtime'

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
  value = '',
  minHeight = 260,
  readOnly = false,
  trust = 'ugc',
  assetBase,
  upload: uploadFile = uploadImage,
  variant = 'editor',
  onChange
}: MarkdownEditorProps) {
  const { color } = useColor()
  const { lang } = useLocale()
  const editorID = `md-${useId().replace(/[^a-zA-Z0-9_-]/g, '')}`
  setMarkdownAssetBase(editorID, assetBase)
  useEffect(() => () => clearMarkdownAssetBase(editorID), [editorID])
  const language = lang === 'zh' ? 'zh-CN' : 'en-US'
  const theme = color === 'dark' ? 'dark' : 'light'
  const common = {
    id: editorID,
    value,
    theme,
    language,
    previewTheme: 'github',
    codeTheme: 'github',
    sanitize: sanitizerForTrust(trust),
    transformImgUrl: (url: string) => rewriteAssetURL(url, assetBase),
    noMermaid: true,
    noEcharts: true,
    noImgZoomIn: true
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
        placeholder=""
        onChange={(next) => onChange?.(next)}
        onUploadImg={(files, callback) => {
          void uploadFiles(files, uploadFile).then(callback)
        }}
        toolbars={[...toolbars]}
        footers={[]}
        noPrettier
        autoFoldThreshold={80}
        style={{ height: minHeight }}
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
