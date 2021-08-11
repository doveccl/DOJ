import React from 'react'
import ReactMarkdown from 'react-markdown'
import katex from 'rehype-katex'
import math from 'remark-math'
import gfm from 'remark-gfm'

import { Code } from '../code'
import { renderToString } from 'katex'

import './index.less'

const shortCodeRegExp = /\[\[\s*\w+(\s+[^\]]+=[^\]]+)+\s*\]\]/g

const renderMath = (tex: string, displayMode = false) => {
	try {
		const html = renderToString(tex, { displayMode })
		return <span dangerouslySetInnerHTML={{ __html: html }} />
	} catch {
		return <span style={{ color: 'red' }}>error</span>
	}
}

type MarkdownProps = {
	children: string
	shortCode?: boolean
	allowDangerousHtml?: boolean
}

export class MarkDown extends React.Component<MarkdownProps> {
	public shouldComponentUpdate(nextPorps: MarkdownProps) {
		return nextPorps.children !== this.props.children
	}
	public render() {
		const { allowDangerousHtml, shortCode, children } = this.props
		const source = (children || '').replace(/[\s\n]*\n(#+)/g, '\n\n$1')
		return <ReactMarkdown
			// @ts-ignore
			rehypePlugins={[katex]}
			// @ts-ignore
			remarkPlugins={[gfm, math]}
			className="markdown-body"
			skipHtml={!allowDangerousHtml}
			children={shortCode ? source.replace(shortCodeRegExp, code => {
				const arrs = code.slice(2, -2).trim().split(/\s+/)
				const type = arrs.shift().toLowerCase()
				const attrs = {} as any
				arrs.forEach(attr => {
					const [k, v] = attr.split('=')
					attrs[k.trim()] = eval(v.trim())
				})
				if (attrs.id) attrs.url = `/api/file/${attrs.id}`
				switch (type) {
					case 'pdf': return `<object class="pdf" data=${attrs.url} />`
					case 'img': return `<img src=${attrs.url} />`
					default: return `<s>Unknown Tag: ${type}</s>`
				}
			}) : source}
			components={{
				code({inline, className, children}) {
					if (inline || !className) return <code>{children}</code>
					const lan = /language-(\w+)/.exec(className)[1]
					return <Code static language={lan} value={String(children)} />
				}
			}}
		/>
	}
}
