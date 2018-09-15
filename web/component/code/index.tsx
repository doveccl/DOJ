import * as ace from 'ace-builds'
import * as React from 'react'
import * as highlight from '../../util/highlight'

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

function escapeHtml(unsafe: string) {
	if (!unsafe) { return '' }
	return unsafe
		.replace(/&/g, '&amp;')
		.replace(/</g, '&lt;')
		.replace(/>/g, '&gt;')
		.replace(/"/g, '&quot;')
		.replace(/'/g, '&apos;')
}

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

const language2mode = (mode: string) => {
	if (mode in LAN_SET) { return mode }
	if (mode in LAN_MAP) { return LAN_MAP[mode] }
	return false
}

export default class extends React.Component<CodeProps> {
	private prevLanguage = undefined as string
	private prevTheme = undefined as string
	private editor = undefined as ace.Ace.Editor
	private viewer = undefined as HTMLElement
	private getOptions = (props?: CodeProps) => {
		const { language, theme, value } = props || this.props
		const options = this.props.options || {}
		const mode = language2mode(language)
		if (value) { options.value = value }
		if (mode) { options.mode = `ace/mode/${mode}` }
		if (theme) { options.theme = `ace/theme/${theme}` }
		return options
	}
	private refEditor = (code: HTMLElement) => {
		if (!code) { return }
		const { onChange } = this.props
		const editor = ace.edit(code, this.getOptions())
		editor.on('change', () => onChange && onChange(editor.getValue()))
		this.editor = editor
	}
	private refViewer = (code: HTMLElement) => {
		if (!code) { return }
		code.innerHTML = escapeHtml(this.props.value)
		highlight(code, this.getOptions())
		this.viewer = code
	}
	public componentWillReceiveProps(nextProps: CodeProps) {
		if (this.editor) {
			const { language, theme } = nextProps
			if (theme && this.prevTheme !== theme) {
				this.prevTheme = theme
				this.editor.setTheme(`ace/theme/${theme}`)
			}
			if (language && this.prevLanguage !== language) {
				this.prevLanguage = language
				const mode = language2mode(language)
				this.editor.session.setMode(`ace/mode/${mode}`)
			}
		} else if (this.viewer) {
			const { language, theme } = nextProps
			if (
				(theme && this.prevTheme !== theme) ||
				(language && this.prevLanguage !== language)
			) {
				this.prevTheme = theme
				this.prevLanguage = language
				this.viewer.innerHTML = escapeHtml(nextProps.value)
				highlight(this.viewer, this.getOptions(nextProps))
			}
		}
	}
	public render() {
		return this.props.static ?
			<pre ref={this.refViewer} className="viewer" /> :
			<div ref={this.refEditor} className="editor" />
	}
}