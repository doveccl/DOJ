import ace from 'ace-builds'
import React from 'react'

const highlight = ace.require('ace/ext/static_highlight')

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
	if (LAN_SET.includes(lan)) { return lan }
	if (lan in LAN_MAP) { return LAN_MAP[lan] }
	return false
}

export class Code extends React.Component<CodeProps> {
	private refViewer = React.createRef<HTMLDivElement>()
	private refEditor = React.createRef<HTMLDivElement>()
	private getOptions = (props?: CodeProps) => {
		const { language, theme, value } = props || this.props
		const options = this.props.options || {}
		const mode = language2mode(language)
		if (value) { options.value = value }
		if (mode) { options.mode = `ace/mode/${mode}` }
		if (theme) { options.theme = `ace/theme/${theme}` }
		return options
	}
	private update(props: CodeProps) {
		if (props.static) {
			const code = this.refViewer.current
			code.textContent = props.value
			highlight(code, this.getOptions(props))
		} else {
			const { onChange } = props
			const code = this.refEditor.current
			const editor = ace.edit(code, this.getOptions(props))
			editor.on('change', () => onChange && onChange(editor.getValue()))
		}
	}
	public shouldComponentUpdate(nextProps?: CodeProps) {
		let flag = false
		Object.keys(nextProps).forEach(key => {
			if (nextProps[key] != this.props[key]) {
				this.update(nextProps)
				flag = true
			}
		})
		return flag
	}
	public componentDidMount() {
		this.update(this.props)
	}
	public render() {
		return this.props.static ?
			<div ref={this.refViewer} className="code-viewer" /> :
			<div ref={this.refEditor} className="code-editor" />
	}
}
