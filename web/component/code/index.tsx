import './index.less'
import ace from 'ace-builds'

import { useEffect, useRef } from 'react'
import { MarkDown } from '../markdown'

ace.config.set('basePath', 'https://cdn.jsdelivr.net/npm/ace-builds/src-min-noconflict')

interface CodeProps {
  value?: string
  theme?: string
  language?: string
  static?: boolean
  onChange?: (value: string) => void
  options?: Partial<ace.Ace.EditorOptions>
}

const LAN_SET = [
  'c_cpp',
  'csharp',
  'css',
  'erlang',
  'golang',
  'haskell',
  'html',
  'java',
  'javascript',
  'json',
  'kotlin',
  'latex',
  'lua',
  'markdown',
  'objectivec',
  'pascal',
  'perl',
  'php',
  'python',
  'ruby',
  'rust',
  'sql',
  'svg',
  'swift',
  'xml',
  'yaml'
]

const LAN_MAP: Record<string, string> = {
  c: 'c_cpp',
  cpp: 'c_cpp',
  cs: 'csharp',
  erl: 'erlang',
  go: 'golang',
  hs: 'haskell',
  js: 'javascript',
  kt: 'kotlin',
  md: 'markdown',
  m: 'objectivec',
  pas: 'pascal',
  pl: 'perl',
  py: 'python',
  rb: 'ruby',
  rs: 'rust'
}

function language2mode(lan: string): string {
  if (LAN_SET.includes(lan)) return lan
  if (lan in LAN_MAP) return LAN_MAP[lan]
  return 'text'
}

function CodeEditor(props: CodeProps) {
  const div = useRef<HTMLDivElement>(null)
  const editor = useRef<ace.Ace.Editor>()
  const { value, language, theme, options } = props

  useEffect(() => {
    if (!div.current) return
    const e = editor.current = ace.edit(div.current, options)
    e.on('change', () => props.onChange?.(e.getValue()))
    return () => e.destroy()
  }, [])

  useEffect(() => {
    if (value !== editor.current?.getValue())
      editor.current?.setValue(value ?? '', -1)
  }, [value])
  useEffect(() => {
    const mode = language2mode(language ?? '')
    mode && editor.current?.getSession().setMode(`ace/mode/${mode}`)
  }, [language])
  useEffect(() => {
    theme && editor.current?.setTheme(`ace/theme/${theme}`)
  }, [theme])

  return <div ref={div} className="code-editor"></div>
}

export function Code(props: CodeProps) {
  const { value = '', language = '' } = props
  if (!props.static) return CodeEditor(props)
  return <MarkDown children={`~~~${language}\n${value}\n~~~`} />
}
