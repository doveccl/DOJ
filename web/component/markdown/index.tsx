import React from 'react'
import marked from 'marked'
import hljs from 'highlight.js'
import { sanitize } from 'dompurify'
import { renderToString } from 'katex'

import './index.less'

const mathRegExp = /(\${1,2})([\s\S]+?)\1/g
const codeRegExp = /\s*([`~]{3})(\S*)\s+([\s\S]+?)\n\1/g
const shortCodeRegExp = /\[\[\s*(\w+)(?:\s+(\w+)="(\S+)")+\s*\]\]/g

type MarkdownProps = {
	children: string
	autohljs?: boolean
	trusted?: boolean
}

export class MarkDown extends React.Component<MarkdownProps> {
	public shouldComponentUpdate(nextPorps: MarkdownProps) {
		return nextPorps.children !== this.props.children
	}
	public render() {
		const codes: string[] = []
		const { trusted, children } = this.props
		let result = children || ''
		// replace code blocks with empty string
		// avoid sanitize: '#include<xxx>' => '#include'
		result = result.replace(codeRegExp, (_, tok, lan, code) => {
			codes.push(code)
			return `\n${tok}${lan}\n${tok}`
		})
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
			className="markdown-body"
			dangerouslySetInnerHTML={{
				__html: marked(result, {
					silent: true,
					highlight: (_, lan) => {
						// restore replaced code
						let code = codes.shift(), lans = [lan]
						if (!lan && this.props.autohljs) lans = undefined
						return hljs.highlightAuto(code, lans).value
					}
				})
			}}
		/>
	}
}
