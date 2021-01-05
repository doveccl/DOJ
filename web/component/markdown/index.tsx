import React from 'react'
import Markdown from 'react-markdown'
import math from 'remark-math'
import gfm from 'remark-gfm'

import { Code } from '../code'
import { renderToString } from 'katex'

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
		return <Markdown
			plugins={[gfm, math]}
			allowDangerousHtml={allowDangerousHtml}
			children={shortCode ? children.replace(shortCodeRegExp, code => {
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
			}) : children}
			renderers={{
				math: ({ value }) => renderMath(value, true),
				inlineMath: ({ value }) => renderMath(value),
				code: ({ value, language }) => <Code
					static value={value} language={language}
				/>
			}}
		/>
	}
}
