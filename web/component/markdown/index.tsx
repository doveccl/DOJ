import React from 'react'
import hljs from 'highlight.js'
import { marked } from 'marked'
import { message } from 'antd'
import { sanitize } from 'dompurify'
import { renderToString } from 'katex'

import 'highlightjs-line-numbers.js'
import './index.less'

const mathRegExp = /(\${1,2})([\s\S]+?)\1/g
const codeRegExp = /\s*([`~]{3})(\S*)\s+?([\s\S]+?)\n\1/g
const shortCodeRegExp = /\[\[\s*(\w+)(?:\s+(\w+)="(\S+)")+\s*\]\]/g

type MarkdownProps = {
	children: string
	trusted?: boolean
}

export class MarkDown extends React.Component<MarkdownProps> {
	private codes: string[]
	private ref = React.createRef<HTMLDivElement>()
	public shouldComponentUpdate(nextPorps: MarkdownProps) {
		return nextPorps.children !== this.props.children
	}
	private copy(ev: MouseEvent) {
		const el = ev.currentTarget as HTMLElement
		navigator.clipboard.writeText(el.dataset.code)
		message.success('Copied')
	}
	private highlight() {
		this.ref.current.querySelectorAll<HTMLElement>('pre>code').forEach(el => {
			el.dataset.code = el.textContent = this.codes.shift()
			if (!el.className) el.className = 'language-plaintext'
			hljs.highlightElement(el)
			// @ts-ignore highlightjs-line-numbers.js has no type define
			hljs.lineNumbersBlock(el)
			el.onclick = this.copy
		})
	}
	public componentDidMount() {
		this.highlight()
	}
	public componentDidUpdate() {
		this.highlight()
	}
	public render() {
		const { trusted, children } = this.props
		let result = children || ''; this.codes = []
		// replace code blocks with empty string
		// avoid sanitize: '#include<xxx>' => '#include'
		result = result.replace(codeRegExp, (_, tok, lan, code) => {
			this.codes.push(code)
			return `\n${tok}${lan}\n${tok}`
		})
		// ensure an empty line before title
		result = result.replace(/\s*\n(#{1,6})/g, '\n\n$1').trimLeft()
		// trusted ? parse shortcode : sanitize it
		result = trusted ? result.replace(shortCodeRegExp, (_, tag, ...args) => {
			let id = '', src = ''
			for (let i = 1; i < args.length; i += 2) {
				switch (args[i - 1]) {
					case 'id': id = args[i]; break
					case 'src': src = args[i]; break
				}
			}
			if (id) src = `/api/file/${id}`
			switch (src && tag.toLowerCase()) {
				case 'pdf': return `<object class="pdf" data=${src} />`
				case 'img': return `<img src=${src} />`
				default: return `<red>[${tag}]</red>`
			}
		}) : sanitize(result, { ALLOWED_TAGS: [] })
		// parse math block $...$, $$...$$ with katex
		result = result.replace(mathRegExp, (_, tok, math) => {
			const displayMode = tok === '$$'
			const tag = displayMode ? 'div' : 'span'
			try {
				return renderToString(math, { displayMode })
			} catch {
				const el = document.createElement(tag)
				el.style.color = 'red'
				el.textContent = math
				return el.outerHTML
			}
		})
		return <div
			ref={this.ref}
			className="markdown-body"
			dangerouslySetInnerHTML={{ __html: marked(result) }}
		/>
	}
}
