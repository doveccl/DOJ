import { MdEditor, MdPreview } from 'md-editor-rt'
import type { ComponentProps } from 'react'
import { useEffect, useId, useMemo } from 'react'

import { api, apiData, uploadImage } from '../client'
import { useColor } from '../color'
import { localeCode, useLocale } from '../locale'
import { rewriteAssetURL } from '../utils/markdown'
import { configureMarkdownRuntime } from './markdown-runtime'

type MarkdownEditorProps = {
  id?: string
  value?: string
  height?: number
  readOnly?: boolean
  upload?: (file: File) => Promise<string>
  variant?: 'editor' | 'preview'
  onChange?: (value: string) => void
  mentions?: string[]
  draftKey?: string
}

type CompletionSource = NonNullable<ComponentProps<typeof MdEditor>['completions']>[number]

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
  height = 500,
  readOnly = false,
  upload: uploadFile = uploadImage,
  variant = 'editor',
  onChange,
  mentions,
  draftKey
}: MarkdownEditorProps) {
  const { color } = useColor()
  const { lang } = useLocale()
  const generatedID = `md-${useId().replace(/[^a-zA-Z0-9_-]/g, '')}`
  const editorID = mentions === undefined ? id ?? generatedID : `discussion-${generatedID}`
  const language = localeCode(lang)
  const theme = color === 'dark' ? 'dark' : 'light'
  const mentionKey = mentions?.join('\0')
  const completions = useMemo(() => {
    if (mentionKey === undefined) {
      return undefined
    }
    return [mentionCompletion(mentionKey ? mentionKey.split('\0') : [])]
  }, [mentionKey])
  useEffect(() => {
    if (!draftKey) {
      return
    }
    const draft = window.localStorage.getItem(draftKey)
    if (draft !== null && draft !== value) {
      onChange?.(draft)
    }
  }, [draftKey])
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
        onChange={(next) => {
          if (draftKey) {
            if (next) {
              window.localStorage.setItem(draftKey, next)
            } else {
              window.localStorage.removeItem(draftKey)
            }
          }
          onChange?.(next)
        }}
        completions={completions}
        onUploadImg={(files, callback) => {
          void uploadFiles(files, uploadFile).then(callback)
        }}
        toolbars={[...toolbars]}
        noPrettier
        style={{ height }}
      />
    </div>
  )
}

function mentionCompletion(localNames: string[]): CompletionSource {
  return async (context) => {
    const word = context.matchBefore(/@[A-Za-z0-9_-]*/)
    if (!word || word.from > 0 && /[A-Za-z0-9_@-]/.test(context.state.sliceDoc(word.from - 1, word.from))) {
      return null
    }
    const query = word.text.slice(1).toLowerCase()
    const names: string[] = []
    const seen = new Set<string>()
    const add = (name: string) => {
      const key = name.toLowerCase()
      if (!seen.has(key) && (!query || key.includes(query))) {
        seen.add(key)
        names.push(name)
      }
    }
    localNames.forEach(add)
    if (query) {
      const controller = new AbortController()
      context.addEventListener('abort', () => controller.abort(), { onDocChange: true })
      try {
        const users = await apiData(api.GET('/api/users', {
          params: { query: { q: query } },
          signal: controller.signal
        }))
        if (context.aborted) {
          return null
        }
        users.forEach((user) => add(user.name))
      } catch {
        if (context.aborted) {
          return null
        }
      }
    }
    if (names.length === 0) {
      return null
    }
    return {
      from: word.from,
      options: names.slice(0, 20).map((name) => ({ label: `@${name}`, apply: `@${name} `, type: 'text' }))
    }
  }
}

async function uploadFiles(files: File[], upload: (file: File) => Promise<string>) {
  const urls = await Promise.all(files.map((file) => upload(file)))
  return urls.map((url, index) => ({
    url,
    alt: files[index]?.name ?? '',
    title: files[index]?.name ?? ''
  }))
}
