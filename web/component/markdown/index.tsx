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
		const { trusted, children } = this.props
		let result = (trusted ? children?.replace(shortCodeRegExp, code => {
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
		}) : sanitize(children, {
			ALLOWED_TAGS: []
		}))?.replace(/\${1,2}[\s\S]+?\${1,2}/g, str => {
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
					highlight(code, lang) {
						if (!lang) return code
						return hljs.highlightAuto(code).value
					}
				})
			}}
		/>
	}
}
