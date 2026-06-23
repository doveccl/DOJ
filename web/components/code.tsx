import { EditorView } from '@codemirror/view'
import { githubDark, githubLight } from '@uiw/codemirror-theme-github'
import CodeMirror from '@uiw/react-codemirror'
import { languages } from '@codemirror/language-data'
import { useEffect, useMemo, useState } from 'react'
import type { ComponentProps } from 'react'

import { useColor } from '../color'

type CodeEditorProps = {
  value: string
  language?: string
  minHeight?: string
  readOnly?: boolean
  onChange?: (value: string) => void
}

type CodeMirrorExtensions = NonNullable<ComponentProps<typeof CodeMirror>['extensions']>

export function CodeEditor({ value, language, minHeight = '320px', readOnly = false, onChange }: CodeEditorProps) {
  const { color } = useColor()
  const [langExtensions, setLangExtensions] = useState<CodeMirrorExtensions>([])
  useEffect(() => {
    let alive = true
    void loadLanguageExtension(language)
      .catch(() => [])
      .then((extensions) => {
        if (alive) {
          setLangExtensions(extensions)
        }
      })
    return () => {
      alive = false
    }
  }, [language])
  const extensions = useMemo(() => [...langExtensions, EditorView.lineWrapping], [langExtensions])

  return (
    <div className="codeEditor">
      <CodeMirror
        value={value}
        minHeight={minHeight}
        theme={color === 'dark' ? githubDark : githubLight}
        extensions={extensions}
        readOnly={readOnly}
        editable={!readOnly}
        basicSetup={{
          lineNumbers: true,
          foldGutter: true,
          dropCursor: true,
          allowMultipleSelections: true,
          indentOnInput: true,
          bracketMatching: true,
          closeBrackets: true,
          autocompletion: true,
          rectangularSelection: true,
          highlightSelectionMatches: true
        }}
        onChange={(next) => onChange?.(next)}
      />
    </div>
  )
}

async function loadLanguageExtension(language = ''): Promise<CodeMirrorExtensions> {
  const key = language.toLowerCase()
  if (key === 'plain' || key === 'plaintext' || key === 'text') {
    return []
  }
  const candidates = languageCandidates(key)
  const description = languages.find((item) => {
    const name = item.name.toLowerCase()
    if (candidates.includes(name)) {
      return true
    }
    if (item.alias.some((alias) => candidates.includes(alias.toLowerCase()))) {
      return true
    }
    if (item.extensions.some((extension) => candidates.includes(extension.toLowerCase()))) {
      return true
    }
    return candidates.some((candidate) => item.filename?.test(candidate))
  })
  if (description) {
    return [await description.load()]
  }
  return []
}

function languageCandidates(value: string) {
  const parts = value
    .split(/[\s,;|]+/)
    .map((part) => part.trim().toLowerCase())
    .filter(Boolean)
  const candidates = new Set(parts)
  for (const part of parts) {
    const file = part.split('/').pop() ?? part
    candidates.add(file)
    const dot = file.lastIndexOf('.')
    if (dot >= 0 && dot < file.length - 1) {
      candidates.add(file.slice(dot + 1))
    }
  }
  return [...candidates]
}
