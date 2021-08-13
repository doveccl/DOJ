import React from 'react'
import marked from 'marked'
import hljs from 'highlight.js'
import { sanitize } from 'dompurify'
import { renderToString } from 'katex'

import './index.less'

const shortCodeRegExp = /\[\[\s*\w+(\s+[^\]]+=[^\]]+)+\s*\]\]/g

type MarkdownProps = {
	children: string
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
		// replace code blocks with simple string
		// avoid sanitize: '#include<xxx>' => '#include'
		marked(result, {
			silent: true,
			highlight(code) {
				codes.push(code)
				result = result.replace(code, 'replacer')
			}
		})
		// trusted ? parse shortcode : sanitize it
		result = trusted ? result.replace(shortCodeRegExp, code => {
			const arrs = code.slice(2, -2).trim().split(/\s+/)
			const type = arrs.shift().toLowerCase()
			const attrs = {} as any
			arrs.forEach(attr => {
				const [k, v] = attr.split('=')
				attrs[k.trim()] = v.trim().replace(/"/g, '')
			})
			if (attrs.id) attrs.src = `/api/file/${attrs.id}`
			switch (type) {
				case 'pdf': return `<object class="pdf" data=${attrs.src} />`
				case 'img': return `<img src=${attrs.src} />`
				default: return `<s>Unknown Tag: ${type}</s>`
			}
		}) : sanitize(result, { ALLOWED_TAGS: [] })
		// parse math block $...$, $$...$$ with katex
		result = result.replace(/\${1,2}[\s\S]+?\${1,2}/g, str => {
			const displayMode = str.startsWith('$$')
			const math = str.replace(/^\$+|\$+$/g, '')
			try {
				return renderToString(math, { displayMode })
			} catch {
				const el = document.createElement(displayMode ? 'div' : 'span')
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
					highlight(_, lan) {
						// restore replaced code
						const code = codes.shift()
						const lans = lan ? [lan] : undefined
						return hljs.highlightAuto(code, lans).value
					}
				})
			}}
		/>
	}
}
