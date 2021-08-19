import React from 'react'
import ace from 'ace-builds'
import { MarkDown } from '../markdown'

import './index.less'

interface CodeProps {
	value?: string
	theme?: string
	language?: string
	static?: boolean
	onChange?: (value: string) => any
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

const LAN_MAP: any = {
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

const language2mode = (lan: string) => {
	if (LAN_SET.includes(lan)) return lan
	if (lan in LAN_MAP) return LAN_MAP[lan]
	return false
}

export class Code extends React.Component<CodeProps> {
	private editor: ace.Ace.Editor
	private refEditor = React.createRef<HTMLDivElement>()
	private getOptions = (props?: CodeProps) => {
		const { language, theme, value } = props || this.props
		const options = this.props.options || {}
		const mode = language2mode(language)
		if (value) options.value = value
		if (mode) options.mode = `ace/mode/${mode}`
		if (theme) options.theme = `ace/theme/${theme}`
		return options
	}
	private destroy() {
		if (this.editor) {
			this.editor.destroy()
			this.editor = null
		}
	}
	private update() {
		if (!this.editor) {
			const code = this.refEditor.current
			const cb = this.props.onChange || (() => {})
			this.editor = ace.edit(code, this.getOptions(this.props))
			this.editor.on('change', () => cb(this.editor.getValue()))
		} else { // change editor language & theme
			const { mode, theme } = this.getOptions(this.props)
			if (this.editor.getTheme() !== theme) this.editor.setTheme(theme)
			this.editor.getSession().setMode(mode)
		}
	}

	public componentDidMount() {
		if (!this.props.static) this.update()
	}
	public shouldComponentUpdate(nextProps?: CodeProps) {
		for (const k of Object.keys(nextProps)) {
			if (k === 'onChange') continue
			if (nextProps[k] !== this.props[k]) {
				nextProps.static && this.destroy()
				return true
			}
		}
		return false
	}
	public componentDidUpdate() {
		if (!this.props.static) this.update()
	}
	public componentWillUnmount() {
		this.destroy()
	}
	public render() {
		const code = this.props.value
		const lan = this.props.language || ''
		return this.props.static ?
			<MarkDown>{`~~~${lan}\n${code}\n~~~`}</MarkDown> :
			<div ref={this.refEditor} className="code-editor" />
	}
}
